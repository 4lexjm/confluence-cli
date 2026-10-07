# Pages

Pages are the primary documentation unit in Confluence.

## Commands

### List pages
```bash
# List pages in the default space
confluence page list

# List pages in a specific space
confluence page list --space ENG --limit 50

# Filter by title
confluence page list --title "Roadmap"
```

### View a page
```bash
# View by numeric page ID (default renders as clean Markdown)
confluence page view 123456

# View by title within a space
confluence page view "Architecture Overview" --space ENG

# View raw storage XHTML format
confluence page view 123456 --format storage

# Include comments on the page
confluence page view 123456 --comments

# Open in browser
confluence page view 123456 --web
```

### Create a page
AI agents write standard Markdown (headers, lists, tables, code blocks, links).
`confluence` automatically converts Markdown into valid Confluence storage XHTML.

```bash
# Create with inline Markdown
confluence page create \
  --space ENG \
  --title "Postmortem - Incident #1024" \
  --body "## Summary\nRoot cause was a database connection leak."

# Create under a parent page
confluence page create \
  --space ENG \
  --parent 123456 \
  --title "Child Page" \
  --body "Details here"

# Create from a local Markdown file
confluence page create \
  --space ENG \
  --title "Design Doc" \
  --file ./docs/design.md
```

### Edit a page
Editing increments the version number. Always include `--version-comment` to
track changes in Confluence's page history.

```bash
# Update title and body
confluence page edit 123456 \
  --title "Updated Architecture Guide" \
  --body "New content" \
  --version-comment "Updated diagram links and endpoints"

# Update body from a file
confluence page edit 123456 \
  --file ./updated-doc.md \
  --version-comment "Synced with repo v2.1"
```

### Move a page
```bash
# Reparent under another page
confluence page move 123456 --parent 987654

# Move to another space
confluence page move 123456 --space ARCH
```

### Copy a page
```bash
confluence page copy 123456 --title "Template Clone" --space ENG
```

### Page history
```bash
confluence page history 123456 --limit 10
```

### Labels
```bash
confluence page label list 123456
confluence page label add 123456 architecture devops approved
confluence page label remove 123456 draft
```

### Delete a page
> [!CAUTION]
> Deleting a page moves it to the trash. Requires `--confirm`.

```bash
confluence page delete 123456 --confirm
```
