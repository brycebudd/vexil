package render_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"vexil/internal/domain"
	"vexil/internal/ports"
	"vexil/pkg/render"
)

// Helper to construct a comprehensive domain.Document
func sampleDocument() *domain.Document {
	return &domain.Document{
		Meta: map[string]string{
			"title":  "Sample Vexil Document",
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
					{Level: 0, Children: []domain.InlineNode{&domain.Text{Value: "Step 2"}}},
				},
			},
			&domain.TaskList{
				Items: []*domain.TaskItem{
					{Checked: false, Level: 0, Children: []domain.InlineNode{&domain.Text{Value: "Todo task"}}},
					{Checked: true, Level: 0, Children: []domain.InlineNode{&domain.Text{Value: "Done task"}}},
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
					&domain.Text{Value: "A wise quote"},
				},
			},
			&domain.CodeFence{
				Lang:       "go",
				Identifier: "sample-snippet",
				Content:    "fmt.Println(\"Hello, World!\")\n",
			},
			&domain.Table{
				Headers: []string{"Col A", "Col B"},
				Rows: [][][]domain.InlineNode{
					{
						{&domain.Text{Value: "Cell A1"}},
						{&domain.Text{Value: "Cell B1"}},
					},
					{
						{&domain.Text{Value: "Cell A2"}},
						{&domain.Text{Value: "Cell B2"}},
					},
				},
			},
			&domain.Divider{},
		},
	}
}

func TestJSON(t *testing.T) {
	t.Run("ContractVerification", func(t *testing.T) {
		var _ ports.Renderer = render.NewJSONRenderer()
	})

	t.Run("NilHandling", func(t *testing.T) {
		r := render.NewJSONRenderer()
		var buf bytes.Buffer

		if err := r.Render(nil, &buf); err == nil {
			t.Fatal("expected error when rendering nil document, got nil")
		} else if !errors.Is(err, render.ErrNilDocument) {
			t.Fatalf("expected ErrNilDocument, got %v", err)
		}

		doc := &domain.Document{}
		if err := r.Render(doc, nil); err == nil {
			t.Fatal("expected error when writer is nil, got nil")
		} else if !errors.Is(err, render.ErrNilWriter) {
			t.Fatalf("expected ErrNilWriter, got %v", err)
		}
	})

	t.Run("FullDocumentSerialization", func(t *testing.T) {
		doc := sampleDocument()
		r := render.NewJSONRenderer()

		var buf bytes.Buffer
		if err := r.Render(doc, &buf); err != nil {
			t.Fatalf("render failed: %v", err)
		}

		out := buf.String()
		if !strings.Contains(out, `"type": "document"`) {
			t.Fatalf("expected json to contain document type, got: %s", out)
		}
		if !strings.Contains(out, `"title": "Sample Vexil Document"`) {
			t.Fatalf("expected json to contain doc meta, got: %s", out)
		}

		// Verify roundtrip decoding using domain.UnmarshalNode
		decodedNode, err := domain.UnmarshalNode(buf.Bytes())
		if err != nil {
			t.Fatalf("failed to unmarshal rendered JSON: %v", err)
		}

		decodedDoc, ok := decodedNode.(*domain.Document)
		if !ok {
			t.Fatalf("expected *domain.Document, got %T", decodedNode)
		}

		if decodedDoc.Meta["title"] != "Sample Vexil Document" {
			t.Fatalf("meta title mismatch: %v", decodedDoc.Meta["title"])
		}
		if len(decodedDoc.Children) != len(doc.Children) {
			t.Fatalf("children count mismatch: expected %d, got %d", len(doc.Children), len(decodedDoc.Children))
		}
	})

	t.Run("IndentationOption", func(t *testing.T) {
		doc := &domain.Document{
			Meta: map[string]string{"k": "v"},
		}

		prettyRenderer := render.NewJSONRenderer(render.WithIndent("    "))
		var prettyBuf bytes.Buffer
		if err := prettyRenderer.Render(doc, &prettyBuf); err != nil {
			t.Fatalf("pretty render failed: %v", err)
		}
		if !strings.Contains(prettyBuf.String(), "    ") {
			t.Fatalf("expected 4 spaces indentation, got: %s", prettyBuf.String())
		}

		compactRenderer := render.NewJSONRenderer(render.WithIndent(""))
		var compactBuf bytes.Buffer
		if err := compactRenderer.Render(doc, &compactBuf); err != nil {
			t.Fatalf("compact render failed: %v", err)
		}
		if strings.Contains(compactBuf.String(), "\n") {
			t.Fatalf("expected compact JSON without newlines, got: %s", compactBuf.String())
		}
	})
}

