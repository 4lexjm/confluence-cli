package confluence

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/atlassian/confluence-cli/pkg/httpx"
)

// Client wraps the lower-level HTTP client with Confluence REST API methods.
type Client struct {
	http *httpx.Client
}

// NewClient returns a new Confluence API client.
func NewClient(http *httpx.Client) *Client {
	return &Client{http: http}
}

// BaseURL returns the base URL of the client.
func (c *Client) BaseURL() string {
	return c.http.BaseURL()
}

// HTTPClient returns the underlying httpx.Client.
func (c *Client) HTTPClient() *httpx.Client {
	return c.http
}

// ─────────────────────────────────────────────────────────────────────────────
// Spaces
// ─────────────────────────────────────────────────────────────────────────────

// ListSpaces retrieves spaces with optional filtering by type.
func (c *Client) ListSpaces(ctx context.Context, limit, start int, spaceType string) (*ListResponse[Space], error) {
	if limit <= 0 {
		limit = 25
	}
	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))
	if start > 0 {
		params.Set("start", strconv.Itoa(start))
	}
	if spaceType != "" {
		params.Set("type", spaceType)
	}
	params.Set("status", "current")

	path := "/rest/api/space?" + params.Encode()
	var resp ListResponse[Space]
	if _, err := c.http.Get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("list spaces: %w", err)
	}
	return &resp, nil
}

// GetSpace retrieves details of a space by its space key.
func (c *Client) GetSpace(ctx context.Context, spaceKey string) (*Space, error) {
	path := fmt.Sprintf("/rest/api/space/%s?expand=description.plain,description.view,homepage", url.PathEscape(spaceKey))
	var space Space
	if _, err := c.http.Get(ctx, path, &space); err != nil {
		return nil, fmt.Errorf("get space %q: %w", spaceKey, err)
	}
	return &space, nil
}

// CreateSpace creates a new space.
func (c *Client) CreateSpace(ctx context.Context, key, name, description string) (*Space, error) {
	payload := map[string]any{
		"key":  key,
		"name": name,
	}
	if description != "" {
		payload["description"] = map[string]any{
			"plain": map[string]string{
				"value":          description,
				"representation": "plain",
			},
		}
	}

	var created Space
	if _, err := c.http.Post(ctx, "/rest/api/space", payload, &created); err != nil {
		return nil, fmt.Errorf("create space %q: %w", key, err)
	}
	return &created, nil
}

// DeleteSpace deletes a space.
func (c *Client) DeleteSpace(ctx context.Context, spaceKey string) error {
	path := fmt.Sprintf("/rest/api/space/%s", url.PathEscape(spaceKey))
	if _, err := c.http.Delete(ctx, path); err != nil {
		return fmt.Errorf("delete space %q: %w", spaceKey, err)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Pages & Content
// ─────────────────────────────────────────────────────────────────────────────

// ListPages lists pages, optionally filtered by space key and title.
func (c *Client) ListPages(ctx context.Context, spaceKey string, limit, start int, title string) (*ListResponse[*Content], error) {
	if limit <= 0 {
		limit = 25
	}
	params := url.Values{}
	params.Set("type", "page")
	params.Set("limit", strconv.Itoa(limit))
	params.Set("expand", "version,space,history")
	if start > 0 {
		params.Set("start", strconv.Itoa(start))
	}
	if spaceKey != "" {
		params.Set("spaceKey", spaceKey)
	}
	if title != "" {
		params.Set("title", title)
	}

	path := "/rest/api/content?" + params.Encode()
	var resp ListResponse[*Content]
	if _, err := c.http.Get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("list pages: %w", err)
	}
	return &resp, nil
}

// GetPage retrieves a page by ID or spaceKey+title.
func (c *Client) GetPage(ctx context.Context, idOrTitle, spaceKey string, expand string) (*Content, error) {
	if expand == "" {
		expand = "body.storage,body.view,version,space,ancestors,history,metadata.labels"
	}

	// If numeric ID, fetch directly by ID
	if _, err := strconv.ParseInt(idOrTitle, 10, 64); err == nil {
		path := fmt.Sprintf("/rest/api/content/%s?expand=%s", url.PathEscape(idOrTitle), url.QueryEscape(expand))
		var page Content
		if _, err := c.http.Get(ctx, path, &page); err != nil {
			return nil, fmt.Errorf("get page by id %s: %w", idOrTitle, err)
		}
		return &page, nil
	}

	// Otherwise, find by title in space
	params := url.Values{}
	params.Set("type", "page")
	params.Set("title", idOrTitle)
	params.Set("expand", expand)
	if spaceKey != "" {
		params.Set("spaceKey", spaceKey)
	}
	path := "/rest/api/content?" + params.Encode()

	var resp ListResponse[*Content]
	if _, err := c.http.Get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("find page %q: %w", idOrTitle, err)
	}
	if len(resp.Results) == 0 {
		return nil, fmt.Errorf("page %q not found in space %q", idOrTitle, spaceKey)
	}
	return resp.Results[0], nil
}

