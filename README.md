# Vexil (`.vxl`)

> **Deterministic, context-dense structured document language engineered for LLMs, RAG pipelines, and single-pass streaming.**

[![Go Standard Library Only](https://img.shields.io/badge/dependencies-zero-brightgreen.svg)](go.mod)
[![Streaming $O(n)$](https://img.shields.io/badge/parser-streaming%20O(n)-blue.svg)](internal/parser)
[![Go Version](https://img.shields.io/badge/go-1.26+-00ADD8.svg)](go.mod)

---

## Overview

**Vexil** is a context-dense, deterministic document format designed to bridge human readability and high-efficiency AI processing. While formats like Markdown suffer from complex multi-pass grammar ambiguities, and JSON/XML suffer from high token overhead, Vexil provides:

- **Zero External Dependencies**: Implemented entirely within the Go standard library.
- **Zero Regular Expressions**: Lexical scanning and delimiter matching run in linear time with zero backtracking.
- **Single-Pass Streaming ($O(n)$)**: Documents parse line-by-line via `io.Reader` with constant memory overhead.
- **Native RAG Semantic Chunking**: Automatic segmentation at heading and table boundaries with ancestor breadcrumbs and inherited metadata.
- **First-Class Metadata**: Hierarchical `@key: value` directives attach to documents and block elements.
- **Multiple Down-Compilers**: Transpile to semantic HTML5, GitHub-Flavored Markdown (GFM), canonical JSON AST, and RAG retrieval chunks.
- **Canonical Auto-Formatting**: Enforce consistent indentation, table column alignment, and whitespace hygiene.

---

## Quick Start

### Installation & Build

Build the `vxl` command-line utility from source:

```bash
git clone https://github.com/example/vexil.git
cd vexil
go build -o vxl ./cmd/vxl
```

### CLI Commands

```bash
# Display help and available commands
./vxl --help

# Compile .vxl document to semantic HTML5
./vxl compile -target=html -o document.html document.vxl

# Transpile .vxl document to GitHub-Flavored Markdown
./vxl compile -target=md -o document.md document.vxl

# Emit canonical JSON AST
./vxl ast document.vxl

# Extract semantic RAG chunks as JSON
./vxl chunk document.vxl

# Auto-format document in-place
./vxl fmt -w document.vxl

# Pipe via stdin / stdout
cat document.vxl | ./vxl compile -target=html
```

---

## CLI Reference

### Subcommands

| Command | Usage | Description |
| :--- | :--- | :--- |
| `compile` | `vxl compile [flags] [file]` | Compile .vxl to target format (`html`, `md`, `json`, `chunks`) |
| `fmt` | `vxl fmt [flags] [file]` | Format .vxl document to canonical specification |
| `ast` | `vxl ast [-o out] [file]` | Shorthand for compiling to canonical JSON AST |
| `chunk` | `vxl chunk [-o out] [file]` | Shorthand for extracting RAG chunks as JSON |
| `version` | `vxl version` | Print engine version |
| `help` | `vxl help` | Print usage and help information |

### Flags

- `compile`:
  - `-target`: Output format: `html` (default), `md`, `json`, `chunks`.
  - `-o`: Output destination file path (defaults to `stdout`).
- `fmt`:
  - `-w`: Write formatted result directly to source file in-place.
  - `-align`: Align table columns to uniform widths.
- `ast` / `chunk`:
  - `-o`: Output destination file path (defaults to `stdout`).

---

## Go API Usage

Vexil is structured as clean Go packages ready for library integration:

### 1. Parsing into AST

```go
package main

import (
	"strings"
	"vexil/internal/parser"
)

func main() {
	input := `@title: Getting Started
# Hello Vexil
This is *bold* and _italic_ text.`

	doc, err := parser.NewParser().Parse(strings.NewReader(input))
	if err != nil {
		panic(err)
	}
	_ = doc
}
```

### 2. Streaming Line-by-Line

```go
package main

import (
	"fmt"
	"strings"
	"vexil/internal/parser"
)

func main() {
	r := strings.NewReader("# H1\n\nParagraph\n\n- List item\n")
	nodeCh, errCh := parser.NewParser().ParseStream(r)

	for node := range nodeCh {
		fmt.Printf("Streamed node: %s\n", node.Type())
	}
	if err := <-errCh; err != nil {
		panic(err)
	}
}
```

### 3. Rendering Targets

```go
package main

import (
	"os"
	"strings"
	"vexil/internal/parser"
	"vexil/pkg/render"
)

func main() {
	doc, _ := parser.NewParser().Parse(strings.NewReader("# Title\n\nBody prose."))

	// HTML5
	_ = render.NewHTMLRenderer().Render(doc, os.Stdout)

	// Markdown
	_ = render.NewMarkdownRenderer().Render(doc, os.Stdout)

	// JSON AST
	_ = render.NewJSONRenderer().Render(doc, os.Stdout)
}
```

### 4. Semantic RAG Chunking

```go
package main

import (
	"fmt"
	"strings"
	"vexil/internal/parser"
	"vexil/pkg/render"
)

func main() {
	src := `# Architecture
High-level overview.

## Storage Engine
Details on Badger and Bolt.`

	doc, _ := parser.NewParser().Parse(strings.NewReader(src))
	chunks, _ := render.NewRAGChunker().Chunk(doc)

	for _, c := range chunks {
		fmt.Printf("Chunk ID: %s | Title: %s | Breadcrumbs: %v | Tokens: %d\n",
			c.ID, c.Title, c.Breadcrumbs, c.TokenCount)
	}
}
```

### 5. Formatting

```go
package main

import (
	"os"
	"strings"
	"vexil/pkg/formatter"
)

func main() {
	raw := "   \n#   Unformatted Title   \n\n-  item\n"
	_ = formatter.Format(strings.NewReader(raw), os.Stdout)
}
```

---

## Syntax Overview

```vxl
@title: Architecture Overview
@version: 1.0.0

# System Architecture

Vexil combines *bold prose*, _italic emphasis_, `inline code`, and [links](https://vexil.org).

---

@id: sec-core
## Core Pipeline

- First bullet
  - Nested two-space sub-item
- Second bullet

1. Ordered step one
  . Auto-incremented sub-step
2. Ordered step two

[x] Task completed
[ ] Pending review

!note: Memory Boundary
Parser runs in strict O(n) streaming time without backtracking.

> Standard blockquote quote text.

```go
func main() {}
```

@id: tbl-rates
::table [Component | Throughput]
- Parser | 310 MB/s
- Formatter | 520 MB/s
::
```

For full language specifications, see [docs/spec.md](docs/spec.md).

---

## Package Architecture

```
vexil/
├── cmd/
│   └── vxl/               # CLI entrypoint (compile, fmt, ast, chunk)
├── internal/
│   ├── domain/            # AST node structs, token constants, and interfaces
│   ├── parser/            # Single-pass block scanner, delimiter stack, streaming parser
│   └── ports/             # Hexagonal port contracts (Parser, Renderer, Chunker, Formatter)
├── pkg/
│   ├── formatter/         # Canonical source code auto-formatter
│   └── render/            # Down-compilers (HTML5, Markdown, JSON, RAG Chunker)
├── examples/              # Full-feature example documents (spec.vxl)
├── testdata/              # Golden file verification suites
└── docs/                  # Formal language specification
```

---

## License

MIT License. See [LICENSE](LICENSE) for details.
