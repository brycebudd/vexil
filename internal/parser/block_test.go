package parser

import (
	"strings"
	"testing"

	"vexil/internal/domain"
)

func TestBlockMetadata(t *testing.T) {
	// Top of document metadata attaches to Document.Meta
	input := `@title: Vexil Language Spec
@version: 1.0.0
@author: Vexil Authors

# Section 1
This is paragraph 1.

@id: p-2
@class: highlight
This is paragraph 2.
`
	scanner := NewBlockScanner(strings.NewReader(input))
	doc, err := scanner.ScanDocument()
	if err != nil {
		t.Fatalf("unexpected error scanning document: %v", err)
	}

	if doc.Meta == nil {
		t.Fatal("expected Document.Meta to be populated, got nil")
	}
	if doc.Meta["title"] != "Vexil Language Spec" {
		t.Errorf("expected title 'Vexil Language Spec', got %q", doc.Meta["title"])
	}
	if doc.Meta["version"] != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", doc.Meta["version"])
	}
	if doc.Meta["author"] != "Vexil Authors" {
		t.Errorf("expected author 'Vexil Authors', got %q", doc.Meta["author"])
	}

	if len(doc.Children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(doc.Children))
	}

	// Heading should not have doc-level meta
	h, ok := doc.Children[0].(*domain.Heading)
	if !ok {
		t.Fatalf("expected child 0 to be *domain.Heading, got %T", doc.Children[0])
	}
	if len(h.Meta) != 0 {
		t.Errorf("expected heading meta to be empty, got %v", h.Meta)
	}

	// Paragraph 1 has no meta
	p1, ok := doc.Children[1].(*domain.Paragraph)
	if !ok {
		t.Fatalf("expected child 1 to be *domain.Paragraph, got %T", doc.Children[1])
	}
	if len(p1.Meta) != 0 {
		t.Errorf("expected p1 meta to be empty, got %v", p1.Meta)
	}

	// Paragraph 2 has block meta
	p2, ok := doc.Children[2].(*domain.Paragraph)
	if !ok {
		t.Fatalf("expected child 2 to be *domain.Paragraph, got %T", doc.Children[2])
	}
	if p2.Meta == nil {
		t.Fatal("expected p2.Meta to be populated, got nil")
	}
	if p2.Meta["id"] != "p-2" {
		t.Errorf("expected id 'p-2', got %q", p2.Meta["id"])
	}
	if p2.Meta["class"] != "highlight" {
		t.Errorf("expected class 'highlight', got %q", p2.Meta["class"])
	}

	// Metadata-only document
	metaOnly := `@key1: value1
@key2: value2
`
	scanner2 := NewBlockScanner(strings.NewReader(metaOnly))
	doc2, err := scanner2.ScanDocument()
	if err != nil {
		t.Fatalf("unexpected error scanning meta-only: %v", err)
	}
	if doc2.Meta["key1"] != "value1" || doc2.Meta["key2"] != "value2" {
		t.Errorf("unexpected meta-only map: %v", doc2.Meta)
	}
	if len(doc2.Children) != 0 {
		t.Errorf("expected 0 children for meta-only, got %d", len(doc2.Children))
	}
}

