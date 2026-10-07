package markdown

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarkdownToStorage(t *testing.T) {
	input := `# Title

This is a paragraph with **bold** and *italic* and ` + "`code`" + ` and [link](https://example.com).

## Subtitle

- Item 1
- Item 2

1. First
2. Second

> This is a quote

` + "```go\npackage main\n\nfunc main() {}\n```" + `

| Name | Role |
| --- | --- |
| Alice | Admin |
| Bob | Dev |
`

	storage := MarkdownToStorage(input)

	assert.Contains(t, storage, "<h1>Title</h1>")
	assert.Contains(t, storage, "<strong>bold</strong>")
	assert.Contains(t, storage, "<em>italic</em>")
	assert.Contains(t, storage, "<code>code</code>")
	assert.Contains(t, storage, `<a href="https://example.com">link</a>`)
	assert.Contains(t, storage, "<h2>Subtitle</h2>")
	assert.Contains(t, storage, "<ul>")
	assert.Contains(t, storage, "<li>Item 1</li>")
	assert.Contains(t, storage, "<ol>")
	assert.Contains(t, storage, "<li>First</li>")
	assert.Contains(t, storage, "<blockquote><p>This is a quote</p></blockquote>")
	assert.Contains(t, storage, `<ac:structured-macro ac:name="code">`)
	assert.Contains(t, storage, `<ac:parameter ac:name="language">go</ac:parameter>`)
	assert.Contains(t, storage, `func main() {}`)
	assert.Contains(t, storage, "<table><thead><tr>")
	assert.Contains(t, storage, "<th>Name</th>")
	assert.Contains(t, storage, "<td>Alice</td>")
}

func TestStorageToMarkdown(t *testing.T) {
	storage := `<h1>Main Header</h1>
<p>Hello <strong>world</strong>! Check out this <a href="https://example.com">site</a>.</p>
<h2>Code Sample</h2>
<ac:structured-macro ac:name="code">
<ac:parameter ac:name="language">python</ac:parameter>
<ac:plain-text-body><![CDATA[def hello():
    print("hi")]]></ac:plain-text-body>
</ac:structured-macro>
<ul>
<li>Bullet A</li>
<li>Bullet B</li>
</ul>
<table>
<tr><th>Col 1</th><th>Col 2</th></tr>
<tr><td>Val 1</td><td>Val 2</td></tr>
</table>`

	md := StorageToMarkdown(storage)

	assert.Contains(t, md, "# Main Header")
	assert.Contains(t, md, "Hello **world**!")
	assert.Contains(t, md, "[site](https://example.com)")
	assert.Contains(t, md, "## Code Sample")
	assert.True(t, strings.Contains(md, "```python") || strings.Contains(md, "```"))
	assert.Contains(t, md, "def hello():")
	assert.Contains(t, md, "* Bullet A")
	assert.Contains(t, md, "| Col 1 | Col 2 |")
	assert.Contains(t, md, "| Val 1 | Val 2 |")
}
