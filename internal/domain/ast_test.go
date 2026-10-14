package domain_test

import (
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"vexil/internal/domain"
	"vexil/internal/ports"
)

func TestNodeTypes(t *testing.T) {
	tests := []struct {
		name     string
		node     domain.Node
		expected string
	}{
		{"Document", &domain.Document{}, "document"},
		{"Heading", &domain.Heading{}, "heading"},
		{"Paragraph", &domain.Paragraph{}, "paragraph"},
		{"UnorderedList", &domain.UnorderedList{}, "unordered_list"},
		{"OrderedList", &domain.OrderedList{}, "ordered_list"},
		{"ListItem", &domain.ListItem{}, "list_item"},
		{"TaskList", &domain.TaskList{}, "task_list"},
		{"TaskItem", &domain.TaskItem{}, "task_item"},
		{"Callout", &domain.Callout{}, "callout"},
		{"Blockquote", &domain.Blockquote{}, "blockquote"},
		{"CodeFence", &domain.CodeFence{}, "code_fence"},
		{"Table", &domain.Table{}, "table"},
		{"Divider", &domain.Divider{}, "divider"},
		{"Text", &domain.Text{}, "text"},
		{"Strong", &domain.Strong{}, "strong"},
		{"Emphasis", &domain.Emphasis{}, "emphasis"},
		{"InlineCode", &domain.InlineCode{}, "inline_code"},
		{"Strikethrough", &domain.Strikethrough{}, "strikethrough"},
		{"Link", &domain.Link{}, "link"},
		{"Media", &domain.Media{}, "media"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.node.Type(); got != tt.expected {
				t.Fatalf("expected node type %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestInlineNodeInterface(t *testing.T) {
	// Verify all inline node types satisfy domain.InlineNode
	var _ domain.InlineNode = &domain.Text{}
	var _ domain.InlineNode = &domain.Strong{}
	var _ domain.InlineNode = &domain.Emphasis{}
	var _ domain.InlineNode = &domain.InlineCode{}
	var _ domain.InlineNode = &domain.Strikethrough{}
	var _ domain.InlineNode = &domain.Link{}
	var _ domain.InlineNode = &domain.Media{}
}

func TestDocumentJSONSerialization(t *testing.T) {
	doc := &domain.Document{
		Meta: map[string]string{
			"title":  "Sample Document",
			"author": "Alice",
		},
		Children: []domain.Node{
			&domain.Heading{
				Level: 1,
				Meta:  map[string]string{"id": "intro"},
				Children: []domain.InlineNode{
					&domain.Text{Value: "Introduction"},
				},
			},
			&domain.Paragraph{
				Children: []domain.InlineNode{
					&domain.Text{Value: "This is "},
					&domain.Strong{
						Children: []domain.InlineNode{
							&domain.Text{Value: "bold"},
						},
					},
					&domain.Text{Value: " and "},
					&domain.Emphasis{
						Children: []domain.InlineNode{
							&domain.Text{Value: "italic"},
						},
					},
					&domain.Text{Value: " with "},
					&domain.InlineCode{Value: "code()"},
					&domain.Text{Value: " and "},
					&domain.Strikethrough{
						Children: []domain.InlineNode{
							&domain.Text{Value: "strike"},
						},
					},
					&domain.Text{Value: " and a "},
					&domain.Link{
						Label:  []domain.InlineNode{&domain.Text{Value: "docs"}},
						Target: "https://example.com",
					},
					&domain.Text{Value: " and "},
					&domain.Media{
						Alt:    "Logo",
						Target: "img:https://example.com/logo.png",
					},
					&domain.Text{Value: "."},
				},
			},
			&domain.UnorderedList{
				Items: []*domain.ListItem{
					{Level: 0, Children: []domain.InlineNode{&domain.Text{Value: "Bullet 1"}}},
					{Level: 1, Children: []domain.InlineNode{&domain.Text{Value: "Nested bullet"}}},
				},
			},
			&domain.OrderedList{
				Items: []*domain.ListItem{
					{Level: 0, Children: []domain.InlineNode{&domain.Text{Value: "Step 1"}}},
				},
			},
			&domain.TaskList{
				Items: []*domain.TaskItem{
					{Checked: false, Level: 0, Children: []domain.InlineNode{&domain.Text{Value: "Todo"}}},
					{Checked: true, Level: 0, Children: []domain.InlineNode{&domain.Text{Value: "Done"}}},
				},
			},
			&domain.Callout{
				CalloutType: "warning",
				Title:       "Attention",
				Children: []domain.InlineNode{
					&domain.Text{Value: "Be careful!"},
				},
			},
			&domain.Blockquote{
				Children: []domain.InlineNode{
					&domain.Text{Value: "A famous quote"},
				},
			},
			&domain.CodeFence{
				Lang:       "go",
				Identifier: "sample",
				Content:    "fmt.Println(\"hello\")\n",
			},
			&domain.Table{
				Headers: []string{"Col A", "Col B"},
				Rows: [][][]domain.InlineNode{
					{
						{&domain.Text{Value: "A1"}},
						{&domain.Text{Value: "B1"}},
					},
				},
			},
			&domain.Divider{},
		},
	}

	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("failed to marshal document to JSON: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON into map: %v", err)
	}

	if parsed["type"] != "document" {
		t.Fatalf("expected document type 'document', got %v", parsed["type"])
	}

	children, ok := parsed["children"].([]interface{})
	if !ok || len(children) != 10 {
		t.Fatalf("expected 10 children, got %v", len(children))
	}
}

func TestJSONRoundTrip(t *testing.T) {
	doc := &domain.Document{
		Meta: map[string]string{
			"version": "1.0",
		},
		Children: []domain.Node{
			&domain.Heading{
				Level: 2,
				Meta:  map[string]string{"key": "val"},
				Children: []domain.InlineNode{
					&domain.Text{Value: "Heading Text"},
				},
			},
			&domain.Paragraph{
				Children: []domain.InlineNode{
					&domain.Text{Value: "Para text"},
				},
			},
			&domain.Divider{
				Meta: map[string]string{"style": "solid"},
			},
			&domain.CodeFence{
				Lang:    "json",
				Content: `{"a": 1}`,
			},
			&domain.Table{
				Headers: []string{"H1", "H2"},
				Rows: [][][]domain.InlineNode{
					{
						{&domain.Text{Value: "Cell 1"}},
						{&domain.Text{Value: "Cell 2"}},
					},
				},
			},
			&domain.Callout{
				CalloutType: "info",
				Title:       "Tip Title",
				Children: []domain.InlineNode{
					&domain.Text{Value: "Callout body"},
				},
			},
			&domain.Blockquote{
				Children: []domain.InlineNode{
					&domain.Text{Value: "Quote body"},
				},
			},
			&domain.UnorderedList{
				Items: []*domain.ListItem{
					{Level: 0, Children: []domain.InlineNode{&domain.Text{Value: "List item"}}},
				},
			},
			&domain.OrderedList{
				Items: []*domain.ListItem{
					{Level: 1, Children: []domain.InlineNode{&domain.Text{Value: "Ordered item"}}},
				},
			},
			&domain.TaskList{
				Items: []*domain.TaskItem{
					{Checked: true, Level: 0, Children: []domain.InlineNode{&domain.Text{Value: "Done item"}}},
				},
			},
		},
	}

	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded domain.Document
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal roundtrip: %v", err)
	}

	if !reflect.DeepEqual(doc, &decoded) {
		t.Fatalf("roundtrip mismatch:\nwant: %#v\ngot:  %#v", doc, &decoded)
	}
}

func TestIndividualNodeSerialization(t *testing.T) {
	tests := []struct {
		name     string
		node     domain.Node
		expected string
	}{
		{
			name:     "Text",
			node:     &domain.Text{Value: "hello"},
			expected: `{"type":"text","value":"hello"}`,
		},
		{
			name:     "InlineCode",
			node:     &domain.InlineCode{Value: "var x = 1"},
			expected: `{"type":"inline_code","value":"var x = 1"}`,
		},
		{
			name:     "Divider",
			node:     &domain.Divider{},
			expected: `{"type":"divider"}`,
		},
		{
			name:     "TaskItem",
			node:     &domain.TaskItem{Checked: false, Level: 1},
			expected: `{"type":"task_item","checked":false,"level":1}`,
		},
		{
			name:     "Link",
			node:     &domain.Link{Target: "https://example.com"},
			expected: `{"type":"link","target":"https://example.com"}`,
		},
		{
			name:     "Media",
			node:     &domain.Media{Alt: "desc", Target: "img:pic.png"},
			expected: `{"type":"media","alt":"desc","target":"img:pic.png"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.node)
			if err != nil {
				t.Fatalf("failed to marshal %s: %v", tt.name, err)
			}
			if string(b) != tt.expected {
				t.Fatalf("expected JSON %s, got %s", tt.expected, string(b))
			}
		})
	}
}

func TestUnmarshalNodeErrors(t *testing.T) {
	// Unknown node type
	_, err := domain.UnmarshalNode([]byte(`{"type":"unknown_type"}`))
	if err == nil {
		t.Fatal("expected error for unknown node type, got nil")
	}
	if !errors.Is(err, domain.ErrUnknownNodeType) {
		t.Fatalf("expected ErrUnknownNodeType, got %v", err)
	}

	// Invalid JSON
	_, err = domain.UnmarshalNode([]byte(`invalid json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}

	// UnmarshalInlineNode on block node
	_, err = domain.UnmarshalInlineNode([]byte(`{"type":"heading","level":1}`))
	if err == nil {
		t.Fatal("expected error for block node passed to UnmarshalInlineNode, got nil")
	}
	if !errors.Is(err, domain.ErrNotAnInlineNode) {
		t.Fatalf("expected ErrNotAnInlineNode, got %v", err)
	}
}

func TestWalk(t *testing.T) {
	doc := &domain.Document{
		Children: []domain.Node{
			&domain.Heading{
				Level: 1,
				Children: []domain.InlineNode{
					&domain.Text{Value: "Title"},
				},
			},
			&domain.Paragraph{
				Children: []domain.InlineNode{
					&domain.Strong{
						Children: []domain.InlineNode{
							&domain.Text{Value: "Bold text"},
						},
					},
				},
			},
		},
	}

	var visited []string
	domain.Inspect(doc, func(n domain.Node) bool {
		if n != nil {
			visited = append(visited, n.Type())
		}
		return true
	})

	expected := []string{"document", "heading", "text", "paragraph", "strong", "text"}
	if !reflect.DeepEqual(visited, expected) {
		t.Fatalf("expected visit order %v, got %v", expected, visited)
	}

	// Nil node walk
	domain.Walk(nil, nil)
}

func TestToken(t *testing.T) {
	tok := domain.NewToken(domain.TokenHeading, "##", 1, 1)
	if tok.Type != domain.TokenHeading {
		t.Fatalf("expected TokenHeading, got %v", tok.Type)
	}
	if tok.Literal != "##" {
		t.Fatalf("expected literal '##', got %v", tok.Literal)
	}
	if tok.Line != 1 || tok.Column != 1 {
		t.Fatalf("expected line 1 col 1, got %d:%d", tok.Line, tok.Column)
	}

	s := tok.String()
	if !strings.Contains(s, "HEADING") || !strings.Contains(s, "##") {
		t.Fatalf("unexpected token string: %s", s)
	}
}

type testParser struct{}

func (p *testParser) Parse(r io.Reader) (*domain.Document, error) {
	return &domain.Document{}, nil
}

type testStreamParser struct{}

func (s *testStreamParser) ParseStream(r io.Reader) (<-chan domain.Node, <-chan error) {
	nodeCh := make(chan domain.Node)
	errCh := make(chan error)
	close(nodeCh)
	close(errCh)
	return nodeCh, errCh
}

type testRenderer struct{}

func (r *testRenderer) Render(doc *domain.Document, w io.Writer) error {
	return nil
}

type testChunker struct{}

func (c *testChunker) Chunk(doc *domain.Document) ([]ports.Chunk, error) {
	return []ports.Chunk{
		{
			ID:          "chunk-1",
			Title:       "Test Chunk",
			Breadcrumbs: []string{"H1"},
			Content:     "Content",
			CharCount:   7,
			TokenCount:  2,
		},
	}, nil
}

type testFormatter struct{}

func (f *testFormatter) Format(r io.Reader, w io.Writer) error {
	return nil
}

func TestPortContracts(t *testing.T) {
	var _ ports.Parser = &testParser{}
	var _ ports.StreamParser = &testStreamParser{}
	var _ ports.Renderer = &testRenderer{}
	var _ ports.Chunker = &testChunker{}
	var _ ports.Formatter = &testFormatter{}

	parser := &testParser{}
	doc, err := parser.Parse(strings.NewReader("# Title"))
	if err != nil || doc == nil {
		t.Fatalf("parser contract failed: %v", err)
	}

	chunker := &testChunker{}
	chunks, err := chunker.Chunk(doc)
	if err != nil || len(chunks) != 1 {
		t.Fatalf("chunker contract failed: %v", err)
	}

	if chunks[0].ID != "chunk-1" || chunks[0].TokenCount != 2 {
		t.Fatalf("chunk struct mismatch: %+v", chunks[0])
	}
}
