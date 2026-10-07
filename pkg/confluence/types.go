package confluence

import "time"

// User represents a Confluence user.
type User struct {
	AccountID   string `json:"accountId,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	PublicName  string `json:"publicName,omitempty"`
	Username    string `json:"username,omitempty"`
	UserKey     string `json:"userKey,omitempty"`
}

// Space represents a Confluence space.
type Space struct {
	ID          any               `json:"id,omitempty"`
	Key         string            `json:"key"`
	Name        string            `json:"name"`
	Type        string            `json:"type,omitempty"`
	Status      string            `json:"status,omitempty"`
	Description *SpaceDescription `json:"description,omitempty"`
	Homepage    *ContentSummary   `json:"homepage,omitempty"`
	Links       map[string]string `json:"_links,omitempty"`
}

// SpaceDescription holds the space description.
type SpaceDescription struct {
	Plain *ContentBodyValue `json:"plain,omitempty"`
	View  *ContentBodyValue `json:"view,omitempty"`
}

// ContentSummary is a lightweight reference to a page or content item.
type ContentSummary struct {
	ID    string `json:"id"`
	Type  string `json:"type,omitempty"`
	Title string `json:"title,omitempty"`
}

// Content represents a Confluence page, blog post, comment, or attachment.
type Content struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"` // "page", "blogpost", "comment", "attachment"
	Status    string            `json:"status,omitempty"`
	Title     string            `json:"title"`
	Space     *SpaceSummary     `json:"space,omitempty"`
	History   *ContentHistory   `json:"history,omitempty"`
	Version   *ContentVersion   `json:"version,omitempty"`
	Ancestors []*ContentSummary `json:"ancestors,omitempty"`
	Body      *ContentBody      `json:"body,omitempty"`
	Metadata  *ContentMetadata  `json:"metadata,omitempty"`
	Links     map[string]string `json:"_links,omitempty"`
}

// SpaceSummary holds minimal space details.
type SpaceSummary struct {
	ID   any    `json:"id,omitempty"`
	Key  string `json:"key"`
	Name string `json:"name,omitempty"`
}

// ContentHistory contains creation and update metadata.
type ContentHistory struct {
	Latest      bool            `json:"latest,omitempty"`
	CreatedBy   *User           `json:"createdBy,omitempty"`
	CreatedDate string          `json:"createdDate,omitempty"`
	LastUpdated *ContentVersion `json:"lastUpdated,omitempty"`
}

// ContentVersion represents the version information for content.
type ContentVersion struct {
	Number  int    `json:"number"`
	Message string `json:"message,omitempty"`
	When    string `json:"when,omitempty"`
	By      *User  `json:"by,omitempty"`
	Minor   bool   `json:"minorEdit,omitempty"`
}

// ContentBody holds body content representations.
type ContentBody struct {
	Storage *ContentBodyValue `json:"storage,omitempty"`
	View    *ContentBodyValue `json:"view,omitempty"`
	Export  *ContentBodyValue `json:"export_view,omitempty"`
}

// ContentBodyValue represents body value and representation format.
type ContentBodyValue struct {
	Value          string `json:"value"`
	Representation string `json:"representation,omitempty"`
}

// ContentMetadata holds labels and custom attributes.
type ContentMetadata struct {
	Labels *LabelList `json:"labels,omitempty"`
}

// LabelList wraps a list of labels.
type LabelList struct {
	Results []*Label `json:"results,omitempty"`
	Size    int      `json:"size,omitempty"`
}

// Label represents a Confluence label.
type Label struct {
	Prefix string `json:"prefix,omitempty"`
	Name   string `json:"name"`
	ID     string `json:"id,omitempty"`
}

// Attachment represents a Confluence attachment.
type Attachment struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Title      string            `json:"title"`
	Metadata   *AttachMetadata   `json:"metadata,omitempty"`
	Extensions *AttachExtensions `json:"extensions,omitempty"`
	Version    *ContentVersion   `json:"version,omitempty"`
	Links      map[string]string `json:"_links,omitempty"`
}

// AttachMetadata holds media type and comment.
type AttachMetadata struct {
	MediaType string `json:"mediaType,omitempty"`
	Comment   string `json:"comment,omitempty"`
}

// AttachExtensions contains file metadata.
type AttachExtensions struct {
	MediaType string `json:"mediaType,omitempty"`
	FileSize  int64  `json:"fileSize,omitempty"`
}

// ListResponse is the generic paginated response from Confluence v1 REST API.
type ListResponse[T any] struct {
	Results   []T               `json:"results"`
	Start     int               `json:"start"`
	Limit     int               `json:"limit"`
	Size      int               `json:"size"`
	TotalSize int               `json:"totalSize,omitempty"`
	Links     map[string]string `json:"_links,omitempty"`
}

// SearchResult represents CQL or text search results.
type SearchResult struct {
	Results   []*Content        `json:"results"`
	Start     int               `json:"start"`
	Limit     int               `json:"limit"`
	Size      int               `json:"size"`
	TotalSize int               `json:"totalSize,omitempty"`
	CQLQuery  string            `json:"cqlQuery,omitempty"`
	Links     map[string]string `json:"_links,omitempty"`
}

// VersionHistory represents history list.
type VersionHistory struct {
	LastUpdated *ContentVersion   `json:"lastUpdated,omitempty"`
	Previous    *ContentVersion   `json:"previousVersion,omitempty"`
	Next        *ContentVersion   `json:"nextVersion,omitempty"`
	Results     []*ContentVersion `json:"results,omitempty"`
}

// WebUIURL returns the browser URL for a content item given base URL.
func (c *Content) WebUIURL(baseURL string) string {
	if c == nil || c.Links == nil {
		return ""
	}
	webui := c.Links["webui"]
	if webui == "" {
		return ""
	}
	if len(baseURL) > 0 && baseURL[len(baseURL)-1] == '/' && len(webui) > 0 && webui[0] == '/' {
		return baseURL[:len(baseURL)-1] + webui
	}
	return baseURL + webui
}

// CreationTime returns parsed created time if available.
func (c *Content) CreationTime() (time.Time, error) {
	if c == nil || c.History == nil || c.History.CreatedDate == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, c.History.CreatedDate)
}
