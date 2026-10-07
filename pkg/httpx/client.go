// Package httpx provides a Confluence-aware HTTP client with pluggable authentication,
// retry logic, rate limit handling, and debug logging.
package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	confluencecfg "github.com/atlassian/confluence-cli/internal/config"
	"github.com/atlassian/confluence-cli/internal/secret"
	"github.com/atlassian/confluence-cli/pkg/auth"
)

const (
	defaultUserAgent   = "confluence-cli/dev"
	defaultTimeout     = 30 * time.Second
	defaultMaxAttempts = 3
	defaultInitBackoff = 500 * time.Millisecond
	defaultMaxBackoff  = 10 * time.Second
)

// Client is a Confluence-aware HTTP client.
type Client struct {
	baseURL   *url.URL
	http      *http.Client
	userAgent string
	debug     bool

	retry struct {
		MaxAttempts    int
		InitialBackoff time.Duration
		MaxBackoff     time.Duration
	}

	rateMu sync.RWMutex
	rate   RateLimit
}

// RateLimit captures the last observed X-RateLimit-* headers.
type RateLimit struct {
	Limit     int
	Remaining int
	Reset     time.Time
}

// New creates a Client configured for the given host.
func New(host *confluencecfg.Host) (*Client, error) {
	if host == nil {
		return nil, fmt.Errorf("host configuration is required")
	}

	rawURL := strings.TrimRight(host.BaseURL, "/")
	if rawURL == "" {
		return nil, fmt.Errorf("host base URL is empty")
	}

	// For Confluence Cloud, if base URL does not have /wiki, append it
	if (host.Kind == "cloud" || strings.Contains(rawURL, ".atlassian.net")) && !strings.HasSuffix(rawURL, "/wiki") {
		rawURL = rawURL + "/wiki"
	}

	baseURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse base URL %q: %w", rawURL, err)
	}

	transport, err := buildTransport(host)
	if err != nil {
		return nil, err
	}

	c := &Client{
		baseURL:   baseURL,
		userAgent: defaultUserAgent,
		debug:     os.Getenv("CONFLUENCE_HTTP_DEBUG") == "1",
		http: &http.Client{
			Transport: transport,
			Timeout:   defaultTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
	c.retry.MaxAttempts = defaultMaxAttempts
	c.retry.InitialBackoff = defaultInitBackoff
	c.retry.MaxBackoff = defaultMaxBackoff
	return c, nil
}

func buildTransport(host *confluencecfg.Host) (http.RoundTripper, error) {
	base := http.DefaultTransport

	switch host.AuthMethod {
	case confluencecfg.AuthMethodKerberos:
		t, err := auth.NewKerberosTransport(base)
		if err != nil {
			return nil, fmt.Errorf("kerberos transport: %w", err)
		}
		return t, nil

	case confluencecfg.AuthMethodSessionCookie:
		hostKey := host.BaseURL
		getCookie := func() (string, error) {
			if v := secret.EnvSessionCookie(); v != "" {
				return stripCookieName(v), nil
			}
			val, err := secret.Get(secret.SessionKey(hostKey))
			if err != nil {
				return "", auth.ErrSessionNotFound
			}
			return val, nil
		}
		return auth.NewSessionTransport(base, host.BaseURL, getCookie), nil

	case confluencecfg.AuthMethodBearer:
		token := resolveToken(host)
		return &bearerTransport{inner: base, token: token}, nil

	case confluencecfg.AuthMethodBasicFallback:
		token := resolveToken(host)
		user := host.Username
		if user == "" {
			user = host.Email
		}
		return &basicFallbackTransport{inner: base, username: user, token: token}, nil

	default:
		token := resolveToken(host)
		user := host.Username
		if user == "" {
			user = host.Email
		}
		return &basicTransport{inner: base, username: user, token: token}, nil
	}
}

func resolveToken(host *confluencecfg.Host) string {
	if v := secret.EnvToken(); v != "" {
		return v
	}
	if host.Token != "" {
		return host.Token
	}
	if tok, err := secret.Get(secret.TokenKey(host.BaseURL)); err == nil && tok != "" {
		return tok
	}
	return ""
}

type basicTransport struct {
	inner    http.RoundTripper
	username string
	token    string
}

func (t *basicTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	if t.username != "" || t.token != "" {
		clone.SetBasicAuth(t.username, t.token)
	}
	return t.inner.RoundTrip(clone)
}

type bearerTransport struct {
	inner http.RoundTripper
	token string
}

func (t *bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	if t.token != "" {
		clone.Header.Set("Authorization", "Bearer "+t.token)
	}
	return t.inner.RoundTrip(clone)
}

type basicFallbackTransport struct {
	inner    http.RoundTripper
	username string
	token    string
}

func (t *basicFallbackTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	if t.username != "" || t.token != "" {
		clone.SetBasicAuth(t.username, t.token)
	}
	q := clone.URL.Query()
	q.Set("os_authType", "basic")
	clone.URL.RawQuery = q.Encode()
	return t.inner.RoundTrip(clone)
}

// Do executes a request with retry logic.
func (c *Client) Do(req *http.Request, v any) (*http.Response, error) {
	var resp *http.Response
	var err error

	for attempt := 0; attempt < c.retry.MaxAttempts; attempt++ {
		if attempt > 0 {
			delay := c.retryDelay(attempt, resp)
			timer := time.NewTimer(delay)
			select {
			case <-req.Context().Done():
				timer.Stop()
				return nil, req.Context().Err()
			case <-timer.C:
			}
		}

		resp, err = c.http.Do(req)
		if err != nil {
			if !isRetryable(attempt, c.retry.MaxAttempts, req.Method, 0) {
				return nil, err
			}
			continue
		}

		if c.debug {
			fmt.Fprintf(os.Stderr, "confluence: %s %s → %d\n", req.Method, req.URL, resp.StatusCode)
		}

		if resp.StatusCode == http.StatusTooManyRequests || (resp.StatusCode >= 500 && isIdempotent(req.Method)) {
			if !isRetryable(attempt, c.retry.MaxAttempts, req.Method, resp.StatusCode) {
				break
			}
			resp.Body.Close()
			continue
		}
		break
	}

	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var apiErr struct {
			Message   string   `json:"message"`
			Messages  []string `json:"messages"`
			Reason    string   `json:"reason"`
			ErrorText string   `json:"error"`
		}
		if jsonErr := json.Unmarshal(body, &apiErr); jsonErr == nil {
			msg := apiErr.Message
			if msg == "" && len(apiErr.Messages) > 0 {
				msg = strings.Join(apiErr.Messages, "; ")
			}
			if msg == "" {
				msg = apiErr.Reason
			}
			if msg == "" {
				msg = apiErr.ErrorText
			}
			if msg != "" {
				return resp, fmt.Errorf("HTTP %d: %s", resp.StatusCode, msg)
			}
		}
		if len(body) > 0 && len(body) < 500 {
			return resp, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
		return resp, fmt.Errorf("HTTP %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	if v != nil && resp.StatusCode < 300 {
		defer resp.Body.Close()
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return resp, fmt.Errorf("read response body: %w", readErr)
		}
		if len(body) > 0 {
			if decErr := json.Unmarshal(body, v); decErr != nil {
				return resp, fmt.Errorf("decode response: %w", decErr)
			}
		}
	}
	return resp, nil
}

// NewRequest constructs an HTTP request against the Confluence base URL.
func (c *Client) NewRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	u := *c.baseURL
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	rel, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("parse path %q: %w", path, err)
	}

	basePath := strings.TrimSuffix(u.Path, "/")
	relPath := rel.Path

	// Avoid duplicate "/wiki" prefix if relPath already has it
	if strings.HasSuffix(basePath, "/wiki") && strings.HasPrefix(relPath, "/wiki/") {
		basePath = strings.TrimSuffix(basePath, "/wiki")
	}

	u.Path = basePath + relPath
	if rel.RawQuery != "" {
		if u.RawQuery != "" {
			u.RawQuery = u.RawQuery + "&" + rel.RawQuery
		} else {
			u.RawQuery = rel.RawQuery
		}
	}

	var buf io.Reader
	if body != nil {
		switch v := body.(type) {
		case io.Reader:
			buf = v
		case []byte:
			buf = bytes.NewReader(v)
		case string:
			buf = strings.NewReader(v)
		default:
			b, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("marshal request body: %w", err)
			}
			buf = bytes.NewReader(b)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		if _, ok := body.(io.Reader); !ok {
			req.Header.Set("Content-Type", "application/json")
		}
	}
	return req, nil
}

