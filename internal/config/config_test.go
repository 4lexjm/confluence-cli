package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")
	t.Setenv("CONFLUENCE_CONFIG_DIR", tmpDir)

	cfg := &Config{
		Version:       1,
		ActiveContext: "test-cloud",
		Contexts: map[string]*Context{
			"test-cloud": {
				Host:     "https://test.atlassian.net/wiki",
				SpaceKey: "DEV",
			},
		},
		Hosts: map[string]*Host{
			"https://test.atlassian.net/wiki": {
				Kind:         "cloud",
				BaseURL:      "https://test.atlassian.net/wiki",
				Email:        "bot@example.com",
				AuthMethod:   AuthMethodBasic,
				DefaultSpace: "DEV",
				Token:        "secret-token-should-not-persist",
			},
		},
		path: configPath,
	}

	err := cfg.Save()
	require.NoError(t, err)

	// Check file contents do not include Token
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "secret-token-should-not-persist")

	// Load config back
	loaded, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "test-cloud", loaded.ActiveContext)
	assert.Equal(t, "DEV", loaded.Contexts["test-cloud"].SpaceKey)
	assert.Equal(t, "https://test.atlassian.net/wiki", loaded.Hosts["https://test.atlassian.net/wiki"].BaseURL)
	assert.Empty(t, loaded.Hosts["https://test.atlassian.net/wiki"].Token)

	// Active host
	host, err := loaded.ActiveHost()
	require.NoError(t, err)
	assert.Equal(t, "cloud", host.Kind)

	// Space resolution
	assert.Equal(t, "DEV", loaded.ResolveSpaceKey(""))
	assert.Equal(t, "ENG", loaded.ResolveSpaceKey("eng"))
}

func TestConfigEnvOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CONFLUENCE_CONFIG_DIR", tmpDir)
	t.Setenv("CONFLUENCE_HOST", "https://mycorp.atlassian.net")
	t.Setenv("CONFLUENCE_TOKEN", "api-token-123")
	t.Setenv("CONFLUENCE_EMAIL", "agent@mycorp.com")
	t.Setenv("CONFLUENCE_SPACE", "DOCS")

	loaded, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "env", loaded.ActiveContext)

	host, err := loaded.ActiveHost()
	require.NoError(t, err)
	assert.Equal(t, "cloud", host.Kind)
	assert.Equal(t, "api-token-123", host.Token)
	assert.Equal(t, "agent@mycorp.com", host.Email)
	assert.Equal(t, "DOCS", loaded.ResolveSpaceKey(""))
}
