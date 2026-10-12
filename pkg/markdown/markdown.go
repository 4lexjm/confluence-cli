// Package markdown provides bidirectional conversion between Markdown
// and Confluence XHTML storage format.
package markdown

import (
	"bytes"
	"fmt"
	"html"
	"regexp"
	"strings"

	nethtml "golang.org/x/net/html"
)

var (
	headingRegex  = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	boldRegex     = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	italicRegex   = regexp.MustCompile(`(?:\*([^*]+)\*|_([^_]+)_)`)
	codeInlineReg = regexp.MustCompile("`([^`]+)`")
	linkRegex     = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	bulletListReg = regexp.MustCompile(`^[-*+]\s+(.*)$`)
	numListReg    = regexp.MustCompile(`^\d+\.\s+(.*)$`)
	blockquoteReg = regexp.MustCompile(`^>\s+(.*)$`)
	horizRuleReg  = regexp.MustCompile(`^(?:---+|\*\*\*+|___+)$`)
	tableRowReg   = regexp.MustCompile(`^\|(.+)\|$`)
	tableSepReg   = regexp.MustCompile(`^\|(\s*[-:]+[-| :]*)\|$`)
)

// MarkdownToStorage converts Markdown text into Confluence XHTML storage format.
func MarkdownToStorage(md string) string {
	lines := strings.Split(strings.ReplaceAll(md, "\r\n", "\n"), "\n")
	var buf strings.Builder

	inCodeBlock := false
	codeLanguage := ""
	var codeLines []string

	inBulletList := false
	inNumList := false
	inTable := false

	flushLists := func() {
		if inBulletList {
			buf.WriteString("</ul>\n")
			inBulletList = false
		}
		if inNumList {
			buf.WriteString("</ol>\n")
			inNumList = false
		}
	}

	flushTable := func() {
		if inTable {
			buf.WriteString("</tbody></table>\n")
			inTable = false
		}
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Check fenced code block start/end
		if strings.HasPrefix(trimmed, "```") {
			if inCodeBlock {
				// End code block
				buf.WriteString("<ac:structured-macro ac:name=\"code\">\n")
				if codeLanguage != "" {
					buf.WriteString(fmt.Sprintf("<ac:parameter ac:name=\"language\">%s</ac:parameter>\n", html.EscapeString(codeLanguage)))
				}
				buf.WriteString("<ac:plain-text-body><![CDATA[")
				buf.WriteString(strings.Join(codeLines, "\n"))
				buf.WriteString("]]></ac:plain-text-body>\n</ac:structured-macro>\n")
				inCodeBlock = false
				codeLanguage = ""
				codeLines = nil
				continue
			} else {
				// Start code block
				flushLists()
				flushTable()
				inCodeBlock = true
				codeLanguage = strings.TrimPrefix(trimmed, "```")
				codeLanguage = strings.TrimSpace(codeLanguage)
				codeLines = nil
				continue
			}
		}

		if inCodeBlock {
			codeLines = append(codeLines, line)
			continue
		}

		// Empty line
		if trimmed == "" {
			flushLists()
			flushTable()
			continue
		}

		// Horizontal rule
		if horizRuleReg.MatchString(trimmed) {
			flushLists()
			flushTable()
			buf.WriteString("<hr/>\n")
			continue
		}

		// Table row
		if tableRowReg.MatchString(trimmed) {
			flushLists()
			if tableSepReg.MatchString(trimmed) {
				// Table delimiter row like |---|---|
				continue
			}
			cells := parseTableCells(trimmed)
			if !inTable {
				inTable = true
				buf.WriteString("<table><thead><tr>\n")
				for _, c := range cells {
					buf.WriteString("<th>")
					buf.WriteString(formatInline(c))
					buf.WriteString("</th>\n")
				}
				buf.WriteString("</tr></thead><tbody>\n")
			} else {
				buf.WriteString("<tr>\n")
				for _, c := range cells {
					buf.WriteString("<td>")
					buf.WriteString(formatInline(c))
					buf.WriteString("</td>\n")
				}
				buf.WriteString("</tr>\n")
			}
			continue
		} else {
			flushTable()
		}

		// Heading
		if m := headingRegex.FindStringSubmatch(trimmed); m != nil {
			flushLists()
			level := len(m[1])
			content := formatInline(m[2])
			buf.WriteString(fmt.Sprintf("<h%d>%s</h%d>\n", level, content, level))
			continue
		}

		// Bullet list
		if m := bulletListReg.FindStringSubmatch(trimmed); m != nil {
			if inNumList {
				buf.WriteString("</ol>\n")
				inNumList = false
			}
			if !inBulletList {
				buf.WriteString("<ul>\n")
				inBulletList = true
			}
			buf.WriteString(fmt.Sprintf("<li>%s</li>\n", formatInline(m[1])))
			continue
		}

		// Numbered list
		if m := numListReg.FindStringSubmatch(trimmed); m != nil {
			if inBulletList {
				buf.WriteString("</ul>\n")
				inBulletList = false
			}
			if !inNumList {
				buf.WriteString("<ol>\n")
				inNumList = true
			}
			buf.WriteString(fmt.Sprintf("<li>%s</li>\n", formatInline(m[1])))
			continue
		}

		// Blockquote
		if m := blockquoteReg.FindStringSubmatch(trimmed); m != nil {
			flushLists()
			buf.WriteString(fmt.Sprintf("<blockquote><p>%s</p></blockquote>\n", formatInline(m[1])))
			continue
		}

		// Normal paragraph
		flushLists()
		buf.WriteString(fmt.Sprintf("<p>%s</p>\n", formatInline(trimmed)))
	}

	if inCodeBlock {
		buf.WriteString("<ac:structured-macro ac:name=\"code\">\n<ac:plain-text-body><![CDATA[")
		buf.WriteString(strings.Join(codeLines, "\n"))
		buf.WriteString("]]></ac:plain-text-body>\n</ac:structured-macro>\n")
	}
	flushLists()
	flushTable()

	return strings.TrimSpace(buf.String())
}

