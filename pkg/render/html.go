package render

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"strings"

	"vexil/internal/domain"
	"vexil/internal/ports"
)

// Ensure HTMLRenderer implements ports.Renderer.
var _ ports.Renderer = (*HTMLRenderer)(nil)

// HTMLOption configures an HTMLRenderer instance.
type HTMLOption func(*HTMLRenderer)

// HTMLRenderer renders a Vexil AST into semantic HTML5 markup.
type HTMLRenderer struct{}

// NewHTMLRenderer creates a new HTMLRenderer instance with optional configurations.
func NewHTMLRenderer(opts ...HTMLOption) *HTMLRenderer {
	r := &HTMLRenderer{}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Render serializes the given Document AST into semantic HTML5 written to w.
func (r *HTMLRenderer) Render(doc *domain.Document, w io.Writer) error {
	if doc == nil {
		return fmt.Errorf("render html: %w", ErrNilDocument)
	}
	if w == nil {
		return fmt.Errorf("render html: %w", ErrNilWriter)
	}

	for _, child := range doc.Children {
		if err := r.RenderNode(child, w); err != nil {
			return fmt.Errorf("render node: %w", err)
		}
	}

	return nil
}

// RenderToString serializes the given Document AST into an HTML string.
func (r *HTMLRenderer) RenderToString(doc *domain.Document) (string, error) {
	var buf bytes.Buffer
	if err := r.Render(doc, &buf); err != nil {
		return "", fmt.Errorf("render to string: %w", err)
	}
	return buf.String(), nil
}

// RenderNode renders an individual AST node to w.
func (r *HTMLRenderer) RenderNode(node domain.Node, w io.Writer) error {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *domain.Heading:
		return r.renderHeading(n, w)
	case *domain.Paragraph:
		return r.renderParagraph(n, w)
	case *domain.UnorderedList:
		return r.renderUnorderedList(n, w)
	case *domain.OrderedList:
		return r.renderOrderedList(n, w)
	case *domain.TaskList:
		return r.renderTaskList(n, w)
	case *domain.Callout:
		return r.renderCallout(n, w)
	case *domain.Blockquote:
		return r.renderBlockquote(n, w)
	case *domain.CodeFence:
		return r.renderCodeFence(n, w)
	case *domain.Table:
		return r.renderTable(n, w)
	case *domain.Divider:
		return r.renderDivider(n, w)
	case domain.InlineNode:
		return r.renderInline(n, w)
	default:
		return fmt.Errorf("unsupported node type %q: %w", node.Type(), domain.ErrUnknownNodeType)
	}
}

func (r *HTMLRenderer) renderHeading(h *domain.Heading, w io.Writer) error {
	level := h.Level
	if level < 1 {
		level = 1
	} else if level > 4 {
		level = 4
	}

	var inlinesBuf bytes.Buffer
	if err := r.renderInlines(h.Children, &inlinesBuf); err != nil {
		return fmt.Errorf("render heading inlines: %w", err)
	}

	idAttr := ""
	if h.Meta != nil {
		if id, ok := h.Meta["id"]; ok && id != "" {
			idAttr = fmt.Sprintf(` id="%s"`, html.EscapeString(id))
		}
	}

	if _, err := fmt.Fprintf(w, "<h%d%s>%s</h%d>\n", level, idAttr, inlinesBuf.String(), level); err != nil {
		return fmt.Errorf("write heading: %w", err)
	}
	return nil
}

func (r *HTMLRenderer) renderParagraph(p *domain.Paragraph, w io.Writer) error {
	var inlinesBuf bytes.Buffer
	if err := r.renderInlines(p.Children, &inlinesBuf); err != nil {
		return fmt.Errorf("render paragraph inlines: %w", err)
	}

	if _, err := fmt.Fprintf(w, "<p>%s</p>\n", inlinesBuf.String()); err != nil {
		return fmt.Errorf("write paragraph: %w", err)
	}
	return nil
}

type listEntry struct {
	level    int
	checked  bool
	children []domain.InlineNode
}