func TestHTML(t *testing.T) {
	t.Run("ContractVerification", func(t *testing.T) {
		var _ ports.Renderer = render.NewHTMLRenderer()
	})

	t.Run("NilHandling", func(t *testing.T) {
		r := render.NewHTMLRenderer()
		var buf bytes.Buffer

		if err := r.Render(nil, &buf); err == nil {
			t.Fatal("expected error on nil doc, got nil")
		} else if !errors.Is(err, render.ErrNilDocument) {
			t.Fatalf("expected ErrNilDocument, got %v", err)
		}

		doc := &domain.Document{}
		if err := r.Render(doc, nil); err == nil {
			t.Fatal("expected error on nil writer, got nil")
		} else if !errors.Is(err, render.ErrNilWriter) {
			t.Fatalf("expected ErrNilWriter, got %v", err)
		}
	})

	t.Run("Headings", func(t *testing.T) {
		r := render.NewHTMLRenderer()
		doc := &domain.Document{
			Children: []domain.Node{
				&domain.Heading{Level: 1, Children: []domain.InlineNode{&domain.Text{Value: "H1 Title"}}},
				&domain.Heading{Level: 2, Meta: map[string]string{"id": "sub-section"}, Children: []domain.InlineNode{&domain.Text{Value: "H2 Title"}}},
				&domain.Heading{Level: 3, Children: []domain.InlineNode{&domain.Text{Value: "H3 Title"}}},
				&domain.Heading{Level: 4, Children: []domain.InlineNode{&domain.Text{Value: "H4 Title"}}},
				&domain.Heading{Level: 0, Children: []domain.InlineNode{&domain.Text{Value: "Clamped Low"}}},
				&domain.Heading{Level: 7, Children: []domain.InlineNode{&domain.Text{Value: "Clamped High"}}},
			},
		}

		var buf bytes.Buffer
		if err := r.Render(doc, &buf); err != nil {
			t.Fatalf("render headings failed: %v", err)
		}
		out := buf.String()

		expected := []string{
			"<h1>H1 Title</h1>",
			`<h2 id="sub-section">H2 Title</h2>`,
			"<h3>H3 Title</h3>",
			"<h4>H4 Title</h4>",
			"<h1>Clamped Low</h1>",
			"<h4>Clamped High</h4>",
		}
		for _, exp := range expected {
			if !strings.Contains(out, exp) {
				t.Errorf("missing expected heading %q in:\n%s", exp, out)
			}
		}
	})

	t.Run("ParagraphsAndInlines", func(t *testing.T) {
		r := render.NewHTMLRenderer()
		doc := &domain.Document{
			Children: []domain.Node{
				&domain.Paragraph{
					Children: []domain.InlineNode{
						&domain.Text{Value: "Prose with <special> & \"chars\". "},
						&domain.Strong{Children: []domain.InlineNode{&domain.Text{Value: "bold"}}},
						&domain.Text{Value: " and "},
						&domain.Emphasis{Children: []domain.InlineNode{&domain.Text{Value: "italic"}}},
						&domain.Text{Value: " and "},
						&domain.InlineCode{Value: "x < 5 && y > 10"},
						&domain.Text{Value: " and "},
						&domain.Strikethrough{Children: []domain.InlineNode{&domain.Text{Value: "deleted"}}},
						&domain.Text{Value: " and "},
						&domain.Link{
							Label:  []domain.InlineNode{&domain.Text{Value: "Go Home"}},
							Target: "https://example.com?a=1&b=2",
						},
						&domain.Text{Value: " and "},
						&domain.Media{
							Alt:    "User Avatar",
							Target: "img:https://example.com/avatar.png",
						},
						&domain.Text{Value: " and "},
						&domain.Media{
							Alt:    "Diagram",
							Target: "/images/flow.svg",
						},
					},
				},
			},
		}

		var buf bytes.Buffer
		if err := r.Render(doc, &buf); err != nil {
			t.Fatalf("render paragraph failed: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "<p>") || !strings.Contains(out, "</p>") {
			t.Fatalf("expected <p>...</p>, got: %s", out)
		}
		if !strings.Contains(out, "&lt;special&gt; &amp; &#34;chars&#34;.") {
			t.Errorf("entity escaping failed in: %s", out)
		}
		if !strings.Contains(out, "<strong>bold</strong>") {
			t.Errorf("missing strong tag in: %s", out)
		}
		if !strings.Contains(out, "<em>italic</em>") {
			t.Errorf("missing emphasis tag in: %s", out)
		}
		if !strings.Contains(out, "<code>x &lt; 5 &amp;&amp; y &gt; 10</code>") {
			t.Errorf("missing escaped code tag in: %s", out)
		}
		if !strings.Contains(out, "<s>deleted</s>") {
			t.Errorf("missing strikethrough <s> in: %s", out)
		}
		if !strings.Contains(out, `<a href="https://example.com?a=1&amp;b=2">Go Home</a>`) {
			t.Errorf("missing link in: %s", out)
		}
		if !strings.Contains(out, `<img src="https://example.com/avatar.png" alt="User Avatar">`) {
			t.Errorf("missing media image in: %s", out)
		}
		if !strings.Contains(out, `<img src="/images/flow.svg" alt="Diagram">`) {
			t.Errorf("missing media without img: prefix in: %s", out)
		}
	})

	t.Run("Lists", func(t *testing.T) {
		r := render.NewHTMLRenderer()
		doc := &domain.Document{
			Children: []domain.Node{
				&domain.UnorderedList{
					Items: []*domain.ListItem{
						{Level: 0, Children: []domain.InlineNode{&domain.Text{Value: "Top Item"}}},
						{Level: 1, Children: []domain.InlineNode{&domain.Text{Value: "Sub Item"}}},
					},
				},
				&domain.OrderedList{
					Items: []*domain.ListItem{
						{Level: 0, Children: []domain.InlineNode{&domain.Text{Value: "First"}}},
						{Level: 0, Children: []domain.InlineNode{&domain.Text{Value: "Second"}}},
					},
				},
				&domain.TaskList{
					Items: []*domain.TaskItem{
						{Checked: false, Level: 0, Children: []domain.InlineNode{&domain.Text{Value: "Incomplete"}}},
						{Checked: true, Level: 0, Children: []domain.InlineNode{&domain.Text{Value: "Complete"}}},
					},
				},
			},
		}

		var buf bytes.Buffer
		if err := r.Render(doc, &buf); err != nil {
			t.Fatalf("render lists failed: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "<ul>") || !strings.Contains(out, "<li>Top Item") {
			t.Errorf("unordered list failed in: %s", out)
		}
		if !strings.Contains(out, "<ol>") || !strings.Contains(out, "<li>First</li>") {
			t.Errorf("ordered list failed in: %s", out)
		}
		if !strings.Contains(out, `<input type="checkbox" disabled /> Incomplete`) {
			t.Errorf("unchecked task item failed in: %s", out)
		}
		if !strings.Contains(out, `<input type="checkbox" disabled checked /> Complete`) {
			t.Errorf("checked task item failed in: %s", out)
		}
	})

	t.Run("Callouts", func(t *testing.T) {
		r := render.NewHTMLRenderer()
		doc := &domain.Document{
			Children: []domain.Node{
				&domain.Callout{
					CalloutType: "warning",
					Title:       "Attention Needed",
					Children:    []domain.InlineNode{&domain.Text{Value: "Careful with inputs."}},
				},
				&domain.Callout{
					CalloutType: "info",
					Children:    []domain.InlineNode{&domain.Text{Value: "Just an informational tip."}},
				},
			},
		}

		var buf bytes.Buffer
		if err := r.Render(doc, &buf); err != nil {
			t.Fatalf("render callout failed: %v", err)
		}
		out := buf.String()

		expWarning := `<aside class="callout callout-warning"><div class="callout-title">Attention Needed</div><div class="callout-body">Careful with inputs.</div></aside>`
		if !strings.Contains(out, expWarning) {
			t.Errorf("missing callout with title: expected %q in:\n%s", expWarning, out)
		}

		expInfo := `<aside class="callout callout-info"><div class="callout-body">Just an informational tip.</div></aside>`
		if !strings.Contains(out, expInfo) {
			t.Errorf("missing callout without title: expected %q in:\n%s", expInfo, out)
		}
	})

	t.Run("Blockquotes", func(t *testing.T) {
		r := render.NewHTMLRenderer()
		doc := &domain.Document{
			Children: []domain.Node{
				&domain.Blockquote{
					Children: []domain.InlineNode{&domain.Text{Value: "Simple quote"}},
				},
			},
		}

		var buf bytes.Buffer
		if err := r.Render(doc, &buf); err != nil {
			t.Fatalf("render blockquote failed: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "<blockquote>Simple quote</blockquote>") {
			t.Errorf("blockquote mismatch in:\n%s", out)
		}
	})

	t.Run("CodeFence", func(t *testing.T) {
		r := render.NewHTMLRenderer()
		doc := &domain.Document{
			Children: []domain.Node{
				&domain.CodeFence{
					Lang:       "go",
					Identifier: "main-func",
					Content:    "func main() {\n\tprintln(\"test <>\")\n}",
				},
				&domain.CodeFence{
					Lang:    "python",
					Content: "print('hello')",
				},
				&domain.CodeFence{
					Content: "plain code",
				},
			},
		}

		var buf bytes.Buffer
		if err := r.Render(doc, &buf); err != nil {
			t.Fatalf("render code fence failed: %v", err)
		}
		out := buf.String()

		expGo := `<pre><code class="language-go" data-id="main-func">func main() {`
		if !strings.Contains(out, expGo) {
			t.Errorf("code fence with lang and id failed: expected %q in:\n%s", expGo, out)
		}
		if !strings.Contains(out, "&lt;&gt;") {
			t.Errorf("code fence HTML escaping failed in:\n%s", out)
		}
		if !strings.Contains(out, `<pre><code class="language-python">print(&#39;hello&#39;)</code></pre>`) {
			t.Errorf("code fence with only lang failed in:\n%s", out)
		}
		if !strings.Contains(out, `<pre><code>plain code</code></pre>`) {
			t.Errorf("code fence without lang failed in:\n%s", out)
		}
	})

	t.Run("Tables", func(t *testing.T) {
		r := render.NewHTMLRenderer()
		doc := &domain.Document{
			Children: []domain.Node{
				&domain.Table{
					Headers: []string{"Header 1", "Header 2"},
					Rows: [][][]domain.InlineNode{
						{
							{&domain.Text{Value: "Row 1 Col 1"}},
							{&domain.Text{Value: "Row 1 Col 2"}},
						},
					},
				},
			},
		}

		var buf bytes.Buffer
		if err := r.Render(doc, &buf); err != nil {
			t.Fatalf("render table failed: %v", err)
		}
		out := buf.String()

		expTable := `<table><thead><tr><th>Header 1</th><th>Header 2</th></tr></thead><tbody><tr><td>Row 1 Col 1</td><td>Row 1 Col 2</td></tr></tbody></table>`
		if !strings.Contains(out, expTable) {
			t.Errorf("table mismatch: expected %q in:\n%s", expTable, out)
		}
	})

	t.Run("Divider", func(t *testing.T) {
		r := render.NewHTMLRenderer()
		doc := &domain.Document{
			Children: []domain.Node{
				&domain.Divider{},
			},
		}

		var buf bytes.Buffer
		if err := r.Render(doc, &buf); err != nil {
			t.Fatalf("render divider failed: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "<hr />") {
			t.Errorf("divider mismatch in:\n%s", out)
		}
	})
}

func TestMarkdown(t *testing.T) {
	t.Run("ContractVerification", func(t *testing.T) {
		var _ ports.Renderer = render.NewMarkdownRenderer()
	})

	t.Run("NilHandling", func(t *testing.T) {
		r := render.NewMarkdownRenderer()
		var buf bytes.Buffer

		if err := r.Render(nil, &buf); err == nil {
			t.Fatal("expected error on nil doc, got nil")
		} else if !errors.Is(err, render.ErrNilDocument) {
			t.Fatalf("expected ErrNilDocument, got %v", err)
		}

		doc := &domain.Document{}
		if err := r.Render(doc, nil); err == nil {
			t.Fatal("expected error on nil writer, got nil")
		} else if !errors.Is(err, render.ErrNilWriter) {
			t.Fatalf("expected ErrNilWriter, got %v", err)
		}
	})

	t.Run("Headings", func(t *testing.T) {
		r := render.NewMarkdownRenderer()
		doc := &domain.Document{
			Children: []domain.Node{
				&domain.Heading{Level: 1, Children: []domain.InlineNode{&domain.Text{Value: "Level 1"}}},
				&domain.Heading{Level: 2, Children: []domain.InlineNode{&domain.Text{Value: "Level 2"}}},
				&domain.Heading{Level: 3, Children: []domain.InlineNode{&domain.Text{Value: "Level 3"}}},
				&domain.Heading{Level: 4, Children: []domain.InlineNode{&domain.Text{Value: "Level 4"}}},
			},
		}

		var buf bytes.Buffer
		if err := r.Render(doc, &buf); err != nil {
			t.Fatalf("render md headings failed: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "# Level 1") ||
			!strings.Contains(out, "## Level 2") ||
			!strings.Contains(out, "### Level 3") ||
			!strings.Contains(out, "#### Level 4") {
			t.Errorf("headings mismatch in markdown:\n%s", out)
		}
	})

	t.Run("InlinesTranspilation", func(t *testing.T) {
		r := render.NewMarkdownRenderer()
		doc := &domain.Document{
			Children: []domain.Node{
				&domain.Paragraph{
					Children: []domain.InlineNode{
						&domain.Strong{Children: []domain.InlineNode{&domain.Text{Value: "bold text"}}},
						&domain.Text{Value: " and "},
						&domain.Emphasis{Children: []domain.InlineNode{&domain.Text{Value: "italic text"}}},
						&domain.Text{Value: " and "},
						&domain.InlineCode{Value: "func()"},
						&domain.Text{Value: " and "},
						&domain.Strikethrough{Children: []domain.InlineNode{&domain.Text{Value: "struck"}}},
						&domain.Text{Value: " and "},
						&domain.Link{
							Label:  []domain.InlineNode{&domain.Text{Value: "Website"}},
							Target: "https://example.com",
						},
						&domain.Text{Value: " and "},
						&domain.Media{
							Alt:    "Logo",
							Target: "img:https://example.com/logo.png",
						},
					},
				},
			},
		}

		var buf bytes.Buffer
		if err := r.Render(doc, &buf); err != nil {
			t.Fatalf("render md inlines failed: %v", err)
		}
		out := buf.String()

		expectedSegments := []string{
			"**bold text**",
			"*italic text*",
			"`func()`",
			"~~struck~~",
			"[Website](https://example.com)",
			"![Logo](https://example.com/logo.png)",
		}
		for _, seg := range expectedSegments {
			if !strings.Contains(out, seg) {
				t.Errorf("missing markdown segment %q in:\n%s", seg, out)
			}
		}
	})

	t.Run("Lists", func(t *testing.T) {
		r := render.NewMarkdownRenderer()
		doc := &domain.Document{
			Children: []domain.Node{
				&domain.UnorderedList{
					Items: []*domain.ListItem{
						{Level: 0, Children: []domain.InlineNode{&domain.Text{Value: "Item 1"}}},
						{Level: 1, Children: []domain.InlineNode{&domain.Text{Value: "Item 1.1"}}},
					},
				},
				&domain.OrderedList{
					Items: []*domain.ListItem{
						{Level: 0, Children: []domain.InlineNode{&domain.Text{Value: "Step 1"}}},
						{Level: 0, Children: []domain.InlineNode{&domain.Text{Value: "Step 2"}}},
					},
				},
				&domain.TaskList{
					Items: []*domain.TaskItem{
						{Checked: false, Level: 0, Children: []domain.InlineNode{&domain.Text{Value: "Todo"}}},
						{Checked: true, Level: 0, Children: []domain.InlineNode{&domain.Text{Value: "Done"}}},
					},
				},
			},
		}

		var buf bytes.Buffer
		if err := r.Render(doc, &buf); err != nil {
			t.Fatalf("render md lists failed: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "- Item 1\n  - Item 1.1") {
			t.Errorf("unordered list markdown mismatch in:\n%s", out)
		}
		if !strings.Contains(out, "1. Step 1\n2. Step 2") {
			t.Errorf("ordered list markdown mismatch in:\n%s", out)
		}
		if !strings.Contains(out, "- [ ] Todo\n- [x] Done") {
			t.Errorf("task list markdown mismatch in:\n%s", out)
		}
	})

	t.Run("CalloutsToGitHubAlerts", func(t *testing.T) {
		r := render.NewMarkdownRenderer()
		doc := &domain.Document{
			Children: []domain.Node{
				&domain.Callout{
					CalloutType: "warning",
					Title:       "Attention",
					Children:    []domain.InlineNode{&domain.Text{Value: "Watch out for edges."}},
				},
				&domain.Callout{
					CalloutType: "note",
					Children:    []domain.InlineNode{&domain.Text{Value: "Simple note body."}},
				},
			},
		}

		var buf bytes.Buffer
		if err := r.Render(doc, &buf); err != nil {
			t.Fatalf("render md callout failed: %v", err)
		}
		out := buf.String()

		expWarning := "> [!WARNING]\n> **Attention**\n> Watch out for edges."
		if !strings.Contains(out, expWarning) {
			t.Errorf("callout alert with title mismatch: expected %q in:\n%s", expWarning, out)
		}

		expNote := "> [!NOTE]\n> Simple note body."
		if !strings.Contains(out, expNote) {
			t.Errorf("callout alert without title mismatch: expected %q in:\n%s", expNote, out)
		}
	})

	t.Run("Blockquotes", func(t *testing.T) {
		r := render.NewMarkdownRenderer()
		doc := &domain.Document{
			Children: []domain.Node{
				&domain.Blockquote{
					Children: []domain.InlineNode{&domain.Text{Value: "Line 1\nLine 2"}},
				},
			},
		}

		var buf bytes.Buffer
		if err := r.Render(doc, &buf); err != nil {
			t.Fatalf("render md blockquote failed: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "> Line 1\n> Line 2") {
			t.Errorf("blockquote markdown mismatch in:\n%s", out)
		}
	})

	t.Run("CodeFence", func(t *testing.T) {
		r := render.NewMarkdownRenderer()
		doc := &domain.Document{
			Children: []domain.Node{
				&domain.CodeFence{
					Lang:    "go",
					Content: "package main\n\nfunc main() {}\n",
				},
			},
		}

		var buf bytes.Buffer
		if err := r.Render(doc, &buf); err != nil {
			t.Fatalf("render md code fence failed: %v", err)
		}
		out := buf.String()

		exp := "```go\npackage main\n\nfunc main() {}\n```"
		if !strings.Contains(out, exp) {
			t.Errorf("code fence markdown mismatch in:\n%s", out)
		}
	})

	t.Run("Tables", func(t *testing.T) {
		r := render.NewMarkdownRenderer()
		doc := &domain.Document{
			Children: []domain.Node{
				&domain.Table{
					Headers: []string{"Header 1", "Header 2"},
					Rows: [][][]domain.InlineNode{
						{
							{&domain.Text{Value: "R1C1"}},
							{&domain.Text{Value: "R1C2"}},
						},
						{
							{&domain.Text{Value: "R2C1"}},
							{&domain.Text{Value: "R2C2"}},
						},
					},
				},
			},
		}

		var buf bytes.Buffer
		if err := r.Render(doc, &buf); err != nil {
			t.Fatalf("render md table failed: %v", err)
		}
		out := buf.String()

		expHeader := "| Header 1 | Header 2 |"
		expDivider := "|---|---|"
		expRow1 := "| R1C1 | R1C2 |"
		expRow2 := "| R2C1 | R2C2 |"

		for _, exp := range []string{expHeader, expDivider, expRow1, expRow2} {
			if !strings.Contains(out, exp) {
				t.Errorf("missing table element %q in:\n%s", exp, out)
			}
		}
	})

	t.Run("Divider", func(t *testing.T) {
		r := render.NewMarkdownRenderer()
		doc := &domain.Document{
			Children: []domain.Node{
				&domain.Divider{},
			},
		}

		var buf bytes.Buffer
		if err := r.Render(doc, &buf); err != nil {
			t.Fatalf("render md divider failed: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "---") {
			t.Errorf("divider markdown mismatch in:\n%s", out)
		}
	})
}