func TestBlockHeadings(t *testing.T) {
	input := `# Level 1 Heading
## Level 2 Heading
### Level 3 Heading
#### Level 4 Heading
##### Degraded to Paragraph
#hashtag
`
	scanner := NewBlockScanner(strings.NewReader(input))
	doc, err := scanner.ScanDocument()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Children) != 5 {
		t.Fatalf("expected 5 children, got %d", len(doc.Children))
	}

	for i := 1; i <= 4; i++ {
		h, ok := doc.Children[i-1].(*domain.Heading)
		if !ok {
			t.Fatalf("child %d: expected *domain.Heading, got %T", i-1, doc.Children[i-1])
		}
		if h.Level != i {
			t.Errorf("child %d: expected level %d, got %d", i-1, i, h.Level)
		}
		if len(h.Children) == 0 {
			t.Fatalf("child %d: expected inline children, got none", i-1)
		}
		txt, ok := h.Children[0].(*domain.Text)
		if !ok {
			t.Fatalf("child %d: expected *domain.Text inline, got %T", i-1, h.Children[0])
		}
		expectedText := strings.TrimSpace(strings.Repeat("#", i) + " Level " + string(rune('0'+i)) + " Heading")[i+1:]
		if txt.Value != expectedText {
			t.Errorf("child %d: expected text %q, got %q", i-1, expectedText, txt.Value)
		}
	}

	// Child 4: 5 hashes + hashtag degraded to paragraphs
	p, ok := doc.Children[4].(*domain.Paragraph)
	if !ok {
		t.Fatalf("child 4: expected *domain.Paragraph, got %T", doc.Children[4])
	}
	txt, ok := p.Children[0].(*domain.Text)
	if !ok {
		t.Fatalf("child 4 inline: expected *domain.Text, got %T", p.Children[0])
	}
	expectedParagraph := "##### Degraded to Paragraph\n#hashtag"
	if txt.Value != expectedParagraph {
		t.Errorf("child 4: expected %q, got %q", expectedParagraph, txt.Value)
	}
}

func TestBlockUnorderedList(t *testing.T) {
	input := `- Item 1
  - Sub-item 1.1
    - Deep item 1.1.1
	- Tab indented item 1.1.2
- Item 2
`
	scanner := NewBlockScanner(strings.NewReader(input))
	doc, err := scanner.ScanDocument()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Children) != 1 {
		t.Fatalf("expected 1 UnorderedList child, got %d", len(doc.Children))
	}

	ul, ok := doc.Children[0].(*domain.UnorderedList)
	if !ok {
		t.Fatalf("expected *domain.UnorderedList, got %T", doc.Children[0])
	}

	if len(ul.Items) != 5 {
		t.Fatalf("expected 5 items, got %d", len(ul.Items))
	}

	expectedLevels := []int{0, 1, 2, 1, 0}
	expectedTexts := []string{
		"Item 1",
		"Sub-item 1.1",
		"Deep item 1.1.1",
		"Tab indented item 1.1.2",
		"Item 2",
	}

	for i, item := range ul.Items {
		if item.Level != expectedLevels[i] {
			t.Errorf("item %d: expected level %d, got %d", i, expectedLevels[i], item.Level)
		}
		if len(item.Children) == 0 {
			t.Fatalf("item %d: expected children, got none", i)
		}
		txt, ok := item.Children[0].(*domain.Text)
		if !ok || txt.Value != expectedTexts[i] {
			t.Errorf("item %d: expected text %q, got %v", i, expectedTexts[i], item.Children[0])
		}
	}
}

func TestBlockOrderedList(t *testing.T) {
	input := `. First auto item
. Second auto item
  . Sub auto item
1. Numbered item 1
2. Numbered item 2
`
	scanner := NewBlockScanner(strings.NewReader(input))
	doc, err := scanner.ScanDocument()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Children) != 1 {
		t.Fatalf("expected 1 OrderedList child, got %d", len(doc.Children))
	}

	ol, ok := doc.Children[0].(*domain.OrderedList)
	if !ok {
		t.Fatalf("expected *domain.OrderedList, got %T", doc.Children[0])
	}

	if len(ol.Items) != 5 {
		t.Fatalf("expected 5 items, got %d", len(ol.Items))
	}

	expectedLevels := []int{0, 0, 1, 0, 0}
	expectedTexts := []string{
		"First auto item",
		"Second auto item",
		"Sub auto item",
		"Numbered item 1",
		"Numbered item 2",
	}

	for i, item := range ol.Items {
		if item.Level != expectedLevels[i] {
			t.Errorf("item %d: expected level %d, got %d", i, expectedLevels[i], item.Level)
		}
		txt, ok := item.Children[0].(*domain.Text)
		if !ok || txt.Value != expectedTexts[i] {
			t.Errorf("item %d: expected text %q, got %v", i, expectedTexts[i], item.Children[0])
		}
	}
}

