package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	confluencecfg "github.com/atlassian/confluence-cli/internal/config"
	"github.com/atlassian/confluence-cli/pkg/confluence"
	"github.com/atlassian/confluence-cli/pkg/httpx"
)

func TestMCPServerInitializeAndToolsList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer ts.Close()

	hClient, err := httpx.New(&confluencecfg.Host{Kind: "dc", BaseURL: ts.URL, AuthMethod: confluencecfg.AuthMethodBasic})
	require.NoError(t, err)
	cl := confluence.NewClient(hClient)

	server := New(cl, false, "DEV", "1.0.0")

	// 1. Test initialize
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n"
	// 2. Test tools/list
	toolsReq := `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}` + "\n"

	in := bytes.NewBufferString(initReq + toolsReq)
	var out bytes.Buffer

	err = server.Serve(context.Background(), in, &out)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	require.Len(t, lines, 2)

	var initResp response
	err = json.Unmarshal([]byte(lines[0]), &initResp)
	require.NoError(t, err)
	assert.Equal(t, float64(1), initResp.ID)
	resMap := initResp.Result.(map[string]any)
	assert.Equal(t, "2024-11-05", resMap["protocolVersion"])

	var toolsResp response
	err = json.Unmarshal([]byte(lines[1]), &toolsResp)
	require.NoError(t, err)
	assert.Equal(t, float64(2), toolsResp.ID)
	toolsMap := toolsResp.Result.(map[string]any)
	tools := toolsMap["tools"].([]any)
	assert.NotEmpty(t, tools)
}
