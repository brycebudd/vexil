package render_test

import (
	"strings"
	"testing"

	"vexil/internal/domain"
	"vexil/internal/parser"
	"vexil/internal/ports"
	"vexil/pkg/render"
)

func TestRAG_InterfaceContract(t *testing.T) {
	var _ ports.Chunker = render.NewRAGChunker()

	chunker := render.NewRAGChunker()
	_, err := chunker.Chunk(nil)
	if err == nil {
		t.Fatal("expected error when chunking nil document, got nil")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Fatalf("expected error message to mention nil, got %v", err)
	}
}

func TestRAG_EmptyDocument(t *testing.T) {
	chunker := render.NewRAGChunker()
	doc := &domain.Document{
		Children: []domain.Node{},
	}
	chunks, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 0 {
		t.Fatalf("expected 0 chunks for empty doc, got %d", len(chunks))
	}
}

func TestRAG_PreambleChunk(t *testing.T) {
	chunker := render.NewRAGChunker()
	doc := &domain.Document{
		Meta: map[string]string{
			"version": "1.0",
			"author":  "DevOps Team",
		},
		Children: []domain.Node{
			&domain.Paragraph{
				Children: []domain.InlineNode{
					&domain.Text{Value: "This is preamble paragraph one."},
				},
			},
			&domain.Paragraph{
				Children: []domain.InlineNode{
					&domain.Text{Value: "This is preamble paragraph two."},
				},
			},
			&domain.Heading{
				Level: 1,
				Children: []domain.InlineNode{
					&domain.Text{Value: "First Section"},
				},
			},
			&domain.Paragraph{
				Children: []domain.InlineNode{
					&domain.Text{Value: "Content in first section."},
				},
			},
		},
	}

	chunks, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks (preamble + heading), got %d", len(chunks))
	}

	// Preamble chunk verification
	preamble := chunks[0]
	if preamble.Title != "Preamble" {
		t.Errorf("expected preamble title 'Preamble', got %q", preamble.Title)
	}
	if len(preamble.Breadcrumbs) != 0 {
		t.Errorf("expected empty breadcrumbs for preamble, got %v", preamble.Breadcrumbs)
	}
	expectedPreambleContent := "This is preamble paragraph one.\n\nThis is preamble paragraph two."
	if preamble.Content != expectedPreambleContent {
		t.Errorf("preamble content mismatch:\nwant: %q\ngot:  %q", expectedPreambleContent, preamble.Content)
	}
	if preamble.CharCount != len(expectedPreambleContent) {
		t.Errorf("preamble char count mismatch: want %d, got %d", len(expectedPreambleContent), preamble.CharCount)
	}
	expectedTokens := (len(expectedPreambleContent) + 3) / 4
	if preamble.TokenCount != expectedTokens {
		t.Errorf("preamble token count mismatch: want %d, got %d", expectedTokens, preamble.TokenCount)
	}
	if preamble.Meta["version"] != "1.0" || preamble.Meta["author"] != "DevOps Team" {
		t.Errorf("preamble failed to inherit doc metadata: %+v", preamble.Meta)
	}

	// Heading chunk verification
	section := chunks[1]
	if section.Title != "First Section" {
		t.Errorf("expected section title 'First Section', got %q", section.Title)
	}
	if len(section.Breadcrumbs) != 1 || section.Breadcrumbs[0] != "First Section" {
		t.Errorf("expected breadcrumbs ['First Section'], got %v", section.Breadcrumbs)
	}
	expectedSectionContent := "First Section\n\nContent in first section."
	if section.Content != expectedSectionContent {
		t.Errorf("section content mismatch:\nwant: %q\ngot:  %q", expectedSectionContent, section.Content)
	}
}

