// Package config manages persisted CLI configuration and host credentials for Confluence.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const currentVersion = 1

// AuthMethod enumerates the supported authentication mechanisms.
const (
	AuthMethodBasic         = "basic"
	AuthMethodBearer        = "bearer"
	AuthMethodKerberos      = "kerberos"
	AuthMethodSessionCookie = "session-cookie"
	AuthMethodBasicFallback = "basic-fallback"
)

var (
	// ErrContextNotFound is returned when a requested context is missing.
	ErrContextNotFound = errors.New("context not found")
	// ErrHostNotFound is returned when a requested host entry is missing.
	ErrHostNotFound = errors.New("host not found")
)

// Config models persisted CLI state.
type Config struct {
	Version       int                 `yaml:"version"`
	ActiveContext string              `yaml:"active_context,omitempty"`
	Contexts      map[string]*Context `yaml:"contexts,omitempty"`
	Hosts         map[string]*Host    `yaml:"hosts,omitempty"`

	path string
	mu   sync.RWMutex
}

// Context captures user-scoped defaults bound to a host.
type Context struct {
	Host     string `yaml:"host"`
	SpaceKey string `yaml:"space_key,omitempty"`
}

// Host stores connection and credential metadata for a Confluence instance.
type Host struct {
	Kind         string `yaml:"kind"` // "dc" | "cloud"
	BaseURL      string `yaml:"base_url"`
	Username     string `yaml:"username,omitempty"`
	Email        string `yaml:"email,omitempty"`
	AuthMethod   string `yaml:"auth_method,omitempty"`
	DefaultSpace string `yaml:"default_space,omitempty"`

	// Token is never written to disk. Populated at runtime from OS keychain or env vars.
	Token string `yaml:"-"`

	// SessionCookie holds the JSESSIONID value. Stored in keychain, never on disk.
	SessionCookie string `yaml:"-"`

	// SessionExpiresAt is the observed or estimated expiry of the session cookie.
	SessionExpiresAt time.Time `yaml:"-"`

	AllowInsecureStore bool `yaml:"allow_insecure_store,omitempty"`
}

// IsKerberos reports whether the host uses Kerberos/SPNEGO authentication.
func (h *Host) IsKerberos() bool {
	return h != nil && h.AuthMethod == AuthMethodKerberos
}

// IsSessionCookie reports whether the host uses session-cookie authentication.
func (h *Host) IsSessionCookie() bool {
	return h != nil && h.AuthMethod == AuthMethodSessionCookie
}

// MarshalYAML strips runtime-only fields so credentials are never written to disk.
func (h *Host) MarshalYAML() (any, error) {
	if h == nil {
		return nil, nil
	}
	type alias Host
	safe := alias(*h)
	safe.Token = ""
	safe.SessionCookie = ""
	return safe, nil
}

