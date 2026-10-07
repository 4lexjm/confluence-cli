package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	confluencecfg "github.com/atlassian/confluence-cli/internal/config"
)

func TestClientBaseURLCloudNormalization(t *testing.T) {
	host := &confluencecfg.Host{
		Kind:       "cloud",
		BaseURL:    "https://myteam.atlassian.net",
		AuthMethod: confluencecfg.AuthMethodBasic,
		Email:      "agent@example.com",
		Token:      "token123",
	}

	c, err := New(host)
	require.NoError(t, err)
	assert.Equal(t, "https://myteam.atlassian.net/wiki", c.BaseURL())

	req, err := c.NewRequest(context.Background(), http.MethodGet, "/rest/api/space", nil)
	require.NoError(t, err)
	assert.Equal(t, "https://myteam.atlassian.net/wiki/rest/api/space", req.URL.String())
}

func TestClientNewRequestWithWikiPrefix(t *testing.T) {
	host := &confluencecfg.Host{
		Kind:       "cloud",
		BaseURL:    "https://myteam.atlassian.net/wiki",
		AuthMethod: confluencecfg.AuthMethodBasic,
	}

	c, err := New(host)
	require.NoError(t, err)

	// Even if user provides /wiki/rest/api/content, should not double /wiki
	req, err := c.NewRequest(context.Background(), http.MethodGet, "/wiki/rest/api/content", nil)
	require.NoError(t, err)
	assert.Equal(t, "https://myteam.atlassian.net/wiki/rest/api/content", req.URL.String())
}

func TestClientGetAndPost(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		user, pass, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "user@example.com", user)
		assert.Equal(t, "secret-token", pass)

		if r.Method == http.MethodGet && r.URL.Path == "/rest/api/test" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status": "ok"}`))
			return
		}

		if r.Method == http.MethodPost && r.URL.Path == "/rest/api/create" {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id": "123", "title": "Page 1"}`))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	host := &confluencecfg.Host{
		Kind:       "dc",
		BaseURL:    ts.URL,
		AuthMethod: confluencecfg.AuthMethodBasic,
		Username:   "user@example.com",
		Token:      "secret-token",
	}

	client, err := New(host)
	require.NoError(t, err)

	var getResult map[string]string
	resp, err := client.Get(context.Background(), "/rest/api/test", &getResult)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "ok", getResult["status"])

	var postResult map[string]string
	resp, err = client.Post(context.Background(), "/rest/api/create", map[string]string{"name": "test"}, &postResult)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "123", postResult["id"])
}
