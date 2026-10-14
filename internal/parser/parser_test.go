package parser_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"vexil/internal/domain"
	"vexil/internal/parser"
	"vexil/internal/ports"
)

func TestParserContract(t *testing.T) {
	var _ ports.Parser = parser.NewParser()
	var _ ports.StreamParser = parser.NewParser()
	var _ ports.StreamParser = parser.NewStreamParser()

	p := parser.NewParser()
	doc, err := p.ParseString("# Hello")
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}
	if len(doc.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(doc.Children))
	}

	doc2, err := p.ParseBytes([]byte("# Hello"))
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}
	if len(doc2.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(doc2.Children))
	}

	doc3, err := parser.Parse(strings.NewReader("# Hello"))
	if err != nil {
		t.Fatalf("parser.Parse failed: %v", err)
	}
	if len(doc3.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(doc3.Children))
	}

	doc4, err := parser.ParseString("# Hello")
	if err != nil {
		t.Fatalf("parser.ParseString failed: %v", err)
	}
	if len(doc4.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(doc4.Children))
	}

	doc5, err := parser.ParseBytes([]byte("# Hello"))
	if err != nil {
		t.Fatalf("parser.ParseBytes failed: %v", err)
	}
	if len(doc5.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(doc5.Children))
	}
}

func TestParserHeadingWithInline(t *testing.T) {
	input := `# Title with *bold* and _italic_ and ` + "`code`" + ` and [link](https://example.com) and ~strike~`
	doc, err := parser.ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	if len(doc.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(doc.Children))
	}

	h, ok := doc.Children[0].(*domain.Heading)
	if !ok {
		t.Fatalf("expected *domain.Heading, got %T", doc.Children[0])
	}
	if h.Level != 1 {
		t.Errorf("expected level 1, got %d", h.Level)
	}

	expectedInlines := []domain.InlineNode{
		&domain.Text{Value: "Title with "},
		&domain.Strong{Children: []domain.InlineNode{&domain.Text{Value: "bold"}}},
		&domain.Text{Value: " and "},
		&domain.Emphasis{Children: []domain.InlineNode{&domain.Text{Value: "italic"}}},
		&domain.Text{Value: " and "},
		&domain.InlineCode{Value: "code"},
		&domain.Text{Value: " and "},
		&domain.Link{Label: []domain.InlineNode{&domain.Text{Value: "link"}}, Target: "https://example.com"},
		&domain.Text{Value: " and "},
		&domain.Strikethrough{Children: []domain.InlineNode{&domain.Text{Value: "strike"}}},
	}

	if !reflect.DeepEqual(h.Children, expectedInlines) {
		t.Errorf("heading inlines mismatch:\ngot:  %#v\nwant: %#v", h.Children, expectedInlines)
	}
}

func TestParserParagraphWithInline(t *testing.T) {
	input := `This is a paragraph with *bold text* and an image [diagram](system.png).
Second line continues the paragraph with ` + "`const X = 42`" + ` inline.`

	doc, err := parser.ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	if len(doc.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(doc.Children))
	}

	p, ok := doc.Children[0].(*domain.Paragraph)
	if !ok {
		t.Fatalf("expected *domain.Paragraph, got %T", doc.Children[0])
	}

	expectedInlines := []domain.InlineNode{
		&domain.Text{Value: "This is a paragraph with "},
		&domain.Strong{Children: []domain.InlineNode{&domain.Text{Value: "bold text"}}},
		&domain.Text{Value: " and an image "},
		&domain.Media{Alt: "diagram", Target: "system.png"},
		&domain.Text{Value: ".\nSecond line continues the paragraph with "},
		&domain.InlineCode{Value: "const X = 42"},
		&domain.Text{Value: " inline."},
	}

	if !reflect.DeepEqual(p.Children, expectedInlines) {
		t.Errorf("paragraph inlines mismatch:\ngot:  %#v\nwant: %#v", p.Children, expectedInlines)
	}
}

