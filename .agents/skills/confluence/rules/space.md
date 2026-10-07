# Spaces

Spaces are the top-level organizational containers in Confluence. Every page belongs to a space.

## Commands

### List spaces
```bash
confluence space list
confluence space list --type global --limit 50
confluence space list --json | jq '.[].key'
```

### View space details
```bash
confluence space view ENG
confluence space view ENG --web # opens in browser
```

### Create a space
```bash
confluence space create \
  --key ARCH \
  --name "Architecture & Design" \
  --description "Central architecture documents and diagrams"
```

### Delete a space
> [!CAUTION]
> Space deletion removes all pages, attachments, and comments in the space.
> Requires the `--confirm` flag.

```bash
confluence space delete ARCH --confirm
```
