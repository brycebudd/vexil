package formatter_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"vexil/internal/ports"
	"vexil/pkg/formatter"
)

// faultyReader simulates an I/O read failure after emitting a few bytes.
type faultyReader struct {
	err error
}

func (r *faultyReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

// faultyWriter simulates an I/O write failure.
type faultyWriter struct {
	err error
}

func (w *faultyWriter) Write(p []byte) (n int, err error) {
	return 0, w.err
}

func TestFormatterContract(t *testing.T) {
	var _ ports.Formatter = formatter.NewFormatter()
	var _ ports.Formatter = (*formatter.Formatter)(nil)

	f := formatter.NewFormatter()
	input := "# Hello\n"
	var out bytes.Buffer

	if err := f.Format(strings.NewReader(input), &out); err != nil {
		t.Fatalf("Format failed: %v", err)
	}
	if out.String() != input {
		t.Errorf("Format output mismatch: got %q, want %q", out.String(), input)
	}

	strOut, err := f.FormatString(input)
	if err != nil {
		t.Fatalf("FormatString failed: %v", err)
	}
	if strOut != input {
		t.Errorf("FormatString output mismatch: got %q, want %q", strOut, input)
	}

	bytesOut, err := f.FormatBytes([]byte(input))
	if err != nil {
		t.Fatalf("FormatBytes failed: %v", err)
	}
	if string(bytesOut) != input {
		t.Errorf("FormatBytes output mismatch: got %q, want %q", string(bytesOut), input)
	}

	// Package-level convenience functions
	var pkgOut bytes.Buffer
	if err := formatter.Format(strings.NewReader(input), &pkgOut); err != nil {
		t.Fatalf("package Format failed: %v", err)
	}
	if pkgOut.String() != input {
		t.Errorf("package Format output mismatch: got %q, want %q", pkgOut.String(), input)
	}

	pkgStr, err := formatter.FormatString(input)
	if err != nil {
		t.Fatalf("package FormatString failed: %v", err)
	}
	if pkgStr != input {
		t.Errorf("package FormatString output mismatch: got %q, want %q", pkgStr, input)
	}

	pkgBytes, err := formatter.FormatBytes([]byte(input))
	if err != nil {
		t.Fatalf("package FormatBytes failed: %v", err)
	}
	if string(pkgBytes) != input {
		t.Errorf("package FormatBytes output mismatch: got %q, want %q", string(pkgBytes), input)
	}
}

func TestBlankLinesNormalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty document",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only document",
			input:    "   \n\n\t  \n",
			expected: "",
		},
		{
			name:     "leading blank lines stripped",
			input:    "\n\n\n# Title\nParagraph",
			expected: "# Title\nParagraph\n",
		},
		{
			name:     "trailing blank lines stripped to single newline",
			input:    "# Title\nParagraph\n\n\n\n",
			expected: "# Title\nParagraph\n",
		},
		{
			name:     "consecutive blank lines collapsed to at most one",
			input:    "# Title\n\n\n\nParagraph 1\n\n\n\n\nParagraph 2\n",
			expected: "# Title\n\nParagraph 1\n\nParagraph 2\n",
		},
		{
			name:     "single newline at EOF ensured when missing",
			input:    "# Title",
			expected: "# Title\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := formatter.FormatString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("mismatch:\ngot:\n%q\nwant:\n%q", got, tt.expected)
			}
		})
	}
}

func TestIndentationNormalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "tabs converted to two spaces",
			input: "- Item 1\n" +
				"\t- Item 1.1\n" +
				"\t\t- Item 1.1.1\n",
			expected: "- Item 1\n" +
				"  - Item 1.1\n" +
				"    - Item 1.1.1\n",
		},
		{
			name: "odd spaces normalized by two-space increments",
			input: "- Item 1\n" +
				"   - Item 1.1 with 3 spaces\n" +
				"     - Item 1.1.1 with 5 spaces\n",
			expected: "- Item 1\n" +
				"  - Item 1.1 with 3 spaces\n" +
				"    - Item 1.1.1 with 5 spaces\n",
		},
		{
			name: "mixed tabs and spaces",
			input: "1. Step 1\n" +
				" \t. Step 1.1\n",
			expected: "1. Step 1\n" +
				"  . Step 1.1\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := formatter.FormatString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("mismatch:\ngot:\n%q\nwant:\n%q", got, tt.expected)
			}
		})
	}
}