func TestParserListsWithInline(t *testing.T) {
	input := `- Item with *bold*
  - Subitem with _italic_
- Item with ` + "`code`" + `

1. First with [link](https://vexil.org)
  2. Second with ~strike~

[ ] Task with *important* deadline
  [x] Subtask ` + "`done`" + `
`
	doc, err := parser.ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	if len(doc.Children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(doc.Children))
	}

	// Unordered list
	ul, ok := doc.Children[0].(*domain.UnorderedList)
	if !ok {
		t.Fatalf("child 0: expected *domain.UnorderedList, got %T", doc.Children[0])
	}
	if len(ul.Items) != 3 {
		t.Fatalf("expected 3 items in unordered list, got %d", len(ul.Items))
	}
	if ul.Items[0].Level != 0 || len(ul.Items[0].Children) != 2 {
		t.Errorf("unexpected ul item 0: %#v", ul.Items[0])
	}
	if ul.Items[1].Level != 1 {
		t.Errorf("expected level 1 for subitem, got %d", ul.Items[1].Level)
	}

	// Ordered list
	ol, ok := doc.Children[1].(*domain.OrderedList)
	if !ok {
		t.Fatalf("child 1: expected *domain.OrderedList, got %T", doc.Children[1])
	}
	if len(ol.Items) != 2 {
		t.Fatalf("expected 2 items in ordered list, got %d", len(ol.Items))
	}
	if ol.Items[0].Level != 0 || ol.Items[1].Level != 1 {
		t.Errorf("unexpected ol item levels: %d, %d", ol.Items[0].Level, ol.Items[1].Level)
	}

	// Task list
	tl, ok := doc.Children[2].(*domain.TaskList)
	if !ok {
		t.Fatalf("child 2: expected *domain.TaskList, got %T", doc.Children[2])
	}
	if len(tl.Items) != 2 {
		t.Fatalf("expected 2 items in task list, got %d", len(tl.Items))
	}
	if tl.Items[0].Checked != false || tl.Items[0].Level != 0 {
		t.Errorf("unexpected task item 0: %#v", tl.Items[0])
	}
	if tl.Items[1].Checked != true || tl.Items[1].Level != 1 {
		t.Errorf("unexpected task item 1: %#v", tl.Items[1])
	}
}

func TestParserCalloutWithInline(t *testing.T) {
	input := `!warning: Caution Required
This is the *first* body line.
This is the second line with ` + "`danger`" + ` token.`

	doc, err := parser.ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	if len(doc.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(doc.Children))
	}

	c, ok := doc.Children[0].(*domain.Callout)
	if !ok {
		t.Fatalf("expected *domain.Callout, got %T", doc.Children[0])
	}
	if c.CalloutType != "warning" {
		t.Errorf("expected callout type 'warning', got %q", c.CalloutType)
	}
	if c.Title != "Caution Required" {
		t.Errorf("expected title 'Caution Required', got %q", c.Title)
	}

	expectedInlines := []domain.InlineNode{
		&domain.Text{Value: "This is the "},
		&domain.Strong{Children: []domain.InlineNode{&domain.Text{Value: "first"}}},
		&domain.Text{Value: " body line.\nThis is the second line with "},
		&domain.InlineCode{Value: "danger"},
		&domain.Text{Value: " token."},
	}

	if !reflect.DeepEqual(c.Children, expectedInlines) {
		t.Errorf("callout inlines mismatch:\ngot:  %#v\nwant: %#v", c.Children, expectedInlines)
	}
}

func TestParserBlockquoteWithInline(t *testing.T) {
	input := `> Line 1 with *quote*
> Line 2 with _emphasis_ and ` + "`code`"

	doc, err := parser.ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	if len(doc.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(doc.Children))
	}

	bq, ok := doc.Children[0].(*domain.Blockquote)
	if !ok {
		t.Fatalf("expected *domain.Blockquote, got %T", doc.Children[0])
	}

	expectedInlines := []domain.InlineNode{
		&domain.Text{Value: "Line 1 with "},
		&domain.Strong{Children: []domain.InlineNode{&domain.Text{Value: "quote"}}},
		&domain.Text{Value: "\nLine 2 with "},
		&domain.Emphasis{Children: []domain.InlineNode{&domain.Text{Value: "emphasis"}}},
		&domain.Text{Value: " and "},
		&domain.InlineCode{Value: "code"},
	}

	if !reflect.DeepEqual(bq.Children, expectedInlines) {
		t.Errorf("blockquote inlines mismatch:\ngot:  %#v\nwant: %#v", bq.Children, expectedInlines)
	}
}