func TestRAG_HeadingHierarchyAndBreadcrumbs(t *testing.T) {
	chunker := render.NewRAGChunker()
	doc := &domain.Document{
		Children: []domain.Node{
			&domain.Heading{
				Level: 1,
				Children: []domain.InlineNode{
					&domain.Text{Value: "Deployment Spec"},
				},
			},
			&domain.Paragraph{
				Children: []domain.InlineNode{
					&domain.Text{Value: "Deployment overview."},
				},
			},
			&domain.Heading{
				Level: 2,
				Children: []domain.InlineNode{
					&domain.Text{Value: "Service Rollout"},
				},
			},
			&domain.Paragraph{
				Children: []domain.InlineNode{
					&domain.Text{Value: "Rollout details."},
				},
			},
			&domain.Heading{
				Level: 3,
				Children: []domain.InlineNode{
					&domain.Text{Value: "Drain Nodes"},
				},
			},
			&domain.Paragraph{
				Children: []domain.InlineNode{
					&domain.Text{Value: "Drain commands."},
				},
			},
			&domain.Heading{
				Level: 2,
				Children: []domain.InlineNode{
					&domain.Text{Value: "Traffic Shift"},
				},
			},
			&domain.Paragraph{
				Children: []domain.InlineNode{
					&domain.Text{Value: "Traffic shift details."},
				},
			},
			&domain.Heading{
				Level: 1,
				Children: []domain.InlineNode{
					&domain.Text{Value: "Post-Mortem"},
				},
			},
			&domain.Paragraph{
				Children: []domain.InlineNode{
					&domain.Text{Value: "Summary of incidents."},
				},
			},
		},
	}

	chunks, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != 5 {
		t.Fatalf("expected 5 chunks, got %d", len(chunks))
	}

	cases := []struct {
		title       string
		breadcrumbs []string
	}{
		{"Deployment Spec", []string{"Deployment Spec"}},
		{"Service Rollout", []string{"Deployment Spec", "Service Rollout"}},
		{"Drain Nodes", []string{"Deployment Spec", "Service Rollout", "Drain Nodes"}},
		{"Traffic Shift", []string{"Deployment Spec", "Traffic Shift"}},
		{"Post-Mortem", []string{"Post-Mortem"}},
	}

	for i, c := range cases {
		chunk := chunks[i]
		if chunk.Title != c.title {
			t.Errorf("chunk %d title mismatch: want %q, got %q", i, c.title, chunk.Title)
		}
		if len(chunk.Breadcrumbs) != len(c.breadcrumbs) {
			t.Fatalf("chunk %d breadcrumb len mismatch: want %v, got %v", i, c.breadcrumbs, chunk.Breadcrumbs)
		}
		for j := range c.breadcrumbs {
			if chunk.Breadcrumbs[j] != c.breadcrumbs[j] {
				t.Errorf("chunk %d breadcrumb[%d] mismatch: want %q, got %q", i, j, c.breadcrumbs[j], chunk.Breadcrumbs[j])
			}
		}
	}
}

func TestRAG_TableSemanticChunk(t *testing.T) {
	chunker := render.NewRAGChunker()
	doc := &domain.Document{
		Meta: map[string]string{"env": "production"},
		Children: []domain.Node{
			&domain.Heading{
				Level: 1,
				Children: []domain.InlineNode{
					&domain.Text{Value: "Node Status"},
				},
			},
			&domain.Paragraph{
				Children: []domain.InlineNode{
					&domain.Text{Value: "Overview paragraph before table."},
				},
			},
			&domain.Table{
				Meta: map[string]string{
					"title": "Node Inventory",
					"id":    "tbl-nodes",
				},
				Headers: []string{"Node", "Role", "Status"},
				Rows: [][][]domain.InlineNode{
					{
						{&domain.Text{Value: "worker-01"}},
						{&domain.Text{Value: "compute"}},
						{&domain.Text{Value: "Ready"}},
					},
					{
						{&domain.Text{Value: "worker-02"}},
						{&domain.Text{Value: "storage"}},
						{&domain.Text{Value: "Ready"}},
					},
				},
			},
			&domain.Paragraph{
				Children: []domain.InlineNode{
					&domain.Text{Value: "Follow-up paragraph after table."},
				},
			},
		},
	}

	chunks, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks (heading prose, table, post-table prose), got %d", len(chunks))
	}

	// Chunk 1: Heading Prose
	c1 := chunks[0]
	if c1.Title != "Node Status" {
		t.Errorf("chunk 1 title mismatch: %q", c1.Title)
	}
	if len(c1.Breadcrumbs) != 1 || c1.Breadcrumbs[0] != "Node Status" {
		t.Errorf("chunk 1 breadcrumbs mismatch: %v", c1.Breadcrumbs)
	}
	expectedC1 := "Node Status\n\nOverview paragraph before table."
	if c1.Content != expectedC1 {
		t.Errorf("chunk 1 content mismatch: want %q, got %q", expectedC1, c1.Content)
	}

	// Chunk 2: Table
	c2 := chunks[1]
	if c2.ID != "tbl-nodes" {
		t.Errorf("expected chunk 2 ID 'tbl-nodes', got %q", c2.ID)
	}
	if c2.Title != "Node Inventory" {
		t.Errorf("chunk 2 title mismatch: %q", c2.Title)
	}
	if len(c2.Breadcrumbs) != 1 || c2.Breadcrumbs[0] != "Node Status" {
		t.Errorf("chunk 2 breadcrumbs mismatch: %v", c2.Breadcrumbs)
	}
	expectedC2 := "Node | Role | Status\nworker-01 | compute | Ready\nworker-02 | storage | Ready"
	if c2.Content != expectedC2 {
		t.Errorf("chunk 2 table content mismatch:\nwant:\n%s\ngot:\n%s", expectedC2, c2.Content)
	}
	if c2.Meta["env"] != "production" || c2.Meta["title"] != "Node Inventory" {
		t.Errorf("chunk 2 metadata mismatch: %+v", c2.Meta)
	}

	// Chunk 3: Post-table Prose
	c3 := chunks[2]
	if c3.Title != "Node Status" {
		t.Errorf("chunk 3 title mismatch: %q", c3.Title)
	}
	if len(c3.Breadcrumbs) != 1 || c3.Breadcrumbs[0] != "Node Status" {
		t.Errorf("chunk 3 breadcrumbs mismatch: %v", c3.Breadcrumbs)
	}
	if c3.Content != "Follow-up paragraph after table." {
		t.Errorf("chunk 3 content mismatch: %q", c3.Content)
	}
}

