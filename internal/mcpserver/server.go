// Package mcpserver implements the Model Context Protocol (MCP) server for Confluence CLI.
package mcpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/atlassian/confluence-cli/pkg/confluence"
	"github.com/atlassian/confluence-cli/pkg/markdown"
)

// Server coordinates MCP protocol handling over stdio.
type Server struct {
	client       *confluence.Client
	readOnly     bool
	defaultSpace string
	version      string
}

// New creates an MCP server instance.
func New(client *confluence.Client, readOnly bool, defaultSpace, version string) *Server {
	return &Server{
		client:       client,
		readOnly:     readOnly,
		defaultSpace: defaultSpace,
		version:      version,
	}
}

// JSON-RPC 2.0 types
type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   *rErr  `json:"error,omitempty"`
}

type rErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Serve reads JSON-RPC requests from in and writes responses to out until EOF.
func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)
	encoder := json.NewEncoder(out)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := reader.ReadBytes('\n')
		if err != nil {
			if errorsIsEOF(err) {
				return nil
			}
			return err
		}

		line = []byte(strings.TrimSpace(string(line)))
		if len(line) == 0 {
			continue
		}

		var req request
		if err := json.Unmarshal(line, &req); err != nil {
			_ = encoder.Encode(response{
				JSONRPC: "2.0",
				Error:   &rErr{Code: -32700, Message: "Parse error"},
			})
			continue
		}

		resp := s.handleRequest(ctx, &req)
		if resp != nil {
			if err := encoder.Encode(resp); err != nil {
				return err
			}
		}
	}
}

func (s *Server) handleRequest(ctx context.Context, req *request) *response {
	switch req.Method {
	case "initialize":
		return &response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]any{
					"tools": map[string]any{},
				},
				"serverInfo": map[string]any{
					"name":    "confluence-cli",
					"version": s.version,
				},
			},
		}

	case "notifications/initialized":
		return nil

	case "ping":
		return &response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]any{},
		}

	case "tools/list":
		return &response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"tools": s.toolsList(),
			},
		}

	case "tools/call":
		var callParams struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &callParams); err != nil {
			return &response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &rErr{Code: -32602, Message: "Invalid params"},
			}
		}

		resultText, isErr := s.executeTool(ctx, callParams.Name, callParams.Arguments)
		return &response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"content": []map[string]any{
					{
						"type": "text",
						"text": resultText,
					},
				},
				"isError": isErr,
			},
		}

	default:
		return &response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rErr{Code: -32601, Message: fmt.Sprintf("Method not found: %s", req.Method)},
		}
	}
}

