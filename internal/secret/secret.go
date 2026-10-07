// Package secret manages credential storage via OS keychain with file fallback.
package secret

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zalando/go-keyring"
)

const serviceName = "confluence-cli"

// ErrNotFound is returned when no credential exists for the given key.
var ErrNotFound = errors.New("credential not found")

// Set stores value for key in the OS keychain, with optional file fallback.
func Set(key, value string) error {
	if err := keyring.Set(serviceName, key, value); err != nil {
		// If keyring fails (e.g. headless CI or headless Linux without D-Bus),
		// fall back to writing to restricted credentials file.
		return setFileFallback(key, value)
	}
	return nil
}

// Get retrieves the value for key from the OS keychain or fallback store.
// Returns ErrNotFound when no credential exists.
func Get(key string) (string, error) {
	val, err := keyring.Get(serviceName, key)
	if err == nil {
		return val, nil
	}

	// Try fallback file store
	fileVal, fileErr := getFileFallback(key)
	if fileErr == nil {
		return fileVal, nil
	}

	if errors.Is(err, keyring.ErrNotFound) || errors.Is(fileErr, ErrNotFound) {
		return "", ErrNotFound
	}
	return "", fmt.Errorf("retrieve secret %q: %w", key, err)
}

// Delete removes the stored credential for key.
func Delete(key string) error {
	var delErr error
	if err := keyring.Delete(serviceName, key); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		delErr = err
	}
	_ = deleteFileFallback(key)
	return delErr
}

// TokenKey returns the keychain key for a host's API token / PAT.
func TokenKey(host string) string { return "token:" + normalizeHostKey(host) }

// SessionKey returns the keychain key for a host's session cookie.
func SessionKey(host string) string { return "session:" + normalizeHostKey(host) }

// EnvToken reads CONFLUENCE_TOKEN from the environment.
func EnvToken() string { return os.Getenv("CONFLUENCE_TOKEN") }

// EnvSessionCookie reads CONFLUENCE_SESSION_COOKIE from the environment.
func EnvSessionCookie() string { return os.Getenv("CONFLUENCE_SESSION_COOKIE") }

func normalizeHostKey(host string) string {
	return strings.TrimRight(strings.ToLower(strings.TrimSpace(host)), "/")
}

func fallbackDir() (string, error) {
	if dir := os.Getenv("CONFLUENCE_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "secrets"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "confluence", "secrets"), nil
}

func keyFile(key string) (string, error) {
	dir, err := fallbackDir()
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(key))
	filename := hex.EncodeToString(hash[:16]) + ".secret"
	return filepath.Join(dir, filename), nil
}

func setFileFallback(key, value string) error {
	filePath, err := keyFile(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(filePath), 0o700); err != nil {
		return fmt.Errorf("create fallback secrets dir: %w", err)
	}
	return os.WriteFile(filePath, []byte(value), 0o600)
}

func getFileFallback(key string) (string, error) {
	filePath, err := keyFile(key)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrNotFound
		}
		return "", err
	}
	return string(data), nil
}

func deleteFileFallback(key string) error {
	filePath, err := keyFile(key)
	if err != nil {
		return err
	}
	err = os.Remove(filePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