func TestRAG_MetadataPreservationAndInheritance(t *testing.T) {
	chunker := render.NewRAGChunker()
	doc := &domain.Document{
		Meta: map[string]string{
			"project": "Vexil",
			"tier":    "infra",
		},
		Children: []domain.Node{
			&domain.Heading{
				Level: 1,
				Meta: map[string]string{
					"tier": "apps", // Overrides document level
					"id":   "h-apps",
				},
				Children: []domain.InlineNode{
					&domain.Text{Value: "Applications"},
				},
			},
			&domain.Paragraph{
				Children: []domain.InlineNode{
					&domain.Text{Value: "Application configurations."},
				},
			},
		},
	}

	chunks, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}

	chunk := chunks[0]
	if chunk.ID != "h-apps" {
		t.Errorf("expected chunk ID 'h-apps', got %q", chunk.ID)
	}
	if chunk.Meta["project"] != "Vexil" {
		t.Errorf("expected inherited project 'Vexil', got %q", chunk.Meta["project"])
	}
	if chunk.Meta["tier"] != "apps" {
		t.Errorf("expected overridden tier 'apps', got %q", chunk.Meta["tier"])
	}

	// Verify original doc meta was not mutated
	if doc.Meta["tier"] != "infra" {
		t.Errorf("original doc meta mutated: got %q, want 'infra'", doc.Meta["tier"])
	}
}

func TestRAG_MetricsCalculation(t *testing.T) {
	chunker := render.NewRAGChunker()
	doc := &domain.Document{
		Children: []domain.Node{
			&domain.Heading{
				Level: 1,
				Children: []domain.InlineNode{
					&domain.Text{Value: "Content"},
				},
			},
		},
	}

	chunks, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}

	chunk := chunks[0]
	if chunk.Content != "Content" {
		t.Fatalf("expected content 'Content', got %q", chunk.Content)
	}
	// len("Content") == 7, (7+3)/4 == 2
	if chunk.CharCount != 7 {
		t.Errorf("expected CharCount 7, got %d", chunk.CharCount)
	}
	if chunk.TokenCount != 2 {
		t.Errorf("expected TokenCount 2, got %d", chunk.TokenCount)
	}
}