// CreatePage creates a new page in a space.
func (c *Client) CreatePage(ctx context.Context, spaceKey, title, bodyStorage, parentID string) (*Content, error) {
	payload := map[string]any{
		"type":  "page",
		"title": title,
		"space": map[string]string{
			"key": spaceKey,
		},
		"body": map[string]any{
			"storage": map[string]string{
				"value":          bodyStorage,
				"representation": "storage",
			},
		},
	}

	if parentID != "" {
		payload["ancestors"] = []map[string]string{
			{"id": parentID},
		}
	}

	var created Content
	if _, err := c.http.Post(ctx, "/rest/api/content", payload, &created); err != nil {
		return nil, fmt.Errorf("create page %q: %w", title, err)
	}
	return &created, nil
}

// UpdatePage updates an existing page's title and/or body.
func (c *Client) UpdatePage(ctx context.Context, pageID, title, bodyStorage string, versionNum int, versionMessage string) (*Content, error) {
	// If versionNum not provided, fetch current page version first
	if versionNum <= 0 {
		curr, err := c.GetPage(ctx, pageID, "", "version,space")
		if err != nil {
			return nil, fmt.Errorf("fetch page before update: %w", err)
		}
		versionNum = curr.Version.Number + 1
		if title == "" {
			title = curr.Title
		}
	}

	payload := map[string]any{
		"id":    pageID,
		"type":  "page",
		"title": title,
		"body": map[string]any{
			"storage": map[string]string{
				"value":          bodyStorage,
				"representation": "storage",
			},
		},
		"version": map[string]any{
			"number":  versionNum,
			"message": versionMessage,
		},
	}

	path := fmt.Sprintf("/rest/api/content/%s", url.PathEscape(pageID))
	var updated Content
	if _, err := c.http.Put(ctx, path, payload, &updated); err != nil {
		return nil, fmt.Errorf("update page %s: %w", pageID, err)
	}
	return &updated, nil
}

// DeletePage moves a page to trash or deletes it permanently.
func (c *Client) DeletePage(ctx context.Context, pageID string) error {
	path := fmt.Sprintf("/rest/api/content/%s", url.PathEscape(pageID))
	if _, err := c.http.Delete(ctx, path); err != nil {
		return fmt.Errorf("delete page %s: %w", pageID, err)
	}
	return nil
}

// MovePage moves a page under a new parent or to another space.
func (c *Client) MovePage(ctx context.Context, pageID, targetParentID, targetSpaceKey string) (*Content, error) {
	curr, err := c.GetPage(ctx, pageID, "", "version,space,body.storage")
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"id":    pageID,
		"type":  "page",
		"title": curr.Title,
		"body":  curr.Body,
		"version": map[string]any{
			"number":  curr.Version.Number + 1,
			"message": "Moved via confluence-cli",
		},
	}

	if targetSpaceKey != "" {
		payload["space"] = map[string]string{"key": targetSpaceKey}
	}
	if targetParentID != "" {
		payload["ancestors"] = []map[string]string{{"id": targetParentID}}
	}

	path := fmt.Sprintf("/rest/api/content/%s", url.PathEscape(pageID))
	var updated Content
	if _, err := c.http.Put(ctx, path, payload, &updated); err != nil {
		return nil, fmt.Errorf("move page %s: %w", pageID, err)
	}
	return &updated, nil
}