func TestTrailingWhitespace(t *testing.T) {
	input := "# Heading with trailing spaces   \n" +
		"Paragraph line with tabs\t\t\n" +
		"- List item with spaces \t \n"
	expected := "# Heading with trailing spaces\n" +
		"Paragraph line with tabs\n" +
		"- List item with spaces\n"

	got, err := formatter.FormatString(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != expected {
		t.Errorf("mismatch:\ngot:\n%q\nwant:\n%q", got, expected)
	}
}

func TestCodeFenceVerbatimPreservation(t *testing.T) {
	// Inside code fences, indentation, tabs, blank lines, and trailing spaces MUST be preserved verbatim.
	input := "```go main.go\n" +
		"package main\n" +
		"\n" +
		"func main() {\n" +
		"\t// indented with tab\n" +
		"   // indented with 3 spaces\n" +
		"}\n" +
		"```\n"

	got, err := formatter.FormatString(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != input {
		t.Errorf("code fence altered:\ngot:\n%s\nwant:\n%s", got, input)
	}
}

func TestTableFormatting(t *testing.T) {
	t.Run("standard spacing around delimiters", func(t *testing.T) {
		input := "::table [Col1,Col2,  Col3]\n" +
			"- a|b|c\n" +
			"- alpha |beta| gamma\n" +
			"::\n"
		expected := "::table [Col1, Col2, Col3]\n" +
			"- a | b | c\n" +
			"- alpha | beta | gamma\n" +
			"::\n"

		got, err := formatter.FormatString(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != expected {
			t.Errorf("mismatch:\ngot:\n%s\nwant:\n%s", got, expected)
		}
	})

	t.Run("table with escaped pipe and code spans", func(t *testing.T) {
		input := "::table [Endpoint, Signature, Protocol]\n" +
			"- /api/v1 \\| primary|`func(a | b)`|HTTP/2\n" +
			"::\n"
		expected := "::table [Endpoint, Signature, Protocol]\n" +
			"- /api/v1 \\| primary | `func(a | b)` | HTTP/2\n" +
			"::\n"

		got, err := formatter.FormatString(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != expected {
			t.Errorf("mismatch:\ngot:\n%s\nwant:\n%s", got, expected)
		}
	})

	t.Run("table column alignment option", func(t *testing.T) {
		input := "::table [Name, Role]\n" +
			"- Alexander|Engineer\n" +
			"- Bo|Dev\n" +
			"::\n"
		expected := "::table [Name, Role]\n" +
			"- Alexander | Engineer\n" +
			"- Bo        | Dev\n" +
			"::\n"

		f := formatter.NewFormatter(formatter.WithAlignTableColumns(true))
		got, err := f.FormatString(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != expected {
			t.Errorf("aligned table mismatch:\ngot:\n%s\nwant:\n%s", got, expected)
		}
	})
}

func TestBlockTypesPreservation(t *testing.T) {
	input := "@title: Canonical Test Document\n" +
		"@version: 1.0.0\n" +
		"\n" +
		"# Top Heading\n" +
		"## Subheading\n" +
		"### H3 Level\n" +
		"#### H4 Level\n" +
		"\n" +
		"Paragraph with *bold*, _italic_, `code`, and ~strike~.\n" +
		"\n" +
		"- Unordered item 1\n" +
		"  - Nested item 1.1\n" +
		"\n" +
		"1. First step\n" +
		"  2. Second step\n" +
		"  . Third step with dot\n" +
		"\n" +
		"[ ] Pending task\n" +
		"  [x] Completed subtask\n" +
		"[X] Uppercase completed task\n" +
		"\n" +
		"!warning: Security Notice\n" +
		"Never commit raw secret credentials.\n" +
		"\n" +
		"> Quoted message line 1\n" +
		"> Quoted message line 2\n" +
		"\n" +
		"---\n" +
		"\n" +
		"Final closing paragraph.\n"

	got, err := formatter.FormatString(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != input {
		t.Errorf("block types altered:\ngot:\n%s\nwant:\n%s", got, input)
	}
}

func TestIdempotency(t *testing.T) {
	inputs := []string{
		"",
		"# Heading\n",
		"@title: Vexil\n\n# Overview\nSome text.\n",
		`@title: Full Integration Document
@version: 1.0.0
@author: Vexil Team

# Section 1: Overview
This is a comprehensive document for testing *streaming* line-by-line parsing.
It includes consecutive paragraph lines that must merge properly.

## Sub-section 1.1: Lists
- Unordered item 1 with ` + "`inline code`" + `
  - Nested unordered item 1.1 with *bold* text
    - Deep nested 1.1.1
- Unordered item 2 with [link](https://vexil.org)

1. First ordered step
  2. Nested ordered sub-step
3. Third ordered step

[ ] Initial pending task
  [x] Completed subtask
[X] Another completed task

!warning: Security Notice
Do not expose private credentials.
Always sanitize incoming *user input*.

> This is a blockquote.
> It spans across multiple lines with _emphasis_.

::table [Service, Port, Protocol]
- Web Server | 8080 | HTTP/1.1
- DB Cluster \| Primary | 5432 | TCP
- Cache Node | ` + "`6379 | redis`" + ` | TCP
::

` + "```go server.go\npackage main\n\nfunc main() {}\n```" + `

---

Final concluding paragraph with ~strike~ text.
`,
	}

	for i, input := range inputs {
		firstPass, err := formatter.FormatString(input)
		if err != nil {
			t.Fatalf("case %d: first pass failed: %v", i, err)
		}
		secondPass, err := formatter.FormatString(firstPass)
		if err != nil {
			t.Fatalf("case %d: second pass failed: %v", i, err)
		}
		if firstPass != secondPass {
			t.Errorf("case %d failed idempotency:\nfirst pass:\n%s\nsecond pass:\n%s", i, firstPass, secondPass)
		}
	}
}

func TestErrorHandling(t *testing.T) {
	expectedErr := errors.New("simulated I/O failure")

	t.Run("reader error is wrapped", func(t *testing.T) {
		r := &faultyReader{err: expectedErr}
		var out bytes.Buffer
		err := formatter.Format(r, &out)
		if err == nil {
			t.Fatal("expected error from faulty reader, got nil")
		}
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected wrapped error %v, got %v", expectedErr, err)
		}
	})

	t.Run("writer error is wrapped", func(t *testing.T) {
		w := &faultyWriter{err: expectedErr}
		err := formatter.Format(strings.NewReader("# Valid Heading\n"), w)
		if err == nil {
			t.Fatal("expected error from faulty writer, got nil")
		}
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected wrapped error %v, got %v", expectedErr, err)
		}
	})
}
