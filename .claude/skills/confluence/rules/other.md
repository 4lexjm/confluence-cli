# Raw REST API

For Confluence endpoints that do not yet have dedicated subcommands, `confluence api`
provides an authenticated escape hatch directly into the REST API.

## Usage

```bash
confluence api <path> [flags]
```

## Flags

| Flag | Short | Description |
|---|---|---|
| `--method` | `-X` | HTTP method (GET, POST, PUT, DELETE; default GET) |
| `--input` | `-d` | Raw JSON string payload |
| `--field` | `-F` | Key=value body parameter (inferred JSON type) |
| `--header` | `-H` | Custom HTTP header (Header: Value) |
| `--param` | `-P` | Query parameter appended to URL |

## Examples

### Reading data
```bash
# Query content by ID
confluence api /rest/api/content/12345 --json | jq '.title'

# Using query parameters
confluence api /rest/api/space --param limit=10 --param type=global

# Confluence Cloud v2 API
confluence api /wiki/api/v2/pages --param limit=5
```

### Mutating data
```bash
# Create a page via raw POST
confluence api /rest/api/content \
  --method POST \
  -F title="Raw API Created Page" \
  -F type="page"

# Pass full JSON payload via --input
confluence api /rest/api/content -d '{"type":"page","title":"My Page","space":{"key":"ENG"},"body":{"storage":{"value":"<p>Text</p>","representation":"storage"}}}'
```