// CopyPage creates a copy of a page.
func (c *Client) CopyPage(ctx context.Context, pageID, newTitle, targetSpaceKey string) (*Content, error) {
	curr, err := c.GetPage(ctx, pageID, "", "body.storage,space")
	if err != nil {
		return nil, err
	}

	spaceKey := curr.Space.Key
	if targetSpaceKey != "" {
		spaceKey = targetSpaceKey
	}
	if newTitle == "" {
		newTitle = curr.Title + " (Copy)"
	}

	body := ""
	if curr.Body != nil && curr.Body.Storage != nil {
		body = curr.Body.Storage.Value
	}

	return c.CreatePage(ctx, spaceKey, newTitle, body, "")
}

// GetPageHistory retrieves version history for a page.
func (c *Client) GetPageHistory(ctx context.Context, pageID string, limit int) (*VersionHistory, error) {
	if limit <= 0 {
		limit = 20
	}
	path := fmt.Sprintf("/rest/api/content/%s/version?limit=%d&expand=collaborators", url.PathEscape(pageID), limit)
	var resp VersionHistory
	if _, err := c.http.Get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("get page history %s: %w", pageID, err)
	}
	return &resp, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Blog Posts
// ─────────────────────────────────────────────────────────────────────────────

// ListBlogPosts retrieves blog posts in a space.
func (c *Client) ListBlogPosts(ctx context.Context, spaceKey string, limit, start int) (*ListResponse[*Content], error) {
	if limit <= 0 {
		limit = 25
	}
	params := url.Values{}
	params.Set("type", "blogpost")
	params.Set("limit", strconv.Itoa(limit))
	params.Set("expand", "version,space,history")
	if start > 0 {
		params.Set("start", strconv.Itoa(start))
	}
	if spaceKey != "" {
		params.Set("spaceKey", spaceKey)
	}

	path := "/rest/api/content?" + params.Encode()
	var resp ListResponse[*Content]
	if _, err := c.http.Get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("list blog posts: %w", err)
	}
	return &resp, nil
}

// CreateBlogPost creates a blog post.
func (c *Client) CreateBlogPost(ctx context.Context, spaceKey, title, bodyStorage string) (*Content, error) {
	payload := map[string]any{
		"type":  "blogpost",
		"title": title,
		"space": map[string]string{
			"key": spaceKey,
		},
		"body": map[string]any{
			"storage": map[string]string{
				"value":          bodyStorage,
				"representation": "storage",
			},
		},
	}

	var created Content
	if _, err := c.http.Post(ctx, "/rest/api/content", payload, &created); err != nil {
		return nil, fmt.Errorf("create blog post %q: %w", title, err)
	}
	return &created, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Comments
// ─────────────────────────────────────────────────────────────────────────────

// ListComments retrieves comments on a content item.
func (c *Client) ListComments(ctx context.Context, pageID string, limit, start int) (*ListResponse[*Content], error) {
	if limit <= 0 {
		limit = 25
	}
	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))
	params.Set("expand", "body.view,body.storage,version,history")
	if start > 0 {
		params.Set("start", strconv.Itoa(start))
	}

	path := fmt.Sprintf("/rest/api/content/%s/child/comment?%s", url.PathEscape(pageID), params.Encode())
	var resp ListResponse[*Content]
	if _, err := c.http.Get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("list comments for %s: %w", pageID, err)
	}
	return &resp, nil
}

