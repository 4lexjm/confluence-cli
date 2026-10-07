# confluence-cli

`confluence` is a unified CLI for **Confluence Cloud** and **Confluence Data Center / Server**.
It is optimised for **AI agents** — providing structured JSON/YAML output, automatic Markdown ↔ Confluence XHTML conversion, a built-in Model Context Protocol (MCP) server, and agent skills — while being equally productive and intuitive for humans in the terminal.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/atlassian/confluence-cli)](https://goreportcard.com/report/github.com/atlassian/confluence-cli)

---

## Features

- **Pages** — list, view (in clean Markdown or raw XHTML), create, edit, move, copy, delete, and view history
- **Markdown Conversion** — bidirectional translation: write standard Markdown (`#`, `**`, `*`, ` ``` `, tables, links) and the CLI converts it to valid Confluence storage XHTML
- **Spaces** — list, view details, create, and delete spaces
- **Comments** — list, add top-level or threaded reply comments, edit, and delete
- **Attachments** — upload, download, list, and delete page attachments
- **Search & CQL** — full-text search and advanced Confluence Query Language (`--cql`)
- **Labels** — list, add, and remove page labels
- **Blog Posts** — list, view, and publish blog posts
- **MCP Server** — built-in `confluence mcp serve` exposing tools to Claude Code, Cursor, Gemini CLI, and Copilot
- **Headless / CI Mode** — zero-config operation via `CONFLUENCE_TOKEN`, `CONFLUENCE_HOST`, `CONFLUENCE_EMAIL`, `CONFLUENCE_SPACE`
- **Structured Output** — global `--json`, `--yaml`, and `--jq` filtering on every command
- **Raw API Escape Hatch** — `confluence api <path>` for endpoints not yet wrapped

---

## Installation

| Platform | Command |
|---|---|
| Go | `go install github.com/atlassian/confluence-cli/cmd/confluence@latest` |
| Local Build | `make build` (produces `./bin/confluence`) |

---

## Quick Start

### 1. Authenticate

```bash
# Confluence Cloud (API token — https://id.atlassian.com/manage-profile/security/api-tokens)
confluence auth login https://myorg.atlassian.net \
  --email me@example.com \
  --token <API_TOKEN> \
  --space ENG

# Confluence Data Center (Personal Access Token)
confluence auth login https://confluence.example.com \
  --token <PAT> \
  --space DEV

# Confluence Data Center (Browser SSO capture)
confluence auth login https://confluence.example.com --browser
```

### 2. Check Connection

```bash
confluence auth status
confluence auth doctor
```

### 3. Start Working

```bash
# Search for pages
confluence page search "Architecture"

# View a page in clean Markdown
confluence page view 123456

# Create a page in your default space
confluence page create --title "Incident Report" --body "## Summary\nAll services operational."

# Edit an existing page
confluence page edit 123456 --body "Updated docs" --version-comment "Revamped guide"
```

---

## Common Workflows

### 1. Reading & Inspecting Content

```bash
# List pages in the active space
confluence page list

# Read page content as formatted Markdown
confluence page view 123456

# Read page with discussion comments
confluence page view 123456 --comments

# Inspect version history
confluence page history 123456
```

### 2. Creating & Editing Pages

AI agents and humans can write standard Markdown:

```bash
# Create a new page with inline Markdown
confluence page create \
  --space ENG \
  --title "Sprint 42 Plan" \
  --body "# Objectives\n- Migrate Auth service\n- Setup telemetry"

# Create a page from a Markdown file
confluence page create \
  --space ENG \
  --title "Deployment Guide" \
  --file ./docs/deploy.md \
  --parent 10001

# Update a page from file
confluence page edit 123456 \
  --file ./docs/updated.md \
  --version-comment "Updated deployment checklist"
```

### 3. CQL Search

```bash
# Free text search
confluence search "microservices"

# Search by space and type
confluence search "release notes" --space ENG --type page

# Advanced CQL query
confluence search --cql 'space = "ENG" AND type = page AND lastModified >= now("-7d")'

# Filter by label
confluence search --cql 'type = page AND label = "security-review"'
```

### 4. Comments & Attachments

```bash
# List comments on a page
confluence comment list 123456

# Add a comment
confluence comment add 123456 --body "LGTM. Ready to publish."

# Upload an architecture diagram
confluence attachment upload 123456 --file ./diagram.png --comment "v2 architecture"

# Download an attachment
confluence attachment download att987654 --output ./downloaded.png
```

### 5. Multi-Instance Context Management

```bash
# Create a context for Cloud
confluence context create cloud-prod \
  --host https://myorg.atlassian.net \
  --space ENG \
  --set-active

# Create a context for Data Center
confluence context create dc-staging \
  --host https://confluence.staging.internal \
  --space DEV

# Switch between contexts
confluence context use dc-staging
confluence context list
```

### 6. Automation & Structured Output

Every command supports `--json`, `--yaml`, and `--jq`:

```bash
# Extract IDs of search results
confluence search --cql 'space = "ENG"' --json | jq '.[].id'

# In-line jq query
confluence page list --jq '.[].title'

# YAML export
confluence space list --yaml
```

### 7. MCP Server for AI Agents

`confluence` includes a native Model Context Protocol (MCP) server:

```bash
# Start MCP server (stdio transport)
confluence mcp serve

# Read-only mode (safe for untrusted agents)
confluence mcp serve --read-only
```

#### Configuration in Claude Code (`~/.claude/settings.json`)

```json
{
  "mcpServers": {
    "confluence": {
      "command": "confluence",
      "args": ["mcp", "serve", "--context", "cloud-prod"]
    }
  }
}
```

#### Configuration in Cursor (`.cursor/mcp.json`)

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

---

## Headless & CI Environments

In automated pipelines or background agents, specify environment variables:

```bash
export CONFLUENCE_HOST="https://myorg.atlassian.net"
export CONFLUENCE_EMAIL="bot@myorg.com"
export CONFLUENCE_TOKEN="<API_TOKEN>"
export CONFLUENCE_SPACE="ENG"

confluence page create --title "Automated Build" --file ./report.md
```

---

## Raw API Escape Hatch

Access any Confluence REST API directly:

```bash
# Query v1 REST endpoint
confluence api /rest/api/space --param limit=10

# Query v2 REST endpoint (Cloud)
confluence api /wiki/api/v2/pages --param limit=5

# POST raw payload
confluence api /rest/api/content --method POST -d '{"type":"page","title":"Raw Page","space":{"key":"ENG"},"body":{"storage":{"value":"<p>Content</p>","representation":"storage"}}}'
```

---

## License

MIT
