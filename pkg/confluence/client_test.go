package confluence

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	confluencecfg "github.com/atlassian/confluence-cli/internal/config"
	"github.com/atlassian/confluence-cli/pkg/httpx"
)

func TestClientSpaces(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/rest/api/space" {
			resp := ListResponse[Space]{
				Results: []Space{
					{Key: "DEV", Name: "Developer Space", Type: "global"},
				},
				Limit: 25,
				Size:  1,
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/rest/api/space/DEV" {
			space := Space{Key: "DEV", Name: "Developer Space"}
			_ = json.NewEncoder(w).Encode(space)
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/rest/api/space" {
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			space := Space{Key: payload["key"].(string), Name: payload["name"].(string)}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(space)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	hClient, err := httpx.New(&confluencecfg.Host{Kind: "dc", BaseURL: ts.URL, AuthMethod: confluencecfg.AuthMethodBasic})
	require.NoError(t, err)

	client := NewClient(hClient)

	// List
	spaces, err := client.ListSpaces(context.Background(), 10, 0, "")
	require.NoError(t, err)
	assert.Len(t, spaces.Results, 1)
	assert.Equal(t, "DEV", spaces.Results[0].Key)

	// Get
	sp, err := client.GetSpace(context.Background(), "DEV")
	require.NoError(t, err)
	assert.Equal(t, "Developer Space", sp.Name)

	// Create
	created, err := client.CreateSpace(context.Background(), "NEW", "New Space", "Desc")
	require.NoError(t, err)
	assert.Equal(t, "NEW", created.Key)
}

func TestClientPagesAndSearch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/rest/api/content/12345" {
			page := Content{
				ID:    "12345",
				Type:  "page",
				Title: "Architecture Overview",
				Space: &SpaceSummary{Key: "ENG"},
				Version: &ContentVersion{
					Number: 1,
				},
				Body: &ContentBody{
					Storage: &ContentBodyValue{
						Value:          "<p>Hello world</p>",
						Representation: "storage",
					},
				},
			}
			_ = json.NewEncoder(w).Encode(page)
			return
		}

		if r.Method == http.MethodGet && r.URL.Path == "/rest/api/content/search" {
			assert.Contains(t, r.URL.RawQuery, "cql=")
			resp := SearchResult{
				Results: []*Content{
					{ID: "12345", Title: "Architecture Overview"},
				},
				Size: 1,
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	hClient, err := httpx.New(&confluencecfg.Host{Kind: "dc", BaseURL: ts.URL, AuthMethod: confluencecfg.AuthMethodBasic})
	require.NoError(t, err)
	client := NewClient(hClient)

	// GetPage by numeric ID
	page, err := client.GetPage(context.Background(), "12345", "", "")
	require.NoError(t, err)
	assert.Equal(t, "Architecture Overview", page.Title)
	assert.Equal(t, "<p>Hello world</p>", page.Body.Storage.Value)

	// SearchCQL
	searchRes, err := client.SearchCQL(context.Background(), "type = page and space = ENG", 10, 0)
	require.NoError(t, err)
	assert.Len(t, searchRes.Results, 1)
	assert.Equal(t, "12345", searchRes.Results[0].ID)
}