func (r *HTMLRenderer) renderListStructure(entries []listEntry, tag string, isTask bool, w io.Writer) error {
	if len(entries) == 0 {
		if _, err := fmt.Fprintf(w, "<%s>\n</%s>\n", tag, tag); err != nil {
			return fmt.Errorf("write empty list: %w", err)
		}
		return nil
	}

	if _, err := fmt.Fprintf(w, "<%s>\n", tag); err != nil {
		return fmt.Errorf("write list open: %w", err)
	}

	currentLevel := 0

	for i, entry := range entries {
		lvl := entry.level
		if lvl < 0 {
			lvl = 0
		}

		if i == 0 {
			currentLevel = lvl
			for k := 0; k < currentLevel; k++ {
				if _, err := fmt.Fprintf(w, "<li>\n<%s>\n", tag); err != nil {
					return fmt.Errorf("write initial nested list open: %w", err)
				}
			}
			if _, err := io.WriteString(w, "<li>"); err != nil {
				return fmt.Errorf("write first list item open: %w", err)
			}
			if err := r.renderItemBody(entry, isTask, w); err != nil {
				return fmt.Errorf("render item body: %w", err)
			}
		} else {
			if lvl > currentLevel {
				for k := currentLevel; k < lvl; k++ {
					if _, err := fmt.Fprintf(w, "\n<%s>\n<li>", tag); err != nil {
						return fmt.Errorf("write nested list open: %w", err)
					}
				}
				currentLevel = lvl
				if err := r.renderItemBody(entry, isTask, w); err != nil {
					return fmt.Errorf("render item body: %w", err)
				}
			} else if lvl < currentLevel {
				if _, err := io.WriteString(w, "</li>\n"); err != nil {
					return fmt.Errorf("write list item close: %w", err)
				}
				for k := currentLevel; k > lvl; k-- {
					if _, err := fmt.Fprintf(w, "</%s>\n</li>\n", tag); err != nil {
						return fmt.Errorf("write nested list close: %w", err)
					}
				}
				currentLevel = lvl
				if _, err := io.WriteString(w, "<li>"); err != nil {
					return fmt.Errorf("write list item open: %w", err)
				}
				if err := r.renderItemBody(entry, isTask, w); err != nil {
					return fmt.Errorf("render item body: %w", err)
				}
			} else {
				if _, err := io.WriteString(w, "</li>\n<li>"); err != nil {
					return fmt.Errorf("write sibling item separator: %w", err)
				}
				if err := r.renderItemBody(entry, isTask, w); err != nil {
					return fmt.Errorf("render item body: %w", err)
				}
			}
		}
	}

	if _, err := io.WriteString(w, "</li>\n"); err != nil {
		return fmt.Errorf("write last item close: %w", err)
	}

	for k := currentLevel; k > 0; k-- {
		if _, err := fmt.Fprintf(w, "</%s>\n</li>\n", tag); err != nil {
			return fmt.Errorf("write remaining nested close: %w", err)
		}
	}

	if _, err := fmt.Fprintf(w, "</%s>\n", tag); err != nil {
		return fmt.Errorf("write list close: %w", err)
	}

	return nil
}

func (r *HTMLRenderer) renderItemBody(entry listEntry, isTask bool, w io.Writer) error {
	if isTask {
		checkbox := `<input type="checkbox" disabled /> `
		if entry.checked {
			checkbox = `<input type="checkbox" disabled checked /> `
		}
		if _, err := io.WriteString(w, checkbox); err != nil {
			return fmt.Errorf("write checkbox: %w", err)
		}
	}
	if err := r.renderInlines(entry.children, w); err != nil {
		return fmt.Errorf("render item inlines: %w", err)
	}
	return nil
}

func (r *HTMLRenderer) renderUnorderedList(u *domain.UnorderedList, w io.Writer) error {
	entries := make([]listEntry, len(u.Items))
	for i, item := range u.Items {
		entries[i] = listEntry{
			level:    item.Level,
			children: item.Children,
		}
	}
	return r.renderListStructure(entries, "ul", false, w)
}

func (r *HTMLRenderer) renderOrderedList(o *domain.OrderedList, w io.Writer) error {
	entries := make([]listEntry, len(o.Items))
	for i, item := range o.Items {
		entries[i] = listEntry{
			level:    item.Level,
			children: item.Children,
		}
	}
	return r.renderListStructure(entries, "ol", false, w)
}

func (r *HTMLRenderer) renderTaskList(t *domain.TaskList, w io.Writer) error {
	entries := make([]listEntry, len(t.Items))
	for i, item := range t.Items {
		entries[i] = listEntry{
			level:    item.Level,
			checked:  item.Checked,
			children: item.Children,
		}
	}
	return r.renderListStructure(entries, "ul", true, w)
}

func (r *HTMLRenderer) renderCallout(c *domain.Callout, w io.Writer) error {
	calloutClass := html.EscapeString(c.CalloutType)
	var bodyBuf bytes.Buffer
	if err := r.renderInlines(c.Children, &bodyBuf); err != nil {
		return fmt.Errorf("render callout body inlines: %w", err)
	}

	var titleHTML string
	if c.Title != "" {
		titleHTML = fmt.Sprintf(`<div class="callout-title">%s</div>`, html.EscapeString(c.Title))
	}

	calloutHTML := fmt.Sprintf(`<aside class="callout callout-%s">%s<div class="callout-body">%s</div></aside>`+"\n",
		calloutClass, titleHTML, bodyBuf.String())

	if _, err := io.WriteString(w, calloutHTML); err != nil {
		return fmt.Errorf("write callout: %w", err)
	}
	return nil
}

func (r *HTMLRenderer) renderBlockquote(b *domain.Blockquote, w io.Writer) error {
	var inlinesBuf bytes.Buffer
	if err := r.renderInlines(b.Children, &inlinesBuf); err != nil {
		return fmt.Errorf("render blockquote inlines: %w", err)
	}

	if _, err := fmt.Fprintf(w, "<blockquote>%s</blockquote>\n", inlinesBuf.String()); err != nil {
		return fmt.Errorf("write blockquote: %w", err)
	}
	return nil
}