func TestParserTableWithInlinesAndEscapes(t *testing.T) {
	input := `::table [Column 1, Column 2, Column 3]
- cell with \| escaped pipe | ` + "`code | with | pipes`" + ` | *bold cell*
- [link](https://example.com) | normal cell | plain
::`

	doc, err := parser.ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	if len(doc.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(doc.Children))
	}

	tbl, ok := doc.Children[0].(*domain.Table)
	if !ok {
		t.Fatalf("expected *domain.Table, got %T", doc.Children[0])
	}

	if len(tbl.Headers) != 3 || tbl.Headers[0] != "Column 1" || tbl.Headers[1] != "Column 2" || tbl.Headers[2] != "Column 3" {
		t.Errorf("unexpected headers: %v", tbl.Headers)
	}

	if len(tbl.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(tbl.Rows))
	}
	if len(tbl.Rows[0]) != 3 {
		t.Fatalf("expected 3 columns in row 0, got %d", len(tbl.Rows[0]))
	}

	// Row 0, Cell 0: cell with | escaped pipe (escaped pipe unescaped to plain text '|')
	cell00 := tbl.Rows[0][0]
	expected00 := []domain.InlineNode{&domain.Text{Value: "cell with | escaped pipe"}}
	if !reflect.DeepEqual(cell00, expected00) {
		t.Errorf("cell (0,0) mismatch: got %#v, want %#v", cell00, expected00)
	}

	// Row 0, Cell 1: `code | with | pipes` preserved inside inline code
	cell01 := tbl.Rows[0][1]
	expected01 := []domain.InlineNode{&domain.InlineCode{Value: "code | with | pipes"}}
	if !reflect.DeepEqual(cell01, expected01) {
		t.Errorf("cell (0,1) mismatch: got %#v, want %#v", cell01, expected01)
	}

	// Row 0, Cell 2: *bold cell*
	cell02 := tbl.Rows[0][2]
	expected02 := []domain.InlineNode{&domain.Strong{Children: []domain.InlineNode{&domain.Text{Value: "bold cell"}}}}
	if !reflect.DeepEqual(cell02, expected02) {
		t.Errorf("cell (0,2) mismatch: got %#v, want %#v", cell02, expected02)
	}
}

func TestParserCodeFenceAndDivider(t *testing.T) {
	input := `@lang: go
` + "```go worker.go\npackage main\n\nfunc Run() {}\n```" + `

---
`

	doc, err := parser.ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	if len(doc.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(doc.Children))
	}

	cf, ok := doc.Children[0].(*domain.CodeFence)
	if !ok {
		t.Fatalf("child 0: expected *domain.CodeFence, got %T", doc.Children[0])
	}
	if cf.Lang != "go" || cf.Identifier != "worker.go" {
		t.Errorf("unexpected code fence lang/id: %q / %q", cf.Lang, cf.Identifier)
	}
	if cf.Meta["lang"] != "go" {
		t.Errorf("unexpected code fence meta: %v", cf.Meta)
	}
	if cf.Content != "package main\n\nfunc Run() {}" {
		t.Errorf("unexpected code fence content: %q", cf.Content)
	}

	d, ok := doc.Children[1].(*domain.Divider)
	if !ok {
		t.Fatalf("child 1: expected *domain.Divider, got %T", doc.Children[1])
	}
	if d.Type() != domain.TypeDivider {
		t.Errorf("unexpected divider type: %s", d.Type())
	}
}

func TestParserDocumentAndNodeMetadata(t *testing.T) {
	input := `@title: Spec Doc
@version: 2.0.0

@section: 1
# Section 1

@para: p1
Paragraph text.
`
	doc, err := parser.ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	if doc.Meta["title"] != "Spec Doc" || doc.Meta["version"] != "2.0.0" {
		t.Errorf("unexpected doc.Meta: %v", doc.Meta)
	}

	if len(doc.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(doc.Children))
	}

	h := doc.Children[0].(*domain.Heading)
	if h.Meta["section"] != "1" {
		t.Errorf("unexpected heading meta: %v", h.Meta)
	}

	p := doc.Children[1].(*domain.Paragraph)
	if p.Meta["para"] != "p1" {
		t.Errorf("unexpected paragraph meta: %v", p.Meta)
	}
}

func TestParserEmptyAndWhitespace(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"only whitespace", "   \n\n\t\t\n   "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := parser.ParseString(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(doc.Children) != 0 {
				t.Errorf("expected 0 children, got %d", len(doc.Children))
			}
			if len(doc.Meta) != 0 {
				t.Errorf("expected empty meta, got %v", doc.Meta)
			}
		})
	}
}

func TestParserJSONEquivalence(t *testing.T) {
	input := `@title: Doc Title

# Heading
A paragraph with *bold* text.
`
	doc, err := parser.ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	jsonBytes, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var roundtrip domain.Document
	if err := json.Unmarshal(jsonBytes, &roundtrip); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if !reflect.DeepEqual(doc, &roundtrip) {
		t.Errorf("roundtrip document mismatch:\ngot:  %#v\nwant: %#v", &roundtrip, doc)
	}
}