func (s *Server) toolsList() []map[string]any {
	tools := []map[string]any{
		{
			"name":        "confluence_search",
			"description": "Search Confluence pages, blogposts, and content using plain text or CQL (Confluence Query Language).",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query":       map[string]any{"type": "string", "description": "Free text search term"},
					"cql":         map[string]any{"type": "string", "description": "Explicit CQL expression (e.g. `space = 'ENG' AND type = page`)"},
					"space":       map[string]any{"type": "string", "description": "Filter results by space key"},
					"type":        map[string]any{"type": "string", "description": "Filter by content type: page, blogpost, attachment", "enum": []string{"page", "blogpost", "attachment"}},
					"limit":       map[string]any{"type": "integer", "description": "Maximum number of results to return (default 25)"},
				},
			},
		},
		{
			"name":        "confluence_get_page",
			"description": "Read a Confluence page's metadata, title, and body content by page ID or space+title.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id_or_title": map[string]any{"type": "string", "description": "Page ID (e.g. '123456') or page title"},
					"space":       map[string]any{"type": "string", "description": "Space key (required if page title is specified instead of numeric ID)"},
					"format":      map[string]any{"type": "string", "description": "Body format: markdown (default) or storage (XHTML)", "enum": []string{"markdown", "storage"}},
				},
				"required": []string{"id_or_title"},
			},
		},
		{
			"name":        "confluence_list_pages",
			"description": "List pages in a Confluence space.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"space": map[string]any{"type": "string", "description": "Space key (defaults to context default space)"},
					"limit": map[string]any{"type": "integer", "description": "Maximum pages to return (default 25)"},
					"title": map[string]any{"type": "string", "description": "Optional title filter"},
				},
			},
		},
		{
			"name":        "confluence_list_spaces",
			"description": "List Confluence spaces.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"type":  map[string]any{"type": "string", "description": "Space type (global, personal)"},
					"limit": map[string]any{"type": "integer", "description": "Maximum spaces to return (default 25)"},
				},
			},
		},
		{
			"name":        "confluence_get_space",
			"description": "Get detailed metadata for a Confluence space.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{"type": "string", "description": "Space key (e.g. 'ENG')"},
				},
				"required": []string{"key"},
			},
		},
		{
			"name":        "confluence_list_comments",
			"description": "List comments on a Confluence page.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"page_id": map[string]any{"type": "string", "description": "ID of the Confluence page"},
					"limit":   map[string]any{"type": "integer", "description": "Maximum comments to return (default 25)"},
				},
				"required": []string{"page_id"},
			},
		},
		{
			"name":        "confluence_list_attachments",
			"description": "List attachments uploaded to a Confluence page.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"page_id": map[string]any{"type": "string", "description": "ID of the Confluence page"},
					"limit":   map[string]any{"type": "integer", "description": "Maximum attachments to return (default 25)"},
				},
				"required": []string{"page_id"},
			},
		},
		{
			"name":        "confluence_get_page_history",
			"description": "Get version history and changes for a Confluence page.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"page_id": map[string]any{"type": "string", "description": "ID of the Confluence page"},
					"limit":   map[string]any{"type": "integer", "description": "Maximum version entries (default 20)"},
				},
				"required": []string{"page_id"},
			},
		},
	}

	if !s.readOnly {
		tools = append(tools,
			map[string]any{
				"name":        "confluence_create_page",
				"description": "Create a new page in a Confluence space. Accepts Markdown or raw XHTML storage format.",
				"inputSchema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"space":     map[string]any{"type": "string", "description": "Space key (e.g. 'ENG')"},
						"title":     map[string]any{"type": "string", "description": "Title of the page"},
						"body":      map[string]any{"type": "string", "description": "Page body content (Markdown by default)"},
						"parent_id": map[string]any{"type": "string", "description": "Optional parent page ID to nest this page under"},
						"format":    map[string]any{"type": "string", "description": "Body format: markdown (default) or storage (raw XHTML)", "enum": []string{"markdown", "storage"}},
					},
					"required": []string{"space", "title", "body"},
				},
			},
			map[string]any{
				"name":        "confluence_update_page",
				"description": "Update an existing Confluence page's title or body content.",
				"inputSchema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"page_id":         map[string]any{"type": "string", "description": "ID of the page to update"},
						"title":           map[string]any{"type": "string", "description": "New page title (optional)"},
						"body":            map[string]any{"type": "string", "description": "New body content (optional, Markdown by default)"},
						"version_comment": map[string]any{"type": "string", "description": "Note describing the update in version history"},
						"format":          map[string]any{"type": "string", "description": "Body format: markdown (default) or storage (raw XHTML)", "enum": []string{"markdown", "storage"}},
					},
					"required": []string{"page_id"},
				},
			},
			map[string]any{
				"name":        "confluence_delete_page",
				"description": "Delete / trash a Confluence page.",
				"inputSchema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"page_id": map[string]any{"type": "string", "description": "ID of the page to delete"},
					},
					"required": []string{"page_id"},
				},
			},
			map[string]any{
				"name":        "confluence_add_comment",
				"description": "Post a comment on a Confluence page.",
				"inputSchema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"page_id":   map[string]any{"type": "string", "description": "ID of the page to comment on"},
						"body":      map[string]any{"type": "string", "description": "Comment content (Markdown supported)"},
						"parent_id": map[string]any{"type": "string", "description": "Optional parent comment ID for threaded replies"},
					},
					"required": []string{"page_id", "body"},
				},
			},
			map[string]any{
				"name":        "confluence_raw_api",
				"description": "Execute an arbitrary Confluence REST API call (escape hatch for advanced endpoints).",
				"inputSchema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"method": map[string]any{"type": "string", "description": "HTTP method: GET, POST, PUT, DELETE", "enum": []string{"GET", "POST", "PUT", "DELETE"}},
						"path":   map[string]any{"type": "string", "description": "API path (e.g. /rest/api/space or /wiki/api/v2/pages)"},
						"body":   map[string]any{"type": "object", "description": "JSON payload object for POST/PUT requests"},
					},
					"required": []string{"method", "path"},
				},
			},
		)
	}

	return tools
}

