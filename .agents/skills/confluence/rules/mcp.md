# Model Context Protocol (MCP) Server

The `confluence mcp` command starts a Model Context Protocol server that exposes
Confluence operations as structured tools to MCP-compatible AI agents (Claude Code,
Cursor, Gemini CLI, Copilot, and others).

## Command

```bash
# Start MCP server over stdio
confluence mcp serve

# Run in read-only mode (prevents create/edit/delete mutations)
confluence mcp serve --read-only

# Bind to specific context and default space
confluence mcp serve --context prod-cloud --space ENG
```

## Agent Configuration

### Claude Code (`~/.claude/settings.json`)

```json
{
  "mcpServers": {
    "confluence": {
      "command": "confluence",
      "args": ["mcp", "serve", "--context", "prod-cloud"]
    }
  }
}
```

### Cursor (`.cursor/mcp.json`)

```json
{
  "mcpServers": {
    "confluence": {
      "command": "confluence",
      "args": ["mcp", "serve"]
    }
  }
}
```

### Headless / CI Environment Variables

```json
{
  "mcpServers": {
    "confluence": {
      "command": "confluence",
      "args": ["mcp", "serve"],
      "env": {
        "CONFLUENCE_HOST": "https://myorg.atlassian.net",
        "CONFLUENCE_EMAIL": "agent@myorg.com",
        "CONFLUENCE_TOKEN": "${CONFLUENCE_TOKEN}",
        "CONFLUENCE_SPACE": "ENG"
      }
    }
  }
}
```

## Tools Exposed

| Tool | Mode | Description |
|---|---|---|
| `confluence_search` | Read-only | Search content by text or CQL expression |
| `confluence_get_page` | Read-only | Retrieve page title, body (markdown or storage), and metadata |
| `confluence_list_pages` | Read-only | List pages in a space |
| `confluence_list_spaces` | Read-only | List available spaces |
| `confluence_get_space` | Read-only | Get space details |
| `confluence_list_comments` | Read-only | List comments on a page |
| `confluence_list_attachments` | Read-only | List attachments on a page |
| `confluence_get_page_history` | Read-only | View revision history for a page |
| `confluence_create_page` | Read/Write | Create a new page with Markdown or storage format |
| `confluence_update_page` | Read/Write | Update an existing page's title or body |
| `confluence_delete_page` | Read/Write | Delete / trash a page |
| `confluence_add_comment` | Read/Write | Add a comment to a page |
| `confluence_raw_api` | Read/Write | Execute direct Confluence REST API calls |
