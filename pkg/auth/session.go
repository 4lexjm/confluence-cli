package auth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/atlassian/confluence-cli/pkg/browser"
)

// SessionCookieName is the cookie Confluence sets after a successful login.
const SessionCookieName = "JSESSIONID"

// DefaultSessionTTL is the assumed session lifetime when Confluence does not
// advertise an explicit expiry. Confluence DC instances typically default to 8 hours.
const DefaultSessionTTL = 8 * time.Hour

// SessionExpiredError is returned when the JSESSIONID cookie has expired or
// been invalidated server-side (indicated by a 401 or SSO redirect).
type SessionExpiredError struct {
	Host string
}

func (e *SessionExpiredError) Error() string {
	return fmt.Sprintf(
		"Confluence session expired for %s\n"+
			"Run `confluence auth login %s --browser` to re-authenticate via SSO",
		e.Host, e.Host)
}

// SessionTransport is an http.RoundTripper that authenticates requests by
// attaching a JSESSIONID cookie obtained from a prior browser SSO login.
type SessionTransport struct {
	inner     http.RoundTripper
	host      string
	getCookie func() (string, error)
}

// NewSessionTransport creates a SessionTransport.
func NewSessionTransport(inner http.RoundTripper, host string, getCookie func() (string, error)) *SessionTransport {
	if inner == nil {
		inner = http.DefaultTransport
	}
	return &SessionTransport{inner: inner, host: host, getCookie: getCookie}
}

// RoundTrip implements http.RoundTripper.
func (t *SessionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cookie, err := t.getCookie()
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return nil, &SessionExpiredError{Host: t.host}
		}
		return nil, fmt.Errorf("retrieve session cookie: %w", err)
	}

	clone := req.Clone(req.Context())
	existing := clone.Header.Get("Cookie")
	cookieHeader := SessionCookieName + "=" + cookie
	if existing != "" {
		cookieHeader = existing + "; " + cookieHeader
	}
	clone.Header.Set("Cookie", cookieHeader)

	resp, err := t.inner.RoundTrip(clone)
	if err != nil {
		return nil, err
	}

	if isSessionExpired(resp) {
		resp.Body.Close()
		return nil, &SessionExpiredError{Host: t.host}
	}

	return resp, nil
}

// ErrSessionNotFound is returned by the cookie getter when no session is stored.
var ErrSessionNotFound = errors.New("no Confluence session cookie stored")

func isSessionExpired(resp *http.Response) bool {
	if resp.StatusCode == http.StatusUnauthorized {
		return true
	}
	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusMovedPermanently {
		loc := resp.Header.Get("Location")
		if loc != "" && (strings.Contains(loc, "/login") || strings.Contains(loc, "saml") ||
			strings.Contains(loc, "sso") || strings.Contains(loc, "idp") ||
			strings.Contains(loc, "auth") || strings.Contains(loc, "oauth")) {
			return true
		}
	}
	return false
}

// BrowserLoginResult holds the captured session after a browser SSO flow.
type BrowserLoginResult struct {
	Cookie     string
	CapturedAt time.Time
}

// CaptureSessionViaBrowser starts a local reverse proxy, opens the user's
// browser to the Confluence login page, and captures the JSESSIONID cookie.
func CaptureSessionViaBrowser(ctx context.Context, confluenceBaseURL string, b browser.Browser) (*BrowserLoginResult, error) {
	target, err := url.Parse(confluenceBaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse Confluence base URL: %w", err)
	}

	cookieCh := make(chan string, 1)

	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
		req.Header.Del("X-Forwarded-For")
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		for _, c := range resp.Cookies() {
			if c.Name == SessionCookieName && c.Value != "" {
				select {
				case cookieCh <- c.Value:
				default:
				}
			}
		}
		return nil
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("bind local port: %w", err)
	}
	localAddr := ln.Addr().String()

	srv := &http.Server{
		Handler:      proxy,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	srvErrCh := make(chan error, 1)
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			srvErrCh <- err
		}
	}()

	loginPath := "/login.action"
	if strings.HasSuffix(target.Path, "/wiki") {
		loginPath = "/wiki/login.action"
	}
	loginURL := "http://" + localAddr + loginPath

	fmt.Printf("Opening browser for SSO login...\n")
	fmt.Printf("If the browser does not open automatically, navigate to:\n  %s\n\n", loginURL)
	fmt.Printf("Complete the SSO login in the browser. The CLI will capture your session automatically.\n")

	if err := b.Open(loginURL); err != nil {
		fmt.Printf("Could not open browser automatically: %v\n", err)
	}

	var result *BrowserLoginResult
	select {
	case cookie := <-cookieCh:
		result = &BrowserLoginResult{Cookie: cookie, CapturedAt: time.Now()}
	case err := <-srvErrCh:
		return nil, fmt.Errorf("proxy server error: %w", err)
	case <-ctx.Done():
		return nil, fmt.Errorf("login cancelled: %w", ctx.Err())
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)

	return result, nil
}

// EstimatedExpiry returns the estimated expiry of a session captured at capturedAt.
func EstimatedExpiry(capturedAt time.Time) time.Time {
	return capturedAt.Add(DefaultSessionTTL)
}

// SessionAge returns how long ago the session was captured.
func SessionAge(capturedAt time.Time) time.Duration {
	return time.Since(capturedAt)
}

const SessionWarningThreshold = 30 * time.Minute
