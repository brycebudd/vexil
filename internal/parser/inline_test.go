package parser_test

import (
	"reflect"
	"testing"

	"vexil/internal/domain"
	"vexil/internal/parser"
)

func TestInlinePlainText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []domain.InlineNode
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "simple prose",
			input:    "Hello, world!",
			expected: []domain.InlineNode{&domain.Text{Value: "Hello, world!"}},
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: []domain.InlineNode{&domain.Text{Value: "   "}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parser.ParseInline(tc.input)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Fatalf("ParseInline(%q) = %#v; want %#v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestInlineSymmetricDelimiters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []domain.InlineNode
	}{
		{
			name:  "strong bold",
			input: "*bold*",
			expected: []domain.InlineNode{
				&domain.Strong{
					Children: []domain.InlineNode{&domain.Text{Value: "bold"}},
				},
			},
		},
		{
			name:  "emphasis italic",
			input: "_italic_",
			expected: []domain.InlineNode{
				&domain.Emphasis{
					Children: []domain.InlineNode{&domain.Text{Value: "italic"}},
				},
			},
		},
		{
			name:  "inline code verbatim",
			input: "`code`",
			expected: []domain.InlineNode{
				&domain.InlineCode{Value: "code"},
			},
		},
		{
			name:  "strikethrough",
			input: "~strike~",
			expected: []domain.InlineNode{
				&domain.Strikethrough{
					Children: []domain.InlineNode{&domain.Text{Value: "strike"}},
				},
			},
		},
		{
			name:  "in-word bold formatting",
			input: "foo*bar*baz",
			expected: []domain.InlineNode{
				&domain.Text{Value: "foo"},
				&domain.Strong{
					Children: []domain.InlineNode{&domain.Text{Value: "bar"}},
				},
				&domain.Text{Value: "baz"},
			},
		},
		{
			name:  "formatting adjacent to punctuation",
			input: "(*bold*)",
			expected: []domain.InlineNode{
				&domain.Text{Value: "("},
				&domain.Strong{
					Children: []domain.InlineNode{&domain.Text{Value: "bold"}},
				},
				&domain.Text{Value: ")"},
			},
		},
		{
			name:  "multiple sequential spans",
			input: "*bold* and _italic_ and ~strike~ and `code`",
			expected: []domain.InlineNode{
				&domain.Strong{Children: []domain.InlineNode{&domain.Text{Value: "bold"}}},
				&domain.Text{Value: " and "},
				&domain.Emphasis{Children: []domain.InlineNode{&domain.Text{Value: "italic"}}},
				&domain.Text{Value: " and "},
				&domain.Strikethrough{Children: []domain.InlineNode{&domain.Text{Value: "strike"}}},
				&domain.Text{Value: " and "},
				&domain.InlineCode{Value: "code"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parser.ParseInline(tc.input)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Fatalf("ParseInline(%q) = %#v; want %#v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestInlineWhitespaceAdjacencyRule(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []domain.InlineNode
	}{
		{
			name:     "space after opening asterisk degrades to literal text",
			input:    "* bold*",
			expected: []domain.InlineNode{&domain.Text{Value: "* bold*"}},
		},
		{
			name:     "space before closing asterisk degrades to literal text",
			input:    "*bold *",
			expected: []domain.InlineNode{&domain.Text{Value: "*bold *"}},
		},
		{
			name:     "space after opening underscore degrades",
			input:    "_ italic_",
			expected: []domain.InlineNode{&domain.Text{Value: "_ italic_"}},
		},
		{
			name:     "space before closing underscore degrades",
			input:    "_italic _",
			expected: []domain.InlineNode{&domain.Text{Value: "_italic _"}},
		},
		{
			name:     "space after opening tilde degrades",
			input:    "~ strike~",
			expected: []domain.InlineNode{&domain.Text{Value: "~ strike~"}},
		},
		{
			name:     "space before closing tilde degrades",
			input:    "~strike ~",
			expected: []domain.InlineNode{&domain.Text{Value: "~strike ~"}},
		},
		{
			name:     "space after opening backtick degrades",
			input:    "` code`",
			expected: []domain.InlineNode{&domain.Text{Value: "` code`"}},
		},
		{
			name:     "space before closing backtick degrades",
			input:    "`code `",
			expected: []domain.InlineNode{&domain.Text{Value: "`code `"}},
		},
		{
			name:     "isolated mathematical asterisk",
			input:    "5 * 4 = 20",
			expected: []domain.InlineNode{&domain.Text{Value: "5 * 4 = 20"}},
		},
		{
			name:     "empty bold asterisks degrade to literal text",
			input:    "**",
			expected: []domain.InlineNode{&domain.Text{Value: "**"}},
		},
		{
			name:     "empty underscores degrade to literal text",
			input:    "__",
			expected: []domain.InlineNode{&domain.Text{Value: "__"}},
		},
		{
			name:     "empty tildes degrade to literal text",
			input:    "~~",
			expected: []domain.InlineNode{&domain.Text{Value: "~~"}},
		},
		{
			name:     "empty backticks degrade to literal text",
			input:    "``",
			expected: []domain.InlineNode{&domain.Text{Value: "``"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parser.ParseInline(tc.input)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Fatalf("ParseInline(%q) = %#v; want %#v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestInlineBackslashEscaping(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []domain.InlineNode
	}{
		{
			name:     "escaped asterisk",
			input:    `\*bold\*`,
			expected: []domain.InlineNode{&domain.Text{Value: "*bold*"}},
		},
		{
			name:     "escaped underscore",
			input:    `\_italic\_`,
			expected: []domain.InlineNode{&domain.Text{Value: "_italic_"}},
		},
		{
			name:     "escaped tilde",
			input:    `\~strike\~`,
			expected: []domain.InlineNode{&domain.Text{Value: "~strike~"}},
		},
		{
			name:     "escaped backtick",
			input:    "\\`code\\`",
			expected: []domain.InlineNode{&domain.Text{Value: "`code`"}},
		},
		{
			name:     "escaped brackets and parens",
			input:    `\[label\]\(target\)`,
			expected: []domain.InlineNode{&domain.Text{Value: "[label](target)"}},
		},
		{
			name:     "escaped backslash",
			input:    `\\`,
			expected: []domain.InlineNode{&domain.Text{Value: `\`}},
		},
		{
			name:     "escaped backslash followed by bold",
			input:    `\\*bold*`,
			expected: []domain.InlineNode{
				&domain.Text{Value: `\`},
				&domain.Strong{Children: []domain.InlineNode{&domain.Text{Value: "bold"}}},
			},
		},
		{
			name:     "non-delimiter backslash preserved",
			input:    `foo\bar`,
			expected: []domain.InlineNode{&domain.Text{Value: `foo\bar`}},
		},
		{
			name:     "trailing backslash preserved",
			input:    `foo\`,
			expected: []domain.InlineNode{&domain.Text{Value: `foo\`}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parser.ParseInline(tc.input)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Fatalf("ParseInline(%q) = %#v; want %#v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestInlineNesting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []domain.InlineNode
	}{
		{
			name:  "bold wrapping italic",
			input: "*_bold italic_*",
			expected: []domain.InlineNode{
				&domain.Strong{
					Children: []domain.InlineNode{
						&domain.Emphasis{
							Children: []domain.InlineNode{&domain.Text{Value: "bold italic"}},
						},
					},
				},
			},
		},
		{
			name:  "italic wrapping bold",
			input: "_*italic bold*_",
			expected: []domain.InlineNode{
				&domain.Emphasis{
					Children: []domain.InlineNode{
						&domain.Strong{
							Children: []domain.InlineNode{&domain.Text{Value: "italic bold"}},
						},
					},
				},
			},
		},
		{
			name:  "strike wrapping bold",
			input: "~*strike bold*~",
			expected: []domain.InlineNode{
				&domain.Strikethrough{
					Children: []domain.InlineNode{
						&domain.Strong{
							Children: []domain.InlineNode{&domain.Text{Value: "strike bold"}},
						},
					},
				},
			},
		},
		{
			name:  "bold wrapping link",
			input: "*[link](https://example.com)*",
			expected: []domain.InlineNode{
				&domain.Strong{
					Children: []domain.InlineNode{
						&domain.Link{
							Label:  []domain.InlineNode{&domain.Text{Value: "link"}},
							Target: "https://example.com",
						},
					},
				},
			},
		},
		{
			name:  "link with bold label",
			input: "[*bold link*](https://example.com)",
			expected: []domain.InlineNode{
				&domain.Link{
					Label: []domain.InlineNode{
						&domain.Strong{
							Children: []domain.InlineNode{&domain.Text{Value: "bold link"}},
						},
					},
					Target: "https://example.com",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parser.ParseInline(tc.input)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Fatalf("ParseInline(%q) = %#v; want %#v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestInlineLinkAndMedia(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []domain.InlineNode
	}{
		{
			name:  "standard web link",
			input: "[Documentation](https://vexil.org/docs)",
			expected: []domain.InlineNode{
				&domain.Link{
					Label:  []domain.InlineNode{&domain.Text{Value: "Documentation"}},
					Target: "https://vexil.org/docs",
				},
			},
		},
		{
			name:  "media with img prefix",
			input: "[Logo](img:https://example.com/logo.png)",
			expected: []domain.InlineNode{
				&domain.Media{
					Alt:    "Logo",
					Target: "img:https://example.com/logo.png",
				},
			},
		},
		{
			name:  "media by png extension",
			input: "[Diagram](architecture.png)",
			expected: []domain.InlineNode{
				&domain.Media{
					Alt:    "Diagram",
					Target: "architecture.png",
				},
			},
		},
		{
			name:  "media by jpg extension",
			input: "[Photo](photo.jpg)",
			expected: []domain.InlineNode{
				&domain.Media{
					Alt:    "Photo",
					Target: "photo.jpg",
				},
			},
		},
		{
			name:  "media by jpeg extension",
			input: "[Landscape](photo.jpeg)",
			expected: []domain.InlineNode{
				&domain.Media{
					Alt:    "Landscape",
					Target: "photo.jpeg",
				},
			},
		},
		{
			name:  "media by svg extension",
			input: "[Vector](icon.svg)",
			expected: []domain.InlineNode{
				&domain.Media{
					Alt:    "Vector",
					Target: "icon.svg",
				},
			},
		},
		{
			name:  "media by webp extension",
			input: "[Banner](banner.webp)",
			expected: []domain.InlineNode{
				&domain.Media{
					Alt:    "Banner",
					Target: "banner.webp",
				},
			},
		},
		{
			name:  "media by mp4 extension",
			input: "[Demo](screencast.mp4)",
			expected: []domain.InlineNode{
				&domain.Media{
					Alt:    "Demo",
					Target: "screencast.mp4",
				},
			},
		},
		{
			name:  "media by gif extension uppercase",
			input: "[Animation](reaction.GIF)",
			expected: []domain.InlineNode{
				&domain.Media{
					Alt:    "Animation",
					Target: "reaction.GIF",
				},
			},
		},
		{
			name:  "media with formatted alt text flattens to string",
			input: "[*Bold Logo*](logo.png)",
			expected: []domain.InlineNode{
				&domain.Media{
					Alt:    "Bold Logo",
					Target: "logo.png",
				},
			},
		},
		{
			name:     "degraded unclosed label brackets",
			input:    "[open bracket without target",
			expected: []domain.InlineNode{&domain.Text{Value: "[open bracket without target"}},
		},
		{
			name:     "degraded brackets without parens",
			input:    "[brackets only]",
			expected: []domain.InlineNode{&domain.Text{Value: "[brackets only]"}},
		},
		{
			name:     "degraded target without closing paren",
			input:    "[label](unclosed",
			expected: []domain.InlineNode{&domain.Text{Value: "[label](unclosed"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parser.ParseInline(tc.input)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Fatalf("ParseInline(%q) = %#v; want %#v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestInlineUnmatchedGracefulDegradation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []domain.InlineNode
	}{
		{
			name:     "unclosed bold",
			input:    "*unclosed bold",
			expected: []domain.InlineNode{&domain.Text{Value: "*unclosed bold"}},
		},
		{
			name:     "unclosed emphasis",
			input:    "_unclosed italic",
			expected: []domain.InlineNode{&domain.Text{Value: "_unclosed italic"}},
		},
		{
			name:     "unclosed strikethrough",
			input:    "~unclosed strike",
			expected: []domain.InlineNode{&domain.Text{Value: "~unclosed strike"}},
		},
		{
			name:     "unclosed inline code",
			input:    "`unclosed code",
			expected: []domain.InlineNode{&domain.Text{Value: "`unclosed code"}},
		},
		{
			name:  "interleaved delimiters unclosed inner degrades",
			input: "*bold _italic*",
			expected: []domain.InlineNode{
				&domain.Strong{
					Children: []domain.InlineNode{&domain.Text{Value: "bold _italic"}},
				},
			},
		},
		{
			name:  "valid bold followed by unclosed bold",
			input: "*bold* and *unclosed",
			expected: []domain.InlineNode{
				&domain.Strong{Children: []domain.InlineNode{&domain.Text{Value: "bold"}}},
				&domain.Text{Value: " and *unclosed"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parser.ParseInline(tc.input)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Fatalf("ParseInline(%q) = %#v; want %#v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestInlineCodeVerbatim(t *testing.T) {
	input := "`*not bold* and _not italic_ and [not link](url)`"
	expected := []domain.InlineNode{
		&domain.InlineCode{
			Value: "*not bold* and _not italic_ and [not link](url)",
		},
	}

	got := parser.ParseInline(input)
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("ParseInline(%q) = %#v; want %#v", input, got, expected)
	}
}

func TestSplitTableCells(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "basic row with outer pipes",
			input:    "| Col 1 | Col 2 | Col 3 |",
			expected: []string{"Col 1", "Col 2", "Col 3"},
		},
		{
			name:     "row without outer pipes",
			input:    "Col 1 | Col 2 | Col 3",
			expected: []string{"Col 1", "Col 2", "Col 3"},
		},
		{
			name:     "row with escaped pipes",
			input:    `| col1 | col2 \| extra | col3 |`,
			expected: []string{"col1", `col2 \| extra`, "col3"},
		},
		{
			name:     "row with pipe inside code span",
			input:    "| `a | b` | c |",
			expected: []string{"`a | b`", "c"},
		},
		{
			name:     "row with empty cells",
			input:    "| a | | c |",
			expected: []string{"a", "", "c"},
		},
		{
			name:     "empty row string",
			input:    "",
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parser.SplitTableCells(tc.input)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Fatalf("SplitTableCells(%q) = %#v; want %#v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestInlineParserInstance(t *testing.T) {
	p := parser.NewInlineParser()
	if p == nil {
		t.Fatal("NewInlineParser() returned nil")
	}

	input := "*bold* and _italic_"
	got := p.Parse(input)
	expected := parser.ParseInline(input)

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("p.Parse(%q) = %#v; want %#v", input, got, expected)
	}
}