func TestBlockTaskList(t *testing.T) {
	input := `[ ] Unchecked task
[x] Checked task lowercase
[X] Checked task uppercase
  [ ] Indented task
`
	scanner := NewBlockScanner(strings.NewReader(input))
	doc, err := scanner.ScanDocument()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Children) != 1 {
		t.Fatalf("expected 1 TaskList child, got %d", len(doc.Children))
	}

	tl, ok := doc.Children[0].(*domain.TaskList)
	if !ok {
		t.Fatalf("expected *domain.TaskList, got %T", doc.Children[0])
	}

	if len(tl.Items) != 4 {
		t.Fatalf("expected 4 task items, got %d", len(tl.Items))
	}

	expectedChecked := []bool{false, true, true, false}
	expectedLevels := []int{0, 0, 0, 1}
	expectedTexts := []string{
		"Unchecked task",
		"Checked task lowercase",
		"Checked task uppercase",
		"Indented task",
	}

	for i, item := range tl.Items {
		if item.Checked != expectedChecked[i] {
			t.Errorf("item %d: expected checked %v, got %v", i, expectedChecked[i], item.Checked)
		}
		if item.Level != expectedLevels[i] {
			t.Errorf("item %d: expected level %d, got %d", i, expectedLevels[i], item.Level)
		}
		txt, ok := item.Children[0].(*domain.Text)
		if !ok || txt.Value != expectedTexts[i] {
			t.Errorf("item %d: expected text %q, got %v", i, expectedTexts[i], item.Children[0])
		}
	}
}

func TestBlockCallout(t *testing.T) {
	input := `!note: Important Architecture Note
Line 1 of callout body.
Line 2 of callout body.

!warning
Danger without title.
`
	scanner := NewBlockScanner(strings.NewReader(input))
	doc, err := scanner.ScanDocument()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Children) != 2 {
		t.Fatalf("expected 2 callouts, got %d", len(doc.Children))
	}

	c1, ok := doc.Children[0].(*domain.Callout)
	if !ok {
		t.Fatalf("expected *domain.Callout, got %T", doc.Children[0])
	}
	if c1.CalloutType != "note" {
		t.Errorf("expected callout type 'note', got %q", c1.CalloutType)
	}
	if c1.Title != "Important Architecture Note" {
		t.Errorf("expected title 'Important Architecture Note', got %q", c1.Title)
	}
	txt1, ok := c1.Children[0].(*domain.Text)
	if !ok || txt1.Value != "Line 1 of callout body.\nLine 2 of callout body." {
		t.Errorf("unexpected c1 body: %v", c1.Children[0])
	}

	c2, ok := doc.Children[1].(*domain.Callout)
	if !ok {
		t.Fatalf("expected *domain.Callout, got %T", doc.Children[1])
	}
	if c2.CalloutType != "warning" {
		t.Errorf("expected callout type 'warning', got %q", c2.CalloutType)
	}
	if c2.Title != "" {
		t.Errorf("expected empty title, got %q", c2.Title)
	}
	txt2, ok := c2.Children[0].(*domain.Text)
	if !ok || txt2.Value != "Danger without title." {
		t.Errorf("unexpected c2 body: %v", c2.Children[0])
	}
}

func TestBlockBlockquote(t *testing.T) {
	input := `> First quote line
> Second quote line
> Third quote line

Normal paragraph.
`
	scanner := NewBlockScanner(strings.NewReader(input))
	doc, err := scanner.ScanDocument()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(doc.Children))
	}

	bq, ok := doc.Children[0].(*domain.Blockquote)
	if !ok {
		t.Fatalf("expected *domain.Blockquote, got %T", doc.Children[0])
	}
	txt, ok := bq.Children[0].(*domain.Text)
	if !ok || txt.Value != "First quote line\nSecond quote line\nThird quote line" {
		t.Errorf("unexpected quote content: %v", bq.Children[0])
	}

	p, ok := doc.Children[1].(*domain.Paragraph)
	if !ok {
		t.Fatalf("expected *domain.Paragraph, got %T", doc.Children[1])
	}
	txtP, ok := p.Children[0].(*domain.Text)
	if !ok || txtP.Value != "Normal paragraph." {
		t.Errorf("unexpected paragraph content: %v", p.Children[0])
	}
}

