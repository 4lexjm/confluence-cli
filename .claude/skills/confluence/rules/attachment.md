# Attachments

Attachments represent files, images, PDFs, and assets attached to Confluence pages.

## Commands

### List attachments
```bash
confluence attachment list 123456
confluence attachment list 123456 --json | jq '.[].title'
```

### Upload an attachment
```bash
# Upload a file
confluence attachment upload 123456 --file ./diagram.png

# Upload with custom name and comment
confluence attachment upload 123456 \
  --file ./reports/perf-v2.pdf \
  --name "performance-report.pdf" \
  --comment "Initial benchmark measurements"
```

### Download an attachment
```bash
# Download to current directory using attachment title
confluence attachment download att998877

# Download to custom location
confluence attachment download att998877 --output /tmp/diagram.png
```

### Delete an attachment
```bash
confluence attachment delete att998877 --confirm
```