func (r *HTMLRenderer) renderCodeFence(cf *domain.CodeFence, w io.Writer) error {
	escapedContent := html.EscapeString(cf.Content)

	classAttr := ""
	if cf.Lang != "" {
		classAttr = fmt.Sprintf(` class="language-%s"`, html.EscapeString(cf.Lang))
	}

	idAttr := ""
	if cf.Identifier != "" {
		idAttr = fmt.Sprintf(` data-id="%s"`, html.EscapeString(cf.Identifier))
	}

	codeTag := fmt.Sprintf("<code%s%s>", classAttr, idAttr)
	if _, err := fmt.Fprintf(w, "<pre>%s%s</code></pre>\n", codeTag, escapedContent); err != nil {
		return fmt.Errorf("write code fence: %w", err)
	}
	return nil
}

func (r *HTMLRenderer) renderTable(t *domain.Table, w io.Writer) error {
	var sb strings.Builder
	sb.WriteString("<table>")

	if len(t.Headers) > 0 {
		sb.WriteString("<thead><tr>")
		for _, h := range t.Headers {
			sb.WriteString("<th>")
			sb.WriteString(html.EscapeString(h))
			sb.WriteString("</th>")
		}
		sb.WriteString("</tr></thead>")
	}

	if len(t.Rows) > 0 {
		sb.WriteString("<tbody>")
		for _, row := range t.Rows {
			sb.WriteString("<tr>")
			for _, cell := range row {
				sb.WriteString("<td>")
				var cellBuf bytes.Buffer
				if err := r.renderInlines(cell, &cellBuf); err != nil {
					return fmt.Errorf("render table cell inlines: %w", err)
				}
				sb.WriteString(cellBuf.String())
				sb.WriteString("</td>")
			}
			sb.WriteString("</tr>")
		}
		sb.WriteString("</tbody>")
	}

	sb.WriteString("</table>\n")

	if _, err := io.WriteString(w, sb.String()); err != nil {
		return fmt.Errorf("write table: %w", err)
	}
	return nil
}

func (r *HTMLRenderer) renderDivider(_ *domain.Divider, w io.Writer) error {
	if _, err := io.WriteString(w, "<hr />\n"); err != nil {
		return fmt.Errorf("write divider: %w", err)
	}
	return nil
}

func (r *HTMLRenderer) renderInlines(inlines []domain.InlineNode, w io.Writer) error {
	for _, inline := range inlines {
		if err := r.renderInline(inline, w); err != nil {
			return fmt.Errorf("render inline node: %w", err)
		}
	}
	return nil
}

func (r *HTMLRenderer) renderInline(node domain.InlineNode, w io.Writer) error {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *domain.Text:
		if _, err := io.WriteString(w, html.EscapeString(n.Value)); err != nil {
			return fmt.Errorf("write text: %w", err)
		}
	case *domain.Strong:
		if _, err := io.WriteString(w, "<strong>"); err != nil {
			return fmt.Errorf("write strong open: %w", err)
		}
		if err := r.renderInlines(n.Children, w); err != nil {
			return err
		}
		if _, err := io.WriteString(w, "</strong>"); err != nil {
			return fmt.Errorf("write strong close: %w", err)
		}
	case *domain.Emphasis:
		if _, err := io.WriteString(w, "<em>"); err != nil {
			return fmt.Errorf("write emphasis open: %w", err)
		}
		if err := r.renderInlines(n.Children, w); err != nil {
			return err
		}
		if _, err := io.WriteString(w, "</em>"); err != nil {
			return fmt.Errorf("write emphasis close: %w", err)
		}
	case *domain.InlineCode:
		if _, err := fmt.Fprintf(w, "<code>%s</code>", html.EscapeString(n.Value)); err != nil {
			return fmt.Errorf("write inline code: %w", err)
		}
	case *domain.Strikethrough:
		if _, err := io.WriteString(w, "<s>"); err != nil {
			return fmt.Errorf("write strikethrough open: %w", err)
		}
		if err := r.renderInlines(n.Children, w); err != nil {
			return err
		}
		if _, err := io.WriteString(w, "</s>"); err != nil {
			return fmt.Errorf("write strikethrough close: %w", err)
		}
	case *domain.Link:
		var labelBuf bytes.Buffer
		if err := r.renderInlines(n.Label, &labelBuf); err != nil {
			return fmt.Errorf("render link label: %w", err)
		}
		if _, err := fmt.Fprintf(w, `<a href="%s">%s</a>`, html.EscapeString(n.Target), labelBuf.String()); err != nil {
			return fmt.Errorf("write link: %w", err)
		}
	case *domain.Media:
		src := strings.TrimPrefix(n.Target, "img:")
		if _, err := fmt.Fprintf(w, `<img src="%s" alt="%s">`, html.EscapeString(src), html.EscapeString(n.Alt)); err != nil {
			return fmt.Errorf("write media: %w", err)
		}
	default:
		return fmt.Errorf("unsupported inline node type %q: %w", node.Type(), domain.ErrUnknownNodeType)
	}

	return nil
}