func TestBlockCodeFence(t *testing.T) {
	input := "```go main.go\npackage main\n\nfunc main() {}\n```\n"

	scanner := NewBlockScanner(strings.NewReader(input))
	doc, err := scanner.ScanDocument()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(doc.Children))
	}

	cf, ok := doc.Children[0].(*domain.CodeFence)
	if !ok {
		t.Fatalf("expected *domain.CodeFence, got %T", doc.Children[0])
	}
	if cf.Lang != "go" {
		t.Errorf("expected lang 'go', got %q", cf.Lang)
	}
	if cf.Identifier != "main.go" {
		t.Errorf("expected identifier 'main.go', got %q", cf.Identifier)
	}
	expectedContent := "package main\n\nfunc main() {}"
	if cf.Content != expectedContent {
		t.Errorf("expected content %q, got %q", expectedContent, cf.Content)
	}

	// Unclosed code fence reaching EOF gracefully closes
	unclosed := "```python\ndef hello():\n    return 'hi'\n"
	scanner2 := NewBlockScanner(strings.NewReader(unclosed))
	doc2, err := scanner2.ScanDocument()
	if err != nil {
		t.Fatalf("unexpected error on unclosed fence: %v", err)
	}
	if len(doc2.Children) != 1 {
		t.Fatalf("expected 1 child for unclosed fence, got %d", len(doc2.Children))
	}
	cf2, ok := doc2.Children[0].(*domain.CodeFence)
	if !ok {
		t.Fatalf("expected *domain.CodeFence, got %T", doc2.Children[0])
	}
	if cf2.Lang != "python" {
		t.Errorf("expected lang 'python', got %q", cf2.Lang)
	}
	if cf2.Content != "def hello():\n    return 'hi'" {
		t.Errorf("unexpected unclosed content: %q", cf2.Content)
	}
}

func TestBlockTable(t *testing.T) {
	input := `::table [Col1, Col2, Col3]
- A1 | B1 | C1
- A2 | B2 | C2
::
`
	scanner := NewBlockScanner(strings.NewReader(input))
	doc, err := scanner.ScanDocument()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Children) != 1 {
		t.Fatalf("expected 1 Table child, got %d", len(doc.Children))
	}

	tbl, ok := doc.Children[0].(*domain.Table)
	if !ok {
		t.Fatalf("expected *domain.Table, got %T", doc.Children[0])
	}

	if len(tbl.Headers) != 3 || tbl.Headers[0] != "Col1" || tbl.Headers[1] != "Col2" || tbl.Headers[2] != "Col3" {
		t.Errorf("unexpected headers: %v", tbl.Headers)
	}

	if len(tbl.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(tbl.Rows))
	}
	if len(tbl.Rows[0]) != 3 || len(tbl.Rows[1]) != 3 {
		t.Fatalf("expected 3 columns in each row, got %d and %d", len(tbl.Rows[0]), len(tbl.Rows[1]))
	}

	cell00, ok := tbl.Rows[0][0][0].(*domain.Text)
	if !ok || cell00.Value != "A1" {
		t.Errorf("expected cell (0,0) 'A1', got %v", tbl.Rows[0][0])
	}
	cell12, ok := tbl.Rows[1][2][0].(*domain.Text)
	if !ok || cell12.Value != "C2" {
		t.Errorf("expected cell (1,2) 'C2', got %v", tbl.Rows[1][2])
	}

	// Edge case: unclosed table fence reaching EOF recovers gracefully
	unclosed := `::table [H1, H2]
- X | Y
`
	scanner2 := NewBlockScanner(strings.NewReader(unclosed))
	doc2, err := scanner2.ScanDocument()
	if err != nil {
		t.Fatalf("unexpected error on unclosed table: %v", err)
	}
	if len(doc2.Children) != 1 {
		t.Fatalf("expected 1 table child, got %d", len(doc2.Children))
	}
	tbl2, ok := doc2.Children[0].(*domain.Table)
	if !ok {
		t.Fatalf("expected *domain.Table, got %T", doc2.Children[0])
	}
	if len(tbl2.Headers) != 2 || tbl2.Headers[0] != "H1" || tbl2.Headers[1] != "H2" {
		t.Errorf("unexpected headers in unclosed table: %v", tbl2.Headers)
	}
	if len(tbl2.Rows) != 1 {
		t.Fatalf("expected 1 row in unclosed table, got %d", len(tbl2.Rows))
	}
}