// AddComment adds a new comment to a page.
func (c *Client) AddComment(ctx context.Context, pageID, bodyStorage, parentCommentID string) (*Content, error) {
	payload := map[string]any{
		"type": "comment",
		"container": map[string]string{
			"id":   pageID,
			"type": "page",
		},
		"body": map[string]any{
			"storage": map[string]string{
				"value":          bodyStorage,
				"representation": "storage",
			},
		},
	}

	if parentCommentID != "" {
		payload["ancestors"] = []map[string]string{
			{"id": parentCommentID},
		}
	}

	var created Content
	if _, err := c.http.Post(ctx, "/rest/api/content", payload, &created); err != nil {
		return nil, fmt.Errorf("add comment to %s: %w", pageID, err)
	}
	return &created, nil
}

// UpdateComment updates an existing comment.
func (c *Client) UpdateComment(ctx context.Context, commentID, bodyStorage string, versionNum int) (*Content, error) {
	if versionNum <= 0 {
		curr, err := c.GetPage(ctx, commentID, "", "version")
		if err != nil {
			return nil, fmt.Errorf("fetch comment before update: %w", err)
		}
		versionNum = curr.Version.Number + 1
	}

	payload := map[string]any{
		"id":   commentID,
		"type": "comment",
		"body": map[string]any{
			"storage": map[string]string{
				"value":          bodyStorage,
				"representation": "storage",
			},
		},
		"version": map[string]any{
			"number": versionNum,
		},
	}

	path := fmt.Sprintf("/rest/api/content/%s", url.PathEscape(commentID))
	var updated Content
	if _, err := c.http.Put(ctx, path, payload, &updated); err != nil {
		return nil, fmt.Errorf("update comment %s: %w", commentID, err)
	}
	return &updated, nil
}

// DeleteComment deletes a comment.
func (c *Client) DeleteComment(ctx context.Context, commentID string) error {
	path := fmt.Sprintf("/rest/api/content/%s", url.PathEscape(commentID))
	if _, err := c.http.Delete(ctx, path); err != nil {
		return fmt.Errorf("delete comment %s: %w", commentID, err)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Attachments
// ─────────────────────────────────────────────────────────────────────────────

// ListAttachments lists attachments for a page.
func (c *Client) ListAttachments(ctx context.Context, pageID string, limit, start int) (*ListResponse[*Attachment], error) {
	if limit <= 0 {
		limit = 25
	}
	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))
	params.Set("expand", "version")
	if start > 0 {
		params.Set("start", strconv.Itoa(start))
	}

	path := fmt.Sprintf("/rest/api/content/%s/child/attachment?%s", url.PathEscape(pageID), params.Encode())
	var resp ListResponse[*Attachment]
	if _, err := c.http.Get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("list attachments for %s: %w", pageID, err)
	}
	return &resp, nil
}

