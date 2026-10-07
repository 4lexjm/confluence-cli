package auth

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionTransport(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(SessionCookieName)
		if err != nil || cookie.Value != "valid-cookie" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()

	getCookie := func() (string, error) {
		return "valid-cookie", nil
	}

	transport := NewSessionTransport(nil, ts.URL, getCookie)
	client := &http.Client{Transport: transport}

	req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "ok", string(body))
}

func TestSessionExpiryCalculations(t *testing.T) {
	now := time.Now()
	exp := EstimatedExpiry(now)
	assert.True(t, exp.After(now))
	assert.True(t, SessionAge(now.Add(-10*time.Minute)) >= 10*time.Minute)
}