func TestBlockDivider(t *testing.T) {
	input := `---
----
`
	scanner := NewBlockScanner(strings.NewReader(input))
	doc, err := scanner.ScanDocument()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(doc.Children))
	}

	if _, ok := doc.Children[0].(*domain.Divider); !ok {
		t.Errorf("expected child 0 to be *domain.Divider, got %T", doc.Children[0])
	}

	// Child 1 with 4 hyphens should degrade to paragraph
	p, ok := doc.Children[1].(*domain.Paragraph)
	if !ok {
		t.Fatalf("expected child 1 to be *domain.Paragraph, got %T", doc.Children[1])
	}
	txt, ok := p.Children[0].(*domain.Text)
	if !ok || txt.Value != "----" {
		t.Errorf("expected paragraph '----', got %v", p.Children[0])
	}
}

func TestBlockParagraph(t *testing.T) {
	input := `Line 1 of paragraph 1.
Line 2 of paragraph 1.

Line 1 of paragraph 2.
`
	scanner := NewBlockScanner(strings.NewReader(input))
	doc, err := scanner.ScanDocument()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Children) != 2 {
		t.Fatalf("expected 2 paragraphs, got %d", len(doc.Children))
	}

	p1, ok := doc.Children[0].(*domain.Paragraph)
	if !ok {
		t.Fatalf("child 0 not *domain.Paragraph: %T", doc.Children[0])
	}
	txt1 := p1.Children[0].(*domain.Text)
	if txt1.Value != "Line 1 of paragraph 1.\nLine 2 of paragraph 1." {
		t.Errorf("unexpected p1 content: %q", txt1.Value)
	}

	p2, ok := doc.Children[1].(*domain.Paragraph)
	if !ok {
		t.Fatalf("child 1 not *domain.Paragraph: %T", doc.Children[1])
	}
	txt2 := p2.Children[0].(*domain.Text)
	if txt2.Value != "Line 1 of paragraph 2." {
		t.Errorf("unexpected p2 content: %q", txt2.Value)
	}
}

func TestBlockScannerStreamingNextBlock(t *testing.T) {
	input := `# Header

Paragraph body.

---
`
	scanner := NewBlockScanner(strings.NewReader(input))

	node1, err := scanner.NextBlock()
	if err != nil {
		t.Fatalf("unexpected error on block 1: %v", err)
	}
	if _, ok := node1.(*domain.Heading); !ok {
		t.Errorf("block 1: expected *domain.Heading, got %T", node1)
	}

	node2, err := scanner.NextBlock()
	if err != nil {
		t.Fatalf("unexpected error on block 2: %v", err)
	}
	if _, ok := node2.(*domain.Paragraph); !ok {
		t.Errorf("block 2: expected *domain.Paragraph, got %T", node2)
	}

	node3, err := scanner.NextBlock()
	if err != nil {
		t.Fatalf("unexpected error on block 3: %v", err)
	}
	if _, ok := node3.(*domain.Divider); !ok {
		t.Errorf("block 3: expected *domain.Divider, got %T", node3)
	}

	nodeEnd, err := scanner.NextBlock()
	if err == nil || nodeEnd != nil {
		t.Errorf("expected EOF and nil node at end, got node=%v, err=%v", nodeEnd, err)
	}
}

