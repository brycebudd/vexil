package render

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"vexil/internal/domain"
	"vexil/internal/ports"
)

// Ensure MarkdownRenderer implements ports.Renderer.
var (
	_ ports.Renderer = (*MarkdownRenderer)(nil)
)

// MarkdownOption configures a MarkdownRenderer instance.
type MarkdownOption func(*MarkdownRenderer)

// MarkdownRenderer transpiles a Vexil AST into GitHub-Flavored Markdown (GFM).
type MarkdownRenderer struct{}

// CommonMarkRenderer is an alias for MarkdownRenderer.
type CommonMarkRenderer = MarkdownRenderer

// NewMarkdownRenderer creates a new MarkdownRenderer with optional configurations.
func NewMarkdownRenderer(opts ...MarkdownOption) *MarkdownRenderer {
	r := &MarkdownRenderer{}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// NewCommonMarkRenderer creates a new CommonMarkRenderer (alias for MarkdownRenderer).
func NewCommonMarkRenderer(opts ...MarkdownOption) *CommonMarkRenderer {
	return NewMarkdownRenderer(opts...)
}

// Render serializes the given Document AST into GitHub-Flavored Markdown written to w.
func (r *MarkdownRenderer) Render(doc *domain.Document, w io.Writer) error {
	if doc == nil {
		return fmt.Errorf("render markdown: %w", ErrNilDocument)
	}
	if w == nil {
		return fmt.Errorf("render markdown: %w", ErrNilWriter)
	}

	if len(doc.Children) == 0 {
		return nil
	}

	var buf bytes.Buffer
	for _, child := range doc.Children {
		if err := r.RenderNode(child, &buf); err != nil {
			return fmt.Errorf("render node: %w", err)
		}
	}

	// Ensure clean document termination with exactly one trailing newline
	trimmed := strings.TrimRight(buf.String(), "\n") + "\n"
	if _, err := io.WriteString(w, trimmed); err != nil {
		return fmt.Errorf("write markdown: %w", err)
	}

	return nil
}

// RenderToString serializes the given Document AST into a Markdown string.
func (r *MarkdownRenderer) RenderToString(doc *domain.Document) (string, error) {
	var buf bytes.Buffer
	if err := r.Render(doc, &buf); err != nil {
		return "", fmt.Errorf("render to string: %w", err)
	}
	return buf.String(), nil
}

// RenderNode renders an individual AST node into GitHub-Flavored Markdown.
func (r *MarkdownRenderer) RenderNode(node domain.Node, w io.Writer) error {
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

func (r *MarkdownRenderer) renderHeading(h *domain.Heading, w io.Writer) error {
	level := h.Level
	if level < 1 {
		level = 1
	} else if level > 4 {
		level = 4
	}

	prefix := strings.Repeat("#", level)
	var inlinesBuf bytes.Buffer
	if err := r.renderInlines(h.Children, &inlinesBuf); err != nil {
		return fmt.Errorf("render heading inlines: %w", err)
	}

	if _, err := fmt.Fprintf(w, "%s %s\n\n", prefix, inlinesBuf.String()); err != nil {
		return fmt.Errorf("write heading: %w", err)
	}
	return nil
}

func (r *MarkdownRenderer) renderParagraph(p *domain.Paragraph, w io.Writer) error {
	var inlinesBuf bytes.Buffer
	if err := r.renderInlines(p.Children, &inlinesBuf); err != nil {
		return fmt.Errorf("render paragraph inlines: %w", err)
	}

	if _, err := fmt.Fprintf(w, "%s\n\n", inlinesBuf.String()); err != nil {
		return fmt.Errorf("write paragraph: %w", err)
	}
	return nil
}

func (r *MarkdownRenderer) renderUnorderedList(u *domain.UnorderedList, w io.Writer) error {
	for _, item := range u.Items {
		indent := strings.Repeat("  ", item.Level)
		var inlinesBuf bytes.Buffer
		if err := r.renderInlines(item.Children, &inlinesBuf); err != nil {
			return fmt.Errorf("render unordered list item inlines: %w", err)
		}
		if _, err := fmt.Fprintf(w, "%s- %s\n", indent, inlinesBuf.String()); err != nil {
			return fmt.Errorf("write unordered list item: %w", err)
		}
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return fmt.Errorf("write unordered list trailing newline: %w", err)
	}
	return nil
}

func (r *MarkdownRenderer) renderOrderedList(o *domain.OrderedList, w io.Writer) error {
	for i, item := range o.Items {
		indent := strings.Repeat("  ", item.Level)
		var inlinesBuf bytes.Buffer
		if err := r.renderInlines(item.Children, &inlinesBuf); err != nil {
			return fmt.Errorf("render ordered list item inlines: %w", err)
		}
		if _, err := fmt.Fprintf(w, "%s%d. %s\n", indent, i+1, inlinesBuf.String()); err != nil {
			return fmt.Errorf("write ordered list item: %w", err)
		}
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return fmt.Errorf("write ordered list trailing newline: %w", err)
	}
	return nil
}

func (r *MarkdownRenderer) renderTaskList(t *domain.TaskList, w io.Writer) error {
	for _, item := range t.Items {
		indent := strings.Repeat("  ", item.Level)
		check := "[ ]"
		if item.Checked {
			check = "[x]"
		}
		var inlinesBuf bytes.Buffer
		if err := r.renderInlines(item.Children, &inlinesBuf); err != nil {
			return fmt.Errorf("render task list item inlines: %w", err)
		}
		if _, err := fmt.Fprintf(w, "%s- %s %s\n", indent, check, inlinesBuf.String()); err != nil {
			return fmt.Errorf("write task list item: %w", err)
		}
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return fmt.Errorf("write task list trailing newline: %w", err)
	}
	return nil
}

func (r *MarkdownRenderer) renderCallout(c *domain.Callout, w io.Writer) error {
	alertType := strings.ToUpper(c.CalloutType)
	var sb strings.Builder
	fmt.Fprintf(&sb, "> [!%s]\n", alertType)

	if c.Title != "" {
		fmt.Fprintf(&sb, "> **%s**\n", c.Title)
	}

	var bodyBuf bytes.Buffer
	if err := r.renderInlines(c.Children, &bodyBuf); err != nil {
		return fmt.Errorf("render callout inlines: %w", err)
	}

	bodyStr := bodyBuf.String()
	if len(bodyStr) > 0 {
		lines := strings.Split(bodyStr, "\n")
		for _, line := range lines {
			fmt.Fprintf(&sb, "> %s\n", line)
		}
	}
	sb.WriteString("\n")

	if _, err := io.WriteString(w, sb.String()); err != nil {
		return fmt.Errorf("write callout: %w", err)
	}
	return nil
}

func (r *MarkdownRenderer) renderBlockquote(b *domain.Blockquote, w io.Writer) error {
	var inlinesBuf bytes.Buffer
	if err := r.renderInlines(b.Children, &inlinesBuf); err != nil {
		return fmt.Errorf("render blockquote inlines: %w", err)
	}

	lines := strings.Split(inlinesBuf.String(), "\n")
	for _, line := range lines {
		if _, err := fmt.Fprintf(w, "> %s\n", line); err != nil {
			return fmt.Errorf("write blockquote line: %w", err)
		}
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return fmt.Errorf("write blockquote trailing newline: %w", err)
	}
	return nil
}

func (r *MarkdownRenderer) renderCodeFence(cf *domain.CodeFence, w io.Writer) error {
	content := cf.Content
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	if _, err := fmt.Fprintf(w, "```%s\n%s```\n\n", cf.Lang, content); err != nil {
		return fmt.Errorf("write code fence: %w", err)
	}
	return nil
}

func (r *MarkdownRenderer) renderTable(t *domain.Table, w io.Writer) error {
	numCols := len(t.Headers)
	if numCols == 0 && len(t.Rows) > 0 {
		numCols = len(t.Rows[0])
	}
	if numCols == 0 {
		return nil
	}

	var sb strings.Builder

	// Header row
	if len(t.Headers) > 0 {
		sb.WriteString("| ")
		sb.WriteString(strings.Join(t.Headers, " | "))
		sb.WriteString(" |\n")
	} else {
		sb.WriteString("|")
		for i := 0; i < numCols; i++ {
			sb.WriteString("   |")
		}
		sb.WriteString("\n")
	}

	// Divider row: |---|---|
	sb.WriteString("|")
	for i := 0; i < numCols; i++ {
		sb.WriteString("---|")
	}
	sb.WriteString("\n")

	// Table rows
	for _, row := range t.Rows {
		sb.WriteString("| ")
		var cellStrs []string
		for _, cell := range row {
			var cellBuf bytes.Buffer
			if err := r.renderInlines(cell, &cellBuf); err != nil {
				return fmt.Errorf("render table cell inlines: %w", err)
			}
			cellText := strings.ReplaceAll(cellBuf.String(), "|", `\|`)
			cellStrs = append(cellStrs, cellText)
		}
		sb.WriteString(strings.Join(cellStrs, " | "))
		sb.WriteString(" |\n")
	}
	sb.WriteString("\n")

	if _, err := io.WriteString(w, sb.String()); err != nil {
		return fmt.Errorf("write table: %w", err)
	}
	return nil
}

func (r *MarkdownRenderer) renderDivider(_ *domain.Divider, w io.Writer) error {
	if _, err := io.WriteString(w, "---\n\n"); err != nil {
		return fmt.Errorf("write divider: %w", err)
	}
	return nil
}

func (r *MarkdownRenderer) renderInlines(inlines []domain.InlineNode, w io.Writer) error {
	for _, inline := range inlines {
		if err := r.renderInline(inline, w); err != nil {
			return fmt.Errorf("render inline node: %w", err)
		}
	}
	return nil
}

func (r *MarkdownRenderer) renderInline(node domain.InlineNode, w io.Writer) error {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *domain.Text:
		if _, err := io.WriteString(w, n.Value); err != nil {
			return fmt.Errorf("write text: %w", err)
		}
	case *domain.Strong:
		if _, err := io.WriteString(w, "**"); err != nil {
			return fmt.Errorf("write strong open: %w", err)
		}
		if err := r.renderInlines(n.Children, w); err != nil {
			return err
		}
		if _, err := io.WriteString(w, "**"); err != nil {
			return fmt.Errorf("write strong close: %w", err)
		}
	case *domain.Emphasis:
		if _, err := io.WriteString(w, "*"); err != nil {
			return fmt.Errorf("write emphasis open: %w", err)
		}
		if err := r.renderInlines(n.Children, w); err != nil {
			return err
		}
		if _, err := io.WriteString(w, "*"); err != nil {
			return fmt.Errorf("write emphasis close: %w", err)
		}
	case *domain.InlineCode:
		if _, err := fmt.Fprintf(w, "`%s`", n.Value); err != nil {
			return fmt.Errorf("write inline code: %w", err)
		}
	case *domain.Strikethrough:
		if _, err := io.WriteString(w, "~~"); err != nil {
			return fmt.Errorf("write strikethrough open: %w", err)
		}
		if err := r.renderInlines(n.Children, w); err != nil {
			return err
		}
		if _, err := io.WriteString(w, "~~"); err != nil {
			return fmt.Errorf("write strikethrough close: %w", err)
		}
	case *domain.Link:
		var labelBuf bytes.Buffer
		if err := r.renderInlines(n.Label, &labelBuf); err != nil {
			return fmt.Errorf("render link label: %w", err)
		}
		if _, err := fmt.Fprintf(w, "[%s](%s)", labelBuf.String(), n.Target); err != nil {
			return fmt.Errorf("write link: %w", err)
		}
	case *domain.Media:
		src := strings.TrimPrefix(n.Target, "img:")
		if _, err := fmt.Fprintf(w, "![%s](%s)", n.Alt, src); err != nil {
			return fmt.Errorf("write media: %w", err)
		}
	default:
		return fmt.Errorf("unsupported inline node type %q: %w", node.Type(), domain.ErrUnknownNodeType)
	}

	return nil
}