func TestRAG_AllBlockAndInlineTypes(t *testing.T) {
	chunker := render.NewRAGChunker()
	doc := &domain.Document{
		Children: []domain.Node{
			&domain.Heading{
				Level: 1,
				Children: []domain.InlineNode{
					&domain.Text{Value: "Heading with "},
					&domain.Strong{Children: []domain.InlineNode{&domain.Text{Value: "Bold"}}},
					&domain.Text{Value: " and "},
					&domain.Emphasis{Children: []domain.InlineNode{&domain.Text{Value: "Italic"}}},
				},
			},
			&domain.Paragraph{
				Children: []domain.InlineNode{
					&domain.Text{Value: "Paragraph with "},
					&domain.InlineCode{Value: "code()"},
					&domain.Text{Value: ", "},
					&domain.Strikethrough{Children: []domain.InlineNode{&domain.Text{Value: "strike"}}},
					&domain.Text{Value: ", "},
					&domain.Link{Label: []domain.InlineNode{&domain.Text{Value: "LinkText"}}, Target: "https://example.com"},
					&domain.Text{Value: ", and "},
					&domain.Media{Alt: "AltText", Target: "img:pic.png"},
					&domain.Text{Value: "."},
				},
			},
			&domain.UnorderedList{
				Items: []*domain.ListItem{
					{Level: 0, Children: []domain.InlineNode{&domain.Text{Value: "Item A"}}},
					{Level: 1, Children: []domain.InlineNode{&domain.Text{Value: "Sub-item A1"}}},
				},
			},
			&domain.OrderedList{
				Items: []*domain.ListItem{
					{Level: 0, Children: []domain.InlineNode{&domain.Text{Value: "First step"}}},
				},
			},
			&domain.TaskList{
				Items: []*domain.TaskItem{
					{Level: 0, Checked: false, Children: []domain.InlineNode{&domain.Text{Value: "Unchecked"}}},
					{Level: 0, Checked: true, Children: []domain.InlineNode{&domain.Text{Value: "Checked"}}},
				},
			},
			&domain.Callout{
				CalloutType: "warning",
				Title:       "Watch Out",
				Children: []domain.InlineNode{
					&domain.Text{Value: "Important safety message."},
				},
			},
			&domain.Blockquote{
				Children: []domain.InlineNode{
					&domain.Text{Value: "A quoted statement."},
				},
			},
			&domain.CodeFence{
				Lang:    "yaml",
				Content: "version: '3'\nservices:\n  app:\n    image: nginx\n",
			},
			&domain.Divider{},
		},
	}

	chunks, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}

	chunk := chunks[0]
	// Verify that clean text does not contain raw inline delimiters
	if strings.Contains(chunk.Content, "*Bold*") || strings.Contains(chunk.Content, "_Italic_") {
		t.Errorf("chunk content contains raw inline delimiters: %s", chunk.Content)
	}
	if !strings.Contains(chunk.Content, "Heading with Bold and Italic") {
		t.Errorf("missing flattened heading in content: %s", chunk.Content)
	}
	if !strings.Contains(chunk.Content, "Paragraph with code(), strike, LinkText, and AltText.") {
		t.Errorf("missing flattened paragraph in content: %s", chunk.Content)
	}
	if !strings.Contains(chunk.Content, "- Item A\n  - Sub-item A1") {
		t.Errorf("missing unordered list formatting in content: %s", chunk.Content)
	}
	if !strings.Contains(chunk.Content, "1. First step") {
		t.Errorf("missing ordered list formatting in content: %s", chunk.Content)
	}
	if !strings.Contains(chunk.Content, "[ ] Unchecked\n[x] Checked") {
		t.Errorf("missing task list formatting in content: %s", chunk.Content)
	}
	if !strings.Contains(chunk.Content, "Watch Out\nImportant safety message.") {
		t.Errorf("missing callout formatting in content: %s", chunk.Content)
	}
	if !strings.Contains(chunk.Content, "A quoted statement.") {
		t.Errorf("missing blockquote in content: %s", chunk.Content)
	}
	if !strings.Contains(chunk.Content, "version: '3'\nservices:\n  app:\n    image: nginx") {
		t.Errorf("missing code fence in content: %s", chunk.Content)
	}
}