func TestBlockMetadataOnAllBlockTypes(t *testing.T) {
	input := `@h_meta: h1
# Heading

@call_meta: note
!note: Note Title
Body

@quote_meta: bq
> Quoted

@code_meta: cf
` + "```go:worker.go\ncode\n```\n" + `
@tbl_meta: tbl
::table [C1]
- val
::

@ul_meta: ul
- Bullet

@ol_meta: ol
. Item

@tl_meta: tl
[ ] Task

@div_meta: div
---
`
	doc, err := ParseBlocksString(input)
	if err != nil {
		t.Fatalf("unexpected error parsing all blocks with meta: %v", err)
	}

	if len(doc.Children) != 9 {
		t.Fatalf("expected 9 block children, got %d", len(doc.Children))
	}

	h := doc.Children[0].(*domain.Heading)
	if h.Meta["h_meta"] != "h1" {
		t.Errorf("expected heading meta 'h1', got %v", h.Meta)
	}

	c := doc.Children[1].(*domain.Callout)
	if c.Meta["call_meta"] != "note" {
		t.Errorf("expected callout meta 'note', got %v", c.Meta)
	}

	bq := doc.Children[2].(*domain.Blockquote)
	if bq.Meta["quote_meta"] != "bq" {
		t.Errorf("expected blockquote meta 'bq', got %v", bq.Meta)
	}

	cf := doc.Children[3].(*domain.CodeFence)
	if cf.Meta["code_meta"] != "cf" {
		t.Errorf("expected code fence meta 'cf', got %v", cf.Meta)
	}
	if cf.Lang != "go" || cf.Identifier != "worker.go" {
		t.Errorf("expected lang 'go' and id 'worker.go', got %q and %q", cf.Lang, cf.Identifier)
	}

	tbl := doc.Children[4].(*domain.Table)
	if tbl.Meta["tbl_meta"] != "tbl" {
		t.Errorf("expected table meta 'tbl', got %v", tbl.Meta)
	}

	ul := doc.Children[5].(*domain.UnorderedList)
	if ul.Meta["ul_meta"] != "ul" {
		t.Errorf("expected unordered list meta 'ul', got %v", ul.Meta)
	}

	ol := doc.Children[6].(*domain.OrderedList)
	if ol.Meta["ol_meta"] != "ol" {
		t.Errorf("expected ordered list meta 'ol', got %v", ol.Meta)
	}

	tl := doc.Children[7].(*domain.TaskList)
	if tl.Meta["tl_meta"] != "tl" {
		t.Errorf("expected task list meta 'tl', got %v", tl.Meta)
	}

	div := doc.Children[8].(*domain.Divider)
	if div.Meta["div_meta"] != "div" {
		t.Errorf("expected divider meta 'div', got %v", div.Meta)
	}
}

func TestBlockAdjacentBlocksWithoutBlankLines(t *testing.T) {
	input := `# Heading
Paragraph immediately after heading.
---
Another paragraph.
- List item 1
- List item 2
`
	doc, err := ParseBlocksString(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(doc.Children) != 5 {
		t.Fatalf("expected 5 children, got %d", len(doc.Children))
	}

	if _, ok := doc.Children[0].(*domain.Heading); !ok {
		t.Errorf("child 0: expected Heading, got %T", doc.Children[0])
	}
	if _, ok := doc.Children[1].(*domain.Paragraph); !ok {
		t.Errorf("child 1: expected Paragraph, got %T", doc.Children[1])
	}
	if _, ok := doc.Children[2].(*domain.Divider); !ok {
		t.Errorf("child 2: expected Divider, got %T", doc.Children[2])
	}
	if _, ok := doc.Children[3].(*domain.Paragraph); !ok {
		t.Errorf("child 3: expected Paragraph, got %T", doc.Children[3])
	}
	if _, ok := doc.Children[4].(*domain.UnorderedList); !ok {
		t.Errorf("child 4: expected UnorderedList, got %T", doc.Children[4])
	}
}

func TestBlockEmptyDocument(t *testing.T) {
	doc, err := ParseBlocksString("")
	if err != nil {
		t.Fatalf("unexpected error on empty doc: %v", err)
	}
	if doc == nil {
		t.Fatal("expected non-nil doc")
	}
	if len(doc.Children) != 0 {
		t.Errorf("expected 0 children, got %d", len(doc.Children))
	}

	doc2, err := ParseBlocksString("\n\n   \n\t\n")
	if err != nil {
		t.Fatalf("unexpected error on whitespace-only doc: %v", err)
	}
	if len(doc2.Children) != 0 {
		t.Errorf("expected 0 children for whitespace-only, got %d", len(doc2.Children))
	}
}