func parseTableCells(row string) []string {
	trimmed := strings.Trim(row, "|")
	rawCells := strings.Split(trimmed, "|")
	var cells []string
	for _, c := range rawCells {
		cells = append(cells, strings.TrimSpace(c))
	}
	return cells
}

func formatInline(text string) string {
	res := html.EscapeString(text)
	// Bold: **text**
	res = boldRegex.ReplaceAllString(res, "<strong>$1</strong>")
	// Italic: *text* or _text_
	res = italicRegex.ReplaceAllStringFunc(res, func(m string) string {
		inner := strings.Trim(m, "*_")
		return "<em>" + inner + "</em>"
	})
	// Inline code: `text`
	res = codeInlineReg.ReplaceAllString(res, "<code>$1</code>")
	// Link: [text](url)
	res = linkRegex.ReplaceAllString(res, `<a href="$2">$1</a>`)
	return res
}

// StorageToMarkdown converts Confluence XHTML storage or view format to Markdown.
func StorageToMarkdown(storageHTML string) string {
	if strings.TrimSpace(storageHTML) == "" {
		return ""
	}

	doc, err := nethtml.Parse(strings.NewReader(storageHTML))
	if err != nil {
		// Fallback: strip basic tags
		return stripTags(storageHTML)
	}

	var buf bytes.Buffer
	renderNode(&buf, doc)
	result := buf.String()

	// Normalize multiple consecutive blank lines
	multipleNewlines := regexp.MustCompile(`\n{3,}`)
	result = multipleNewlines.ReplaceAllString(result, "\n\n")

	return strings.TrimSpace(result)
}