func TestRAG_IntegrationWithParser(t *testing.T) {
	vxlSource := `@title: System Architecture
@version: 2.1

This is the system architecture specification document.
It describes high-level components and networking.

# Core Services

The platform consists of stateless microservices.

## Gateway

The API gateway handles routing and rate limiting.

@id: tbl-endpoints
@title: Endpoint Definitions
::table [Path, Method, Auth]
- /api/v1/health | GET | Public
- /api/v1/auth | POST | Public
- /api/v1/users | GET | Bearer
::

Gateway logs are forwarded to OpenTelemetry collector.

## Worker Fleet

!warning: Background Queues
Ensure worker concurrency does not exceed DB connection pool.

- Worker instances consume from RabbitMQ.
- Retries follow exponential backoff.
`

	doc, err := parser.NewParser().ParseString(vxlSource)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	chunks, err := render.NewRAGChunker().Chunk(doc)
	if err != nil {
		t.Fatalf("chunk failed: %v", err)
	}

	// Chunks expected:
	// 1. Preamble ("This is the system architecture...")
	// 2. Heading "Core Services"
	// 3. Heading "Gateway" (content before table)
	// 4. Table "tbl-endpoints"
	// 5. Heading "Gateway" (content after table: "Gateway logs...")
	// 6. Heading "Worker Fleet" (with callout and list)
	if len(chunks) != 6 {
		for i, c := range chunks {
			t.Logf("Chunk %d: ID=%s Title=%q Breadcrumbs=%v Content=%q", i, c.ID, c.Title, c.Breadcrumbs, c.Content)
		}
		t.Fatalf("expected 6 chunks, got %d", len(chunks))
	}

	// Validate chunk 0: Preamble
	if chunks[0].Title != "Preamble" {
		t.Errorf("chunk 0 title want 'Preamble', got %q", chunks[0].Title)
	}
	if len(chunks[0].Breadcrumbs) != 0 {
		t.Errorf("chunk 0 breadcrumbs want empty, got %v", chunks[0].Breadcrumbs)
	}
	if chunks[0].Meta["title"] != "System Architecture" || chunks[0].Meta["version"] != "2.1" {
		t.Errorf("chunk 0 meta mismatch: %+v", chunks[0].Meta)
	}

	// Validate chunk 3: Table
	tableChunk := chunks[3]
	if tableChunk.ID != "tbl-endpoints" {
		t.Errorf("table chunk ID want 'tbl-endpoints', got %q", tableChunk.ID)
	}
	if tableChunk.Title != "Endpoint Definitions" {
		t.Errorf("table chunk Title want 'Endpoint Definitions', got %q", tableChunk.Title)
	}
	if len(tableChunk.Breadcrumbs) != 2 || tableChunk.Breadcrumbs[0] != "Core Services" || tableChunk.Breadcrumbs[1] != "Gateway" {
		t.Errorf("table chunk breadcrumbs want ['Core Services', 'Gateway'], got %v", tableChunk.Breadcrumbs)
	}
	if !strings.Contains(tableChunk.Content, "Path | Method | Auth") {
		t.Errorf("table chunk content missing headers: %s", tableChunk.Content)
	}

	// Validate chunk 5: Worker Fleet
	workerChunk := chunks[5]
	if workerChunk.Title != "Worker Fleet" {
		t.Errorf("worker chunk Title want 'Worker Fleet', got %q", workerChunk.Title)
	}
	if len(workerChunk.Breadcrumbs) != 2 || workerChunk.Breadcrumbs[0] != "Core Services" || workerChunk.Breadcrumbs[1] != "Worker Fleet" {
		t.Errorf("worker chunk breadcrumbs want ['Core Services', 'Worker Fleet'], got %v", workerChunk.Breadcrumbs)
	}
	if !strings.Contains(workerChunk.Content, "Background Queues") {
		t.Errorf("worker chunk content missing callout: %s", workerChunk.Content)
	}
	if !strings.Contains(workerChunk.Content, "- Worker instances consume from RabbitMQ.") {
		t.Errorf("worker chunk content missing list items: %s", workerChunk.Content)
	}
}

func TestRAG_StandaloneTableWithoutHeadings(t *testing.T) {
	chunker := render.NewRAGChunker()
	doc := &domain.Document{
		Children: []domain.Node{
			&domain.Table{
				Headers: []string{"Col1", "Col2"},
				Rows: [][][]domain.InlineNode{
					{
						{&domain.Text{Value: "Val1"}},
						{&domain.Text{Value: "Val2"}},
					},
				},
			},
		},
	}

	chunks, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}

	chunk := chunks[0]
	if chunk.Title != "Table" {
		t.Errorf("expected title 'Table', got %q", chunk.Title)
	}
	if len(chunk.Breadcrumbs) != 0 {
		t.Errorf("expected empty breadcrumbs, got %v", chunk.Breadcrumbs)
	}
	if chunk.Content != "Col1 | Col2\nVal1 | Val2" {
		t.Errorf("unexpected content: %q", chunk.Content)
	}
}