func (s *Server) executeTool(ctx context.Context, name string, args map[string]any) (string, bool) {
	if s.readOnly && isMutatingTool(name) {
		return "Error: server is running in read-only mode; mutating tools are disabled", true
	}

	switch name {
	case "confluence_search":
		query, _ := args["query"].(string)
		cql, _ := args["cql"].(string)
		space, _ := args["space"].(string)
		cType, _ := args["type"].(string)
		limit := getIntArg(args, "limit", 25)

		if space == "" {
			space = s.defaultSpace
		}

		var res *confluence.SearchResult
		var err error
		if cql != "" {
			res, err = s.client.SearchCQL(ctx, cql, limit, 0)
		} else {
			res, err = s.client.Search(ctx, query, space, cType, limit)
		}
		if err != nil {
			return fmt.Sprintf("Search error: %v", err), true
		}
		return toJSON(res), false

	case "confluence_get_page":
		idOrTitle, _ := args["id_or_title"].(string)
		space, _ := args["space"].(string)
		format, _ := args["format"].(string)
		if space == "" {
			space = s.defaultSpace
		}

		page, err := s.client.GetPage(ctx, idOrTitle, space, "")
		if err != nil {
			return fmt.Sprintf("Get page error: %v", err), true
		}

		bodyContent := ""
		if page.Body != nil {
			if format == "storage" && page.Body.Storage != nil {
				bodyContent = page.Body.Storage.Value
			} else {
				if page.Body.Storage != nil {
					bodyContent = markdown.StorageToMarkdown(page.Body.Storage.Value)
				}
			}
		}

		result := map[string]any{
			"id":        page.ID,
			"title":     page.Title,
			"space":     page.Space,
			"version":   page.Version,
			"url":       page.WebUIURL(s.client.BaseURL()),
			"body":      bodyContent,
			"format":    format,
			"ancestors": page.Ancestors,
		}
		return toJSON(result), false

	case "confluence_list_pages":
		space, _ := args["space"].(string)
		limit := getIntArg(args, "limit", 25)
		title, _ := args["title"].(string)
		if space == "" {
			space = s.defaultSpace
		}

		pages, err := s.client.ListPages(ctx, space, limit, 0, title)
		if err != nil {
			return fmt.Sprintf("List pages error: %v", err), true
		}
		return toJSON(pages.Results), false

	case "confluence_list_spaces":
		spaceType, _ := args["type"].(string)
		limit := getIntArg(args, "limit", 25)
		spaces, err := s.client.ListSpaces(ctx, limit, 0, spaceType)
		if err != nil {
			return fmt.Sprintf("List spaces error: %v", err), true
		}
		return toJSON(spaces.Results), false

	case "confluence_get_space":
		key, _ := args["key"].(string)
		space, err := s.client.GetSpace(ctx, key)
		if err != nil {
			return fmt.Sprintf("Get space error: %v", err), true
		}
		return toJSON(space), false

	case "confluence_list_comments":
		pageID, _ := args["page_id"].(string)
		limit := getIntArg(args, "limit", 25)
		comments, err := s.client.ListComments(ctx, pageID, limit, 0)
		if err != nil {
			return fmt.Sprintf("List comments error: %v", err), true
		}

		type commentView struct {
			ID     string           `json:"id"`
			Author *confluence.User `json:"author,omitempty"`
			Body   string           `json:"body"`
		}
		var formatted []commentView
		for _, c := range comments.Results {
			b := ""
			if c.Body != nil && c.Body.Storage != nil {
				b = markdown.StorageToMarkdown(c.Body.Storage.Value)
			}
			var author *confluence.User
			if c.Version != nil {
				author = c.Version.By
			}
			formatted = append(formatted, commentView{ID: c.ID, Author: author, Body: b})
		}
		return toJSON(formatted), false

	case "confluence_list_attachments":
		pageID, _ := args["page_id"].(string)
		limit := getIntArg(args, "limit", 25)
		attachments, err := s.client.ListAttachments(ctx, pageID, limit, 0)
		if err != nil {
			return fmt.Sprintf("List attachments error: %v", err), true
		}
		return toJSON(attachments.Results), false

	case "confluence_get_page_history":
		pageID, _ := args["page_id"].(string)
		limit := getIntArg(args, "limit", 20)
		history, err := s.client.GetPageHistory(ctx, pageID, limit)
		if err != nil {
			return fmt.Sprintf("Get history error: %v", err), true
		}
		return toJSON(history.Results), false

	case "confluence_create_page":
		space, _ := args["space"].(string)
		title, _ := args["title"].(string)
		body, _ := args["body"].(string)
		parentID, _ := args["parent_id"].(string)
		format, _ := args["format"].(string)

		if space == "" {
			space = s.defaultSpace
		}
		storageBody := body
		if format == "markdown" || format == "" {
			storageBody = markdown.MarkdownToStorage(body)
		}

		created, err := s.client.CreatePage(ctx, space, title, storageBody, parentID)
		if err != nil {
			return fmt.Sprintf("Create page error: %v", err), true
		}
		return toJSON(created), false

	case "confluence_update_page":
		pageID, _ := args["page_id"].(string)
		title, _ := args["title"].(string)
		body, _ := args["body"].(string)
		versionComment, _ := args["version_comment"].(string)
		format, _ := args["format"].(string)

		storageBody := body
		if body != "" && (format == "markdown" || format == "") {
			storageBody = markdown.MarkdownToStorage(body)
		}

		updated, err := s.client.UpdatePage(ctx, pageID, title, storageBody, 0, versionComment)
		if err != nil {
			return fmt.Sprintf("Update page error: %v", err), true
		}
		return toJSON(updated), false

	case "confluence_delete_page":
		pageID, _ := args["page_id"].(string)
		if err := s.client.DeletePage(ctx, pageID); err != nil {
			return fmt.Sprintf("Delete page error: %v", err), true
		}
		return fmt.Sprintf("Successfully deleted page %s", pageID), false

	case "confluence_add_comment":
		pageID, _ := args["page_id"].(string)
		body, _ := args["body"].(string)
		parentID, _ := args["parent_id"].(string)

		storageBody := markdown.MarkdownToStorage(body)
		created, err := s.client.AddComment(ctx, pageID, storageBody, parentID)
		if err != nil {
			return fmt.Sprintf("Add comment error: %v", err), true
		}
		return toJSON(created), false

	case "confluence_raw_api":
		method, _ := args["method"].(string)
		path, _ := args["path"].(string)
		bodyObj := args["body"]

		_, rawBytes, err := s.client.Raw(ctx, method, path, bodyObj, nil)
		if err != nil {
			return fmt.Sprintf("API error: %v", err), true
		}
		return string(rawBytes), false

	default:
		return fmt.Sprintf("Unknown tool: %s", name), true
	}
}

func isMutatingTool(name string) bool {
	switch name {
	case "confluence_create_page", "confluence_update_page", "confluence_delete_page", "confluence_add_comment", "confluence_raw_api":
		return true
	}
	return false
}

func getIntArg(args map[string]any, key string, def int) int {
	if val, ok := args[key]; ok {
		switch n := val.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return def
}

func toJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

func errorsIsEOF(err error) bool {
	return err == io.EOF
}
