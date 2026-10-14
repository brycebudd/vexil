# Vexil (.vxl) Language Specification

**Version:** 0.1.0  
**Status:** Canonical Reference Specification  
**Authors:** Vexil Working Group  
**Engine:** Pure Go Standard Library Implementation  

---

## 1. Abstract & Design Philosophy

Vexil (`.vxl`) is a deterministic, context-dense structured markup language designed specifically for large language model (LLM) context windows, retrieval-augmented generation (RAG) pipelines, and high-throughput streaming systems.

Traditional markup formats present significant friction when ingested by modern AI architectures:
1. **Markdown** suffers from ambiguous, non-deterministic edge cases (e.g., CommonMark's 17-rule delimiter pairing algorithms, multi-pass AST construction, and whitespace ambiguities).
2. **XML/HTML** consumes excessive token overhead on repetitive opening and closing tags.
3. **JSON/YAML** incurs syntax overhead on quotes, brackets, and commas, obscuring human readability.

Vexil resolves these challenges through four foundational invariants:
- **Single-Pass Streaming ($O(n)$)**: Vexil is tokenized and compiled in a single linear pass over any `io.Reader`. The parser requires zero backtracking, zero lookahead buffers, and zero regular expressions.
- **Strict Line-Prefix Grammar**: Every structural block is identified deterministically by its leading character prefix.
- **Deterministic Error Recovery**: Unclosed fences or mismatched inline delimiters never produce panics or fatal syntax rejections; they degrade gracefully into literal content.
- **First-Class Metadata & Native RAG Semantics**: Metadata directives (`@key: value`) attach cleanly to documents and individual blocks, while document structures naturally define semantic chunk boundaries with ancestor breadcrumb inheritance.

---

## 2. Lexical Structure & Block Scanners

### 2.1 Indentation & Whitespace Adjacency
- **Indentation Unit**: Document hierarchies (such as nested lists) increment strictly in units of **two ASCII spaces (`  `)**.
- **Tab Normalization**: Literal tab characters (`\t`) encountered in structural positions are normalized to two spaces (`  `).
- **Blank Lines**: Consecutive blank lines collapse into a single blank separator. A blank line deterministically terminates block bodies (callouts, blockquotes, lists, paragraphs).

### 2.2 Line-Prefix Taxonomy
A line's block type is resolved immediately upon inspecting its initial non-whitespace characters:

| Line Prefix | Block Node | Description |
| :--- | :--- | :--- |
| `@key: value` | Metadata Directive | Attaches key-value metadata to the document or immediate following block |
| `#{1,4}` | `Heading` | Document section titles from level 1 (`#`) to level 4 (`####`) |
| `---` | `Divider` | Horizontal rule / thematic break (exactly three hyphens) |
| ` ```[lang] [id]` | `CodeFence` | Verbatim multi-line code block with optional language and file identifier |
| `::table [H1 \| H2]` | `Table` | Tabular data block delimited by `::table` and closed by `::` |
| `!type[: Title]` | `Callout` | Semantic advisory block (`!note`, `!warning`, `!important`, `!prompt`) |
| `>` | `Blockquote` | Quotations and citations |
| `[ ]` or `[x]` / `[X]` | `TaskList` | Actionable task item with binary completion state |
| `- ` | `UnorderedList` | Bulleted list item |
| `1. ` or `. ` | `OrderedList` | Sequenced list item with explicit index or automatic counter (`. `) |
| *(None of above)* | `Paragraph` | Free-form prose paragraph; consecutive unmarked lines merge |

---

## 3. Block Directives & Semantics

### 3.1 Metadata Directives (`@key: value`)
Metadata lines take the format `@<key>: <value>`.

1. **Document-Level Metadata**:
   Any `@key: value` declarations placed at the beginning of a document and followed by a blank line are assigned to the document root (`Document.Meta`).
   ```vxl
   @title: System Architecture Spec
   @version: 1.0.0
   @author: Engineering Team

   # Overview
   ```

2. **Block-Level Metadata**:
   Any `@key: value` declarations immediately preceding a structural block (without intervening blank lines) attach to that block (`Node.Meta`):
   ```vxl
   @id: sec-core-pipeline
   ## Core Pipeline
   ```

### 3.2 Headings (`#{1,4}`)
Headings range from level 1 through 4:
```vxl
# Level 1 Heading
## Level 2 Heading
### Level 3 Heading
#### Level 4 Heading
```
Heading text is parsed for inline nodes. Headings automatically establish section boundaries for RAG chunk segmentation.

### 3.3 Code Fences (` ``` `)
Code fences begin with triple backticks ` ``` ` and terminate on a closing line beginning with ` ``` `.
```vxl
```go main.go
package main

func main() {}
```
```
- **Language**: The first token following the opening backticks specifies the syntax highlight language.
- **Identifier**: An optional second token (or colon-separated suffix `go:main.go`) identifies the file or snippet name.
- **Verbatim Body**: All whitespace, indentation, and interior characters are preserved verbatim. If EOF is reached before a closing fence, the parser auto-terminates the code block gracefully.

### 3.4 Tables (`::table ... ::`)
Tables define structured column headers and record rows:
```vxl
@id: tbl-metrics
@title: Performance Benchmarks
::table [Component | Throughput | Latency]
- Lexer | 480 MB/s | 1.2 ms
- Parser | 310 MB/s | 2.4 ms
- Formatter | 520 MB/s | 0.8 ms
::
```
- **Header Declaration**: Declared via bracket notation: `::table [Col1 | Col2]` or comma-delimited `::table [Col1, Col2]`.
- **Row Items**: Rows begin with a hyphen `- ` and separate cells with ` | `.
- **Escaped Pipes**: Literal pipes inside cells are escaped using `\|`. Pipes contained within inline code spans `` `a | b` `` are preserved without cell splitting.
- **Termination & Recovery**: Tables close with a dedicated `::` line. If an unclosed table reaches EOF or encounters an unindented block prefix, the parser flushes the table without error.

### 3.5 Callouts (`!type[: Title]`)
Callouts provide semantic framing for system messages, warnings, and prompts:
```vxl
!warning: Memory Threshold
High allocation volume detected in streaming reader buffer.

!prompt: Summarization Directive
Extract key architectural invariants from this specification.
```
- Standard types: `note`, `warning`, `important`, `prompt`, `tip`, `caution`.
- Body text continues across consecutive non-blank lines and terminates on the first empty line.

### 3.6 Blockquotes (`>`)
Consecutive lines prefixed with `>` merge into a single `Blockquote` node:
```vxl
> Architecture is the decisions that you wish you could get right
> early in a project.
```

### 3.7 Lists & Nested Structures
Indentation increments of two spaces (`  `) create hierarchical children:
```vxl
- Root item 1
  - Child item 1a
  - Child item 1b
- Root item 2

1. First ordered item
  . Sub-step with auto-numbering
  . Second sub-step
2. Second ordered item

[x] Initial spike complete
[ ] Finalize golden test suite
```

---

## 4. Inline Lexical Scanning & Delimiter Stack

### 4.1 Symmetric Delimiters
Inline styling is processed using a single-pass symmetric delimiter stack:

| Syntax | Node Type | Description |
| :--- | :--- | :--- |
| `*text*` | `Strong` | Bold text |
| `_text_` | `Emphasis` | Italic emphasis |
| `` `code` `` | `InlineCode` | Verbatim inline monospace code |
| `~text~` | `Strikethrough` | Strikethrough text |
| `[text](url)` | `Link` | Hyperlink |
| `[alt](img:url)` | `Media` | Image or media embed |

### 4.2 Whitespace Adjacency Rule
To prevent false-positive delimiter matches in mathematical expressions or prose (e.g., `3 * 4 * 5`):
1. **Opening Delimiter**: An opening delimiter (`*`, `_`, `~`, `` ` ``) **MUST NOT** be immediately followed by whitespace (`unicode.IsSpace`).
2. **Closing Delimiter**: A closing delimiter **MUST NOT** be immediately preceded by whitespace.

### 4.3 Backslash Escaping
Any delimiter character preceded by a backslash (`\`) is treated as a literal character:
```
Literal \*asterisks\* and \`backticks\` and \| pipes.
```
The escape character (`\`) is removed from the emitted AST value, preventing escape leakage.

### 4.4 Graceful Delimiter Degradation
If a delimiter opens but never encounters a valid closing delimiter before the end of the block, the parser unwinds the delimiter frame and emits the delimiter as literal text. Unmatched delimiters never invalidate the document.

### 4.5 Link vs. Media Differentiation
When the syntax `[label](target)` is scanned:
- If `target` starts with `img:`, or ends with media extensions (`.png`, `.jpg`, `.jpeg`, `.svg`, `.webp`, `.mp4`, `.gif`), it is emitted as `Media` (`<img src="..." alt="...">`).
- Otherwise, it is emitted as `Link` (`<a href="...">...</a>`).

---

## 5. RAG Semantic Chunking & Inheritance

Vexil documents partition natively into semantic retrieval chunks via `pkg/render.RAGChunker`:

1. **Chunk Boundaries**:
   - Every `Heading` initiates a new chunk.
   - Every `Table` forms an atomic semantic chunk.
   - Content prior to the first heading forms a `Preamble` chunk.
2. **Breadcrumb Inheritance**:
   - Each heading updates an internal ancestor heading stack.
   - Child chunks inherit the titles of all ancestor headings: `Breadcrumbs: ["Specification", "Syntax Overview", "Lists"]`.
3. **Metadata Propagation**:
   - Top-level document `@meta` is copied into every chunk.
   - Block-specific `@meta` is merged into the chunk, overriding duplicate keys.
4. **Token Estimation**:
   - Standard character-to-token ratio defaults to `4.0` characters per token.

---

## 6. Canonical AST Schema

The canonical AST emitted by `pkg/render.JSONRenderer` or `vxl ast` complies with the following JSON schema:

```json
{
  "type": "document",
  "meta": {
    "key": "value"
  },
  "children": [
    {
      "type": "heading",
      "level": 1,
      "meta": {},
      "children": [
        { "type": "text", "value": "Title Text" }
      ]
    },
    {
      "type": "paragraph",
      "meta": {},
      "children": [
        { "type": "text", "value": "Prose with " },
        { "type": "strong", "children": [{ "type": "text", "value": "bold" }] }
      ]
    },
    {
      "type": "callout",
      "callout_type": "note",
      "title": "Optional Title",
      "meta": {},
      "children": [
        { "type": "text", "value": "Body text" }
      ]
    },
    {
      "type": "code_fence",
      "lang": "go",
      "identifier": "main.go",
      "content": "package main\n",
      "meta": {}
    },
    {
      "type": "table",
      "headers": ["Col1", "Col2"],
      "rows": [
        [
          [{ "type": "text", "value": "Cell 1" }],
          [{ "type": "text", "value": "Cell 2" }]
        ]
      ],
      "meta": {}
    },
    {
      "type": "divider",
      "meta": {}
    }
  ]
}
```

---

## 7. Compliance & Reference Implementation

The normative reference engine for Vexil v0.1.0 is implemented in pure Go (standard library only) within this repository:
- Parser: `internal/parser`
- Format Engine: `pkg/formatter`
- Renderers & Chunker: `pkg/render`
- CLI Entrypoint: `cmd/vxl`