func renderNode(buf *bytes.Buffer, n *nethtml.Node) {
	if n.Type == nethtml.TextNode {
		text := n.Data
		// Unescape HTML entities
		text = html.UnescapeString(text)
		buf.WriteString(text)
		return
	}

	if n.Type != nethtml.ElementNode {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			renderNode(buf, c)
		}
		return
	}

	tag := strings.ToLower(n.Data)

	// Check Confluence macro
	if tag == "ac:structured-macro" {
		macroName := getAttr(n, "ac:name")
		if macroName == "code" {
			lang := ""
			code := ""
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Data == "ac:parameter" && getAttr(c, "ac:name") == "language" && c.FirstChild != nil {
					lang = c.FirstChild.Data
				}
				if c.Data == "ac:plain-text-body" && c.FirstChild != nil {
					code = c.FirstChild.Data
				}
			}
			buf.WriteString("\n```" + lang + "\n")
			buf.WriteString(strings.TrimRight(code, "\n"))
			buf.WriteString("\n```\n\n")
			return
		}
		if macroName == "info" || macroName == "note" || macroName == "warning" {
			buf.WriteString("\n> **" + strings.ToUpper(macroName) + ":** ")
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Data == "ac:rich-text-body" {
					var inner bytes.Buffer
					for rc := c.FirstChild; rc != nil; rc = rc.NextSibling {
						renderNode(&inner, rc)
					}
					buf.WriteString(strings.TrimSpace(inner.String()))
				}
			}
			buf.WriteString("\n\n")
			return
		}
	}

	switch tag {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		level := int(tag[1] - '0')
		prefix := strings.Repeat("#", level) + " "
		buf.WriteString("\n" + prefix)
		renderChildren(buf, n)
		buf.WriteString("\n\n")

	case "p":
		buf.WriteString("\n")
		renderChildren(buf, n)
		buf.WriteString("\n\n")

	case "strong", "b":
		buf.WriteString("**")
		renderChildren(buf, n)
		buf.WriteString("**")

	case "em", "i":
		buf.WriteString("*")
		renderChildren(buf, n)
		buf.WriteString("*")

	case "code":
		buf.WriteString("`")
		renderChildren(buf, n)
		buf.WriteString("`")

	case "pre":
		buf.WriteString("\n```\n")
		renderChildren(buf, n)
		buf.WriteString("\n```\n\n")

	case "blockquote":
		buf.WriteString("\n> ")
		var inner bytes.Buffer
		renderChildren(&inner, n)
		lines := strings.Split(strings.TrimSpace(inner.String()), "\n")
		buf.WriteString(strings.Join(lines, "\n> "))
		buf.WriteString("\n\n")

	case "hr":
		buf.WriteString("\n---\n\n")

	case "ul":
		buf.WriteString("\n")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if strings.ToLower(c.Data) == "li" {
				buf.WriteString("* ")
				var itemBuf bytes.Buffer
				renderChildren(&itemBuf, c)
				buf.WriteString(strings.TrimSpace(itemBuf.String()))
				buf.WriteString("\n")
			}
		}
		buf.WriteString("\n")

	case "ol":
		buf.WriteString("\n")
		idx := 1
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if strings.ToLower(c.Data) == "li" {
				buf.WriteString(fmt.Sprintf("%d. ", idx))
				var itemBuf bytes.Buffer
				renderChildren(&itemBuf, c)
				buf.WriteString(strings.TrimSpace(itemBuf.String()))
				buf.WriteString("\n")
				idx++
			}
		}
		buf.WriteString("\n")

	case "a":
		href := getAttr(n, "href")
		var textBuf bytes.Buffer
		renderChildren(&textBuf, n)
		text := textBuf.String()
		if href != "" {
			buf.WriteString(fmt.Sprintf("[%s](%s)", text, href))
		} else {
			buf.WriteString(text)
		}

	case "table":
		buf.WriteString("\n")
		renderTable(buf, n)
		buf.WriteString("\n")

	default:
		renderChildren(buf, n)
	}
}

func renderChildren(buf *bytes.Buffer, n *nethtml.Node) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		renderNode(buf, c)
	}
}

func renderTable(buf *bytes.Buffer, n *nethtml.Node) {
	var rows [][]string
	var collectRows func(*nethtml.Node)
	collectRows = func(curr *nethtml.Node) {
		if strings.ToLower(curr.Data) == "tr" {
			var row []string
			for cell := curr.FirstChild; cell != nil; cell = cell.NextSibling {
				cellTag := strings.ToLower(cell.Data)
				if cellTag == "th" || cellTag == "td" {
					var cellBuf bytes.Buffer
					renderChildren(&cellBuf, cell)
					row = append(row, strings.TrimSpace(cellBuf.String()))
				}
			}
			if len(row) > 0 {
				rows = append(rows, row)
			}
		}
		for c := curr.FirstChild; c != nil; c = c.NextSibling {
			collectRows(c)
		}
	}
	collectRows(n)

	if len(rows) == 0 {
		return
	}

	// First row as header
	buf.WriteString("| " + strings.Join(rows[0], " | ") + " |\n")
	var sep []string
	for range rows[0] {
		sep = append(sep, "---")
	}
	buf.WriteString("| " + strings.Join(sep, " | ") + " |\n")

	for i := 1; i < len(rows); i++ {
		buf.WriteString("| " + strings.Join(rows[i], " | ") + " |\n")
	}
}

func getAttr(n *nethtml.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

func stripTags(s string) string {
	re := regexp.MustCompile("<[^>]*>")
	return re.ReplaceAllString(s, "")
}