// UploadAttachment uploads a file as an attachment to a page.
func (c *Client) UploadAttachment(ctx context.Context, pageID, filename string, r io.Reader, comment string) (*ListResponse[*Attachment], error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("create multipart form: %w", err)
	}
	if _, err := io.Copy(part, r); err != nil {
		return nil, fmt.Errorf("copy file data: %w", err)
	}

	if comment != "" {
		if err := writer.WriteField("comment", comment); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/rest/api/content/%s/child/attachment", url.PathEscape(pageID))
	req, err := c.http.NewRequest(ctx, http.MethodPost, path, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Atlassian-Token", "nocheck")

	var resp ListResponse[*Attachment]
	httpResp, err := c.http.Do(req, &resp)
	if err != nil {
		return nil, fmt.Errorf("upload attachment: %w", err)
	}
	if httpResp.StatusCode >= 400 {
		return nil, fmt.Errorf("upload failed with status %d", httpResp.StatusCode)
	}
	return &resp, nil
}

// DownloadAttachment streams attachment data to an io.Writer.
func (c *Client) DownloadAttachment(ctx context.Context, downloadPath string, w io.Writer) error {
	req, err := c.http.NewRequest(ctx, http.MethodGet, downloadPath, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req, nil)
	if err != nil {
		return fmt.Errorf("download attachment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	if _, err := io.Copy(w, resp.Body); err != nil {
		return fmt.Errorf("save attachment data: %w", err)
	}
	return nil
}

// DeleteAttachment deletes an attachment.
func (c *Client) DeleteAttachment(ctx context.Context, attachmentID string) error {
	path := fmt.Sprintf("/rest/api/content/%s", url.PathEscape(attachmentID))
	if _, err := c.http.Delete(ctx, path); err != nil {
		return fmt.Errorf("delete attachment %s: %w", attachmentID, err)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Labels
// ─────────────────────────────────────────────────────────────────────────────

// ListLabels lists labels on a page.
func (c *Client) ListLabels(ctx context.Context, pageID string) (*LabelList, error) {
	path := fmt.Sprintf("/rest/api/content/%s/label", url.PathEscape(pageID))
	var resp LabelList
	if _, err := c.http.Get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("list labels: %w", err)
	}
	return &resp, nil
}

// AddLabels adds labels to a page.
func (c *Client) AddLabels(ctx context.Context, pageID string, labels []string) error {
	var payload []map[string]string
	for _, l := range labels {
		payload = append(payload, map[string]string{
			"prefix": "global",
			"name":   strings.TrimSpace(l),
		})
	}
	path := fmt.Sprintf("/rest/api/content/%s/label", url.PathEscape(pageID))
	if _, err := c.http.Post(ctx, path, payload, nil); err != nil {
		return fmt.Errorf("add labels: %w", err)
	}
	return nil
}

// RemoveLabel removes a label from a page.
func (c *Client) RemoveLabel(ctx context.Context, pageID, label string) error {
	path := fmt.Sprintf("/rest/api/content/%s/label/%s", url.PathEscape(pageID), url.PathEscape(label))
	if _, err := c.http.Delete(ctx, path); err != nil {
		return fmt.Errorf("remove label %q: %w", label, err)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Search & CQL
// ─────────────────────────────────────────────────────────────────────────────

// Search searches content using free text or type/space filtering.
func (c *Client) Search(ctx context.Context, query, spaceKey, contentType string, limit int) (*SearchResult, error) {
	var cqlParts []string
	if query != "" {
		cqlParts = append(cqlParts, fmt.Sprintf("text ~ %q", query))
	}
	if spaceKey != "" {
		cqlParts = append(cqlParts, fmt.Sprintf("space = %q", spaceKey))
	}
	if contentType != "" {
		cqlParts = append(cqlParts, fmt.Sprintf("type = %q", contentType))
	}

	cql := strings.Join(cqlParts, " AND ")
	if cql == "" {
		cql = "type = page"
	}
	return c.SearchCQL(ctx, cql, limit, 0)
}

// SearchCQL runs an arbitrary Confluence Query Language (CQL) search.
func (c *Client) SearchCQL(ctx context.Context, cql string, limit, start int) (*SearchResult, error) {
	if limit <= 0 {
		limit = 25
	}
	params := url.Values{}
	params.Set("cql", cql)
	params.Set("limit", strconv.Itoa(limit))
	params.Set("expand", "version,space,history")
	if start > 0 {
		params.Set("start", strconv.Itoa(start))
	}

	path := "/rest/api/content/search?" + params.Encode()
	var resp SearchResult
	if _, err := c.http.Get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("search CQL %q: %w", cql, err)
	}
	resp.CQLQuery = cql
	return &resp, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Raw API
// ─────────────────────────────────────────────────────────────────────────────

// Raw executes a raw request against Confluence REST API.
func (c *Client) Raw(ctx context.Context, method, path string, body any, headers map[string]string) (*http.Response, []byte, error) {
	req, err := c.http.NewRequest(ctx, method, path, body)
	if err != nil {
		return nil, nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.http.Do(req, nil)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("read response: %w", err)
	}
	return resp, respBody, nil
}
