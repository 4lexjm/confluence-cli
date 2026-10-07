# Setup and Common Workflows

This guide covers typical tasks and workflows when interacting with Confluence via `confluence`.

## Quick Start

### 1. Authenticate

#### Confluence Cloud (Atlassian Cloud)
```bash
confluence auth login https://myorg.atlassian.net \
  --email agent@myorg.com \
  --token <API_TOKEN> \
  --space ENG
```

#### Confluence Data Center (Personal Access Token)
```bash
confluence auth login https://confluence.example.com \
  --token <PAT> \
  --space DEV
```

#### Confluence Data Center (Browser SSO)
```bash
confluence auth login https://confluence.example.com --browser
```

### 2. Verify Authentication

```bash
confluence auth status
```

---

## Core AI Agent Workflows

### 1. Reading documentation

```bash
# Search for pages mentioning architecture
confluence page search "microservices architecture" --space ENG

# View a page by ID (rendered in clean Markdown)
confluence page view 123456

# View a page by title within a space
confluence page view "System Overview" --space ENG

# Include comments on the page
confluence page view 123456 --comments
```

### 2. Creating documentation

Agents can provide body text directly as Markdown or pass a file:

```bash
# Inline Markdown content
confluence page create \
  --space ENG \
  --title "Sprint 42 Retrospective" \
  --body "## What went well\n- Fast deployment\n\n## Action items\n- Update docs"

# From a markdown file
confluence page create \
  --space ENG \
  --title "API Design Guidelines" \
  --file ./docs/api-guidelines.md \
  --parent 10001
```

### 3. Updating an existing page

```bash
confluence page edit 123456 \
  --body "Updated content with newest design specs" \
  --version-comment "Added section on authentication"
```

### 4. Advanced CQL searching

```bash
# Find pages in space ENG updated in the last 7 days
confluence search --cql 'space = "ENG" AND type = page AND lastModified >= now("-7d")' --json

# Find all documents with label "architecture"
confluence search --cql 'label = "architecture"' --limit 50
```

### 5. Managing comments and feedback

```bash
# List discussion on a page
confluence comment list 123456

# Post a review comment
confluence comment add 123456 --body "LGTM. Verified the endpoint contracts."
```
