package secret

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretFallbackStore(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CONFLUENCE_CONFIG_DIR", tmpDir)

	testKey := TokenKey("https://myorg.atlassian.net/wiki")
	testVal := "super-secret-pat-token"

	// Test fallback save/load
	err := setFileFallback(testKey, testVal)
	require.NoError(t, err)

	val, err := getFileFallback(testKey)
	require.NoError(t, err)
	assert.Equal(t, testVal, val)

	// Test delete
	err = deleteFileFallback(testKey)
	require.NoError(t, err)

	_, err = getFileFallback(testKey)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestSecretEnv(t *testing.T) {
	t.Setenv("CONFLUENCE_TOKEN", "my-env-token")
	t.Setenv("CONFLUENCE_SESSION_COOKIE", "my-session-cookie")

	assert.Equal(t, "my-env-token", EnvToken())
	assert.Equal(t, "my-session-cookie", EnvSessionCookie())
}