func TestRAG_OptionsAndHelperFunction(t *testing.T) {
	doc := &domain.Document{
		Children: []domain.Node{
			&domain.Paragraph{
				Children: []domain.InlineNode{
					&domain.Text{Value: "1234567890"}, // 10 chars
				},
			},
		},
	}

	// Default ratio (4.0): (10 + 3) / 4 = 3 tokens
	chunksDefault, err := render.ChunkRAG(doc)
	if err != nil {
		t.Fatalf("ChunkRAG failed: %v", err)
	}
	if len(chunksDefault) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunksDefault))
	}
	if chunksDefault[0].CharCount != 10 {
		t.Errorf("expected CharCount 10, got %d", chunksDefault[0].CharCount)
	}
	if chunksDefault[0].TokenCount != 3 {
		t.Errorf("expected default TokenCount 3, got %d", chunksDefault[0].TokenCount)
	}

	// Custom ratio (2.0): (10 + 2 - 1) / 2 = 5 tokens
	chunkerCustom := render.NewRAGChunker(render.WithTokenRatio(2.0))
	chunksCustom, err := chunkerCustom.Chunk(doc)
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}
	if chunksCustom[0].TokenCount != 5 {
		t.Errorf("expected custom TokenCount 5, got %d", chunksCustom[0].TokenCount)
	}
}

func TestRAG_ConsecutiveTables(t *testing.T) {
	doc := &domain.Document{
		Children: []domain.Node{
			&domain.Table{
				Meta:    map[string]string{"caption": "First Table"},
				Headers: []string{"A"},
				Rows:    [][][]domain.InlineNode{{{&domain.Text{Value: "1"}}}},
			},
			&domain.Table{
				Meta:    map[string]string{"name": "Second Table"},
				Headers: []string{"B"},
				Rows:    [][][]domain.InlineNode{{{&domain.Text{Value: "2"}}}},
			},
		},
	}

	chunks, err := render.NewRAGChunker().Chunk(doc)
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}

	if chunks[0].Title != "First Table" {
		t.Errorf("chunk 0 title want 'First Table', got %q", chunks[0].Title)
	}
	if chunks[1].Title != "Second Table" {
		t.Errorf("chunk 1 title want 'Second Table', got %q", chunks[1].Title)
	}
}

func TestRAG_DeepHeadingSkips(t *testing.T) {
	doc := &domain.Document{
		Children: []domain.Node{
			&domain.Heading{
				Level: 1,
				Children: []domain.InlineNode{
					&domain.Text{Value: "Top Level"},
				},
			},
			&domain.Heading{
				Level: 4,
				Children: []domain.InlineNode{
					&domain.Text{Value: "Deep Dive"},
				},
			},
			&domain.Heading{
				Level: 2,
				Children: []domain.InlineNode{
					&domain.Text{Value: "Mid Level"},
				},
			},
		},
	}

	chunks, err := render.NewRAGChunker().Chunk(doc)
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}

	// Chunk 0: Top Level -> [Top Level]
	if len(chunks[0].Breadcrumbs) != 1 || chunks[0].Breadcrumbs[0] != "Top Level" {
		t.Errorf("chunk 0 breadcrumbs mismatch: %v", chunks[0].Breadcrumbs)
	}
	// Chunk 1: Deep Dive -> [Top Level, Deep Dive]
	if len(chunks[1].Breadcrumbs) != 2 || chunks[1].Breadcrumbs[0] != "Top Level" || chunks[1].Breadcrumbs[1] != "Deep Dive" {
		t.Errorf("chunk 1 breadcrumbs mismatch: %v", chunks[1].Breadcrumbs)
	}
	// Chunk 2: Mid Level -> [Top Level, Mid Level] (Deep Dive popped because level 4 >= 2)
	if len(chunks[2].Breadcrumbs) != 2 || chunks[2].Breadcrumbs[0] != "Top Level" || chunks[2].Breadcrumbs[1] != "Mid Level" {
		t.Errorf("chunk 2 breadcrumbs mismatch: %v", chunks[2].Breadcrumbs)
	}
}