// Load retrieves configuration from disk or synthesizes headless defaults.
func Load() (*Config, error) {
	path, err := resolvePath()
	if err != nil {
		return nil, err
	}
	cfg := &Config{
		Version:  currentVersion,
		Contexts: make(map[string]*Context),
		Hosts:    make(map[string]*Host),
		path:     path,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg.applyEnvOverrides()
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	if cfg.Contexts == nil {
		cfg.Contexts = make(map[string]*Context)
	}
	if cfg.Hosts == nil {
		cfg.Hosts = make(map[string]*Host)
	}

	cfg.applyEnvOverrides()
	return cfg, nil
}

// applyEnvOverrides injects environment variables for headless/CI operations.
func (c *Config) applyEnvOverrides() {
	envHost := os.Getenv("CONFLUENCE_HOST")
	if envHost == "" {
		return
	}

	cleanHost := strings.TrimRight(envHost, "/")
	kind := "dc"
	if strings.Contains(cleanHost, ".atlassian.net") {
		kind = "cloud"
	}

	authMethod := os.Getenv("CONFLUENCE_AUTH_METHOD")
	if authMethod == "" {
		if os.Getenv("CONFLUENCE_SESSION_COOKIE") != "" {
			authMethod = AuthMethodSessionCookie
		} else if kind == "cloud" || os.Getenv("CONFLUENCE_EMAIL") != "" {
			authMethod = AuthMethodBasic
		} else {
			authMethod = AuthMethodBearer
		}
	}

	email := os.Getenv("CONFLUENCE_EMAIL")
	username := os.Getenv("CONFLUENCE_USER")
	if username == "" {
		username = os.Getenv("CONFLUENCE_USERNAME")
	}

	defaultSpace := os.Getenv("CONFLUENCE_SPACE")

	hostObj := &Host{
		Kind:         kind,
		BaseURL:      cleanHost,
		Email:        email,
		Username:     username,
		AuthMethod:   authMethod,
		DefaultSpace: defaultSpace,
		Token:        os.Getenv("CONFLUENCE_TOKEN"),
	}

	const envContextName = "env"
	c.Hosts[cleanHost] = hostObj
	c.Contexts[envContextName] = &Context{
		Host:     cleanHost,
		SpaceKey: defaultSpace,
	}
	if c.ActiveContext == "" {
		c.ActiveContext = envContextName
	}
}

// Save persists the configuration atomically.
func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.path == "" {
		path, err := resolvePath()
		if err != nil {
			return err
		}
		c.path = path
	}

	if err := os.MkdirAll(filepath.Dir(c.path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return os.Rename(tmp, c.path)
}

// ActiveHost returns the Host associated with the active context.
func (c *Config) ActiveHost() (*Host, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.ActiveContext == "" {
		return nil, errors.New("no active context — run `confluence context create` or `confluence auth login` first")
	}
	ctx, ok := c.Contexts[c.ActiveContext]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrContextNotFound, c.ActiveContext)
	}
	host, ok := c.Hosts[ctx.Host]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrHostNotFound, ctx.Host)
	}
	return host, nil
}

// ActiveContextInfo returns the active context name, context struct, and host struct.
func (c *Config) ActiveContextInfo() (string, *Context, *Host, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.ActiveContext == "" {
		return "", nil, nil, errors.New("no active context — run `confluence context create` or `confluence auth login` first")
	}
	ctx, ok := c.Contexts[c.ActiveContext]
	if !ok {
		return "", nil, nil, fmt.Errorf("%w: %s", ErrContextNotFound, c.ActiveContext)
	}
	host, ok := c.Hosts[ctx.Host]
	if !ok {
		return "", nil, nil, fmt.Errorf("%w: %s", ErrHostNotFound, ctx.Host)
	}
	return c.ActiveContext, ctx, host, nil
}

// ResolveSpaceKey returns the effective space key from argument, flag, env, context, or host.
func (c *Config) ResolveSpaceKey(explicitSpace string) string {
	if explicitSpace != "" {
		return strings.ToUpper(strings.TrimSpace(explicitSpace))
	}
	if env := os.Getenv("CONFLUENCE_SPACE"); env != "" {
		return strings.ToUpper(strings.TrimSpace(env))
	}
	if _, ctx, host, err := c.ActiveContextInfo(); err == nil {
		if ctx.SpaceKey != "" {
			return strings.ToUpper(strings.TrimSpace(ctx.SpaceKey))
		}
		if host.DefaultSpace != "" {
			return strings.ToUpper(strings.TrimSpace(host.DefaultSpace))
		}
	}
	return ""
}

// SetActiveContext updates the active context name and saves.
func (c *Config) SetActiveContext(name string) error {
	c.mu.Lock()
	if _, ok := c.Contexts[name]; !ok {
		c.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrContextNotFound, name)
	}
	c.ActiveContext = name
	c.mu.Unlock()
	return c.Save()
}

func resolvePath() (string, error) {
	if dir := os.Getenv("CONFLUENCE_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "config.yml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home directory: %w", err)
	}
	return filepath.Join(home, ".config", "confluence", "config.yml"), nil
}
