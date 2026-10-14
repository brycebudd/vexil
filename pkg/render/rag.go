package render

import (
	"fmt"
	"strings"

	"vexil/internal/domain"
	"vexil/internal/ports"
)

// Ensure RAGChunker satisfies the ports.Chunker interface.
var _ ports.Chunker = (*RAGChunker)(nil)

type headingEntry struct {
	level int
	title string
}

// RAGChunker segments a Vexil Document AST into semantic chunks for RAG pipelines.
type RAGChunker struct {
	tokenRatio float64
}

// Option configures a RAGChunker instance.
type Option func(*RAGChunker)

// WithTokenRatio configures custom characters-per-token ratio for token estimation.
func WithTokenRatio(ratio float64) Option {
	return func(c *RAGChunker) {
		if ratio > 0 {
			c.tokenRatio = ratio
		}
	}
}

// NewRAGChunker creates a new RAGChunker initialized with default configuration.
func NewRAGChunker(opts ...Option) *RAGChunker {
	c := &RAGChunker{
		tokenRatio: 4.0,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Chunk partitions a Vexil document into semantic chunks bounded by Headings and Tables.
func (c *RAGChunker) Chunk(doc *domain.Document) ([]ports.Chunk, error) {
	if doc == nil {
		return nil, fmt.Errorf("chunk document: document is nil")
	}

	chunks := make([]ports.Chunk, 0)
	var headingStack []headingEntry

	var currentBlocks []domain.Node
	var currentBlockMeta map[string]string
	chunkIndex := 1

	emitChunk := func(title string, breadcrumbs []string, blocks []domain.Node, blockMeta map[string]string) {
		if len(blocks) == 0 {
			return
		}
		content := renderBlocksText(blocks)
		if content == "" {
			return
		}

		meta := copyMeta(doc.Meta)
		if meta == nil && len(blockMeta) > 0 {
			meta = make(map[string]string, len(blockMeta))
		}
		mergeMeta(meta, blockMeta)

		id := fmt.Sprintf("chunk-%d", chunkIndex)
		if blockMeta != nil {
			if customID, ok := blockMeta["id"]; ok && strings.TrimSpace(customID) != "" {
				id = strings.TrimSpace(customID)
			}
		}
		chunkIndex++

		charCount := len(content)
		tokenCount := 0
		if charCount > 0 {
			tokenCount = int((float64(charCount) + c.tokenRatio - 1) / c.tokenRatio)
		}

		bcs := breadcrumbs
		if bcs == nil {
			bcs = []string{}
		}

		chunks = append(chunks, ports.Chunk{
			ID:          id,
			Title:       title,
			Breadcrumbs: bcs,
			Content:     content,
			Meta:        meta,
			CharCount:   charCount,
			TokenCount:  tokenCount,
		})
	}

	for _, child := range doc.Children {
		switch node := child.(type) {
		case *domain.Heading:
			if len(currentBlocks) > 0 {
				if len(headingStack) == 0 {
					emitChunk("Preamble", []string{}, currentBlocks, nil)
				} else {
					emitChunk(currentHeadingTitle(headingStack), copyBreadcrumbs(headingStack), currentBlocks, currentBlockMeta)
				}
			}

			hTitle := flattenInlines(node.Children)
			if hTitle == "" {
				hTitle = "Untitled"
			}

			for len(headingStack) > 0 && headingStack[len(headingStack)-1].level >= node.Level {
				headingStack = headingStack[:len(headingStack)-1]
			}
			headingStack = append(headingStack, headingEntry{level: node.Level, title: hTitle})

			currentBlockMeta = node.Meta
			currentBlocks = []domain.Node{node}

		case *domain.Table:
			if len(currentBlocks) > 0 {
				if len(headingStack) == 0 {
					emitChunk("Preamble", []string{}, currentBlocks, nil)
				} else {
					emitChunk(currentHeadingTitle(headingStack), copyBreadcrumbs(headingStack), currentBlocks, currentBlockMeta)
				}
				currentBlocks = nil
				currentBlockMeta = nil
			}

			tableTitle := getTableTitle(node)
			emitChunk(tableTitle, copyBreadcrumbs(headingStack), []domain.Node{node}, node.Meta)

		default:
			currentBlocks = append(currentBlocks, node)
		}
	}

	if len(currentBlocks) > 0 {
		if len(headingStack) == 0 {
			emitChunk("Preamble", []string{}, currentBlocks, nil)
		} else {
			emitChunk(currentHeadingTitle(headingStack), copyBreadcrumbs(headingStack), currentBlocks, currentBlockMeta)
		}
	}

	return chunks, nil
}

// ChunkRAG provides package-level convenience function for semantic chunking.
func ChunkRAG(doc *domain.Document) ([]ports.Chunk, error) {
	return NewRAGChunker().Chunk(doc)
}

func currentHeadingTitle(stack []headingEntry) string {
	if len(stack) == 0 {
		return "Preamble"
	}
	return stack[len(stack)-1].title
}

func copyBreadcrumbs(stack []headingEntry) []string {
	if len(stack) == 0 {
		return []string{}
	}
	res := make([]string, len(stack))
	for i, entry := range stack {
		res[i] = entry.title
	}
	return res
}

func getTableTitle(t *domain.Table) string {
	if t.Meta != nil {
		if title, ok := t.Meta["title"]; ok && strings.TrimSpace(title) != "" {
			return strings.TrimSpace(title)
		}
		if caption, ok := t.Meta["caption"]; ok && strings.TrimSpace(caption) != "" {
			return strings.TrimSpace(caption)
		}
		if name, ok := t.Meta["name"]; ok && strings.TrimSpace(name) != "" {
			return strings.TrimSpace(name)
		}
	}
	return "Table"
}

func copyMeta(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func mergeMeta(dst, src map[string]string) {
	if src == nil {
		return
	}
	for k, v := range src {
		dst[k] = v
	}
}

func renderBlocksText(nodes []domain.Node) string {
	var parts []string
	for _, node := range nodes {
		text := strings.TrimSpace(renderBlockText(node))
		if text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n\n")
}

func renderBlockText(node domain.Node) string {
	if node == nil {
		return ""
	}

	switch n := node.(type) {
	case *domain.Heading:
		return flattenInlines(n.Children)

	case *domain.Paragraph:
		return flattenInlines(n.Children)

	case *domain.UnorderedList:
		var lines []string
		for _, item := range n.Items {
			if item == nil {
				continue
			}
			indent := strings.Repeat("  ", item.Level)
			text := flattenInlines(item.Children)
			lines = append(lines, fmt.Sprintf("%s- %s", indent, text))
		}
		return strings.Join(lines, "\n")

	case *domain.OrderedList:
		var lines []string
		for i, item := range n.Items {
			if item == nil {
				continue
			}
			indent := strings.Repeat("  ", item.Level)
			text := flattenInlines(item.Children)
			lines = append(lines, fmt.Sprintf("%s%d. %s", indent, i+1, text))
		}
		return strings.Join(lines, "\n")

	case *domain.TaskList:
		var lines []string
		for _, item := range n.Items {
			if item == nil {
				continue
			}
			indent := strings.Repeat("  ", item.Level)
			text := flattenInlines(item.Children)
			mark := "[ ]"
			if item.Checked {
				mark = "[x]"
			}
			lines = append(lines, fmt.Sprintf("%s%s %s", indent, mark, text))
		}
		return strings.Join(lines, "\n")

	case *domain.Callout:
		var parts []string
		if n.Title != "" {
			parts = append(parts, n.Title)
		}
		body := flattenInlines(n.Children)
		if body != "" {
			parts = append(parts, body)
		}
		if len(parts) == 0 && n.CalloutType != "" {
			return n.CalloutType
		}
		return strings.Join(parts, "\n")

	case *domain.Blockquote:
		return flattenInlines(n.Children)

	case *domain.CodeFence:
		return strings.TrimRight(n.Content, "\n")

	case *domain.Table:
		var lines []string
		if len(n.Headers) > 0 {
			lines = append(lines, strings.Join(n.Headers, " | "))
		}
		for _, row := range n.Rows {
			var cells []string
			for _, cell := range row {
				cells = append(cells, flattenInlines(cell))
			}
			lines = append(lines, strings.Join(cells, " | "))
		}
		return strings.Join(lines, "\n")

	case *domain.Divider:
		return ""

	default:
		return ""
	}
}

func flattenInlines(nodes []domain.InlineNode) string {
	var sb strings.Builder
	for _, node := range nodes {
		sb.WriteString(flattenInline(node))
	}
	return sb.String()
}

func flattenInline(node domain.InlineNode) string {
	if node == nil {
		return ""
	}

	switch n := node.(type) {
	case *domain.Text:
		return n.Value
	case *domain.InlineCode:
		return n.Value
	case *domain.Strong:
		return flattenInlines(n.Children)
	case *domain.Emphasis:
		return flattenInlines(n.Children)
	case *domain.Strikethrough:
		return flattenInlines(n.Children)
	case *domain.Link:
		if len(n.Label) > 0 {
			return flattenInlines(n.Label)
		}
		return n.Target
	case *domain.Media:
		if n.Alt != "" {
			return n.Alt
		}
		return n.Target
	default:
		return ""
	}
}