// Get is a convenience wrapper for GET requests.
func (c *Client) Get(ctx context.Context, path string, v any) (*http.Response, error) {
	req, err := c.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req, v)
}

// Post is a convenience wrapper for POST requests.
func (c *Client) Post(ctx context.Context, path string, body, v any) (*http.Response, error) {
	req, err := c.NewRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	return c.Do(req, v)
}

// Put is a convenience wrapper for PUT requests.
func (c *Client) Put(ctx context.Context, path string, body, v any) (*http.Response, error) {
	req, err := c.NewRequest(ctx, http.MethodPut, path, body)
	if err != nil {
		return nil, err
	}
	return c.Do(req, v)
}

// Delete is a convenience wrapper for DELETE requests.
func (c *Client) Delete(ctx context.Context, path string) (*http.Response, error) {
	req, err := c.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req, nil)
}

// BaseURL returns the client's base URL string.
func (c *Client) BaseURL() string { return c.baseURL.String() }

func isRetryable(attempt, max int, method string, status int) bool {
	if attempt+1 >= max {
		return false
	}
	if status == http.StatusTooManyRequests {
		return true
	}
	return isIdempotent(method)
}

func isIdempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut, http.MethodDelete:
		return true
	}
	return false
}

func (c *Client) retryDelay(attempt int, resp *http.Response) time.Duration {
	delay := c.retry.InitialBackoff
	if attempt > 1 {
		delay *= time.Duration(1 << (attempt - 1))
	}
	if delay > c.retry.MaxBackoff {
		delay = c.retry.MaxBackoff
	}
	if resp != nil {
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, err := strconv.Atoi(ra); err == nil {
				d := time.Duration(secs) * time.Second
				if d > 60*time.Second {
					d = 60 * time.Second
				}
				if d > 0 {
					delay = d
				}
			}
		}
	}
	return delay
}

func stripCookieName(v string) string {
	const prefix = auth.SessionCookieName + "="
	if strings.HasPrefix(v, prefix) {
		return v[len(prefix):]
	}
	return v
}
