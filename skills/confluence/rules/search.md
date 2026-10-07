# Search and CQL

Confluence Query Language (CQL) allows searching content across all spaces
using rich filter criteria.

## Basic Search

```bash
# Search for keyword
confluence search "microservices"

# Search in a specific space
confluence search "authentication" --space SEC

# Filter by content type
confluence search "deployment" --type blogpost
confluence search "architecture diagram" --type attachment
```

## Confluence Query Language (CQL)

Pass `--cql` to execute advanced queries:

### Common CQL Fields
- `space = "KEY"`: Space key filter
- `type = page|blogpost|attachment|comment`: Content type
- `title ~ "keyword"`: Title text match
- `text ~ "keyword"`: Body and title full-text search
- `creator = currentUser()` or `creator = "username"`
- `contributor = currentUser()`
- `label = "keyword"`
- `lastModified >= now("-7d")`: Modified recently
- `created >= "2024-01-01"`

### Examples

```bash
# Modified in the last 7 days in space ENG
confluence search --cql 'space = "ENG" AND type = page AND lastModified >= now("-7d")'

# Pages labeled 'architecture' created this year
confluence search --cql 'type = page AND label = "architecture" AND created >= "2024-01-01"'

# Output IDs for batch operations
confluence search --cql 'space = "ARCH" AND type = page' --json | jq '.[].id'
```
