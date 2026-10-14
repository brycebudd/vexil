# tree-sitter-vexil

Tree-sitter grammar and parser implementation for **Vexil** (`.vxl`), the deterministic, context-dense structured markup format engineered for LLM context windows, retrieval-augmented generation (RAG) pipelines, and high-throughput streaming systems.

This grammar conforms to the canonical [Vexil Language Specification v0.1.0](../docs/spec.md) and is compatible with Tree-sitter CLI v0.25+ / v0.27+.

---

## Features

- **Strict AST Alignment**: Full fidelity node emission matching Vexil v0.1.0 AST constructs:
  - Document-level and block-level metadata directives (`@key: value`)
  - Headings levels 1 through 4 (`#` to `####`)
  - Thematic dividers (`---`)
  - Fenced code blocks with language and file identifiers (` ```go main.go `)
  - Tables with typed headers and delimiter rows (`::table [...] ::`)
  - Advisory callouts (`!note`, `!warning`, `!important`, `!prompt`)
  - Blockquotes and citation lines (`> quote`)
  - Ordered lists (`1. `, `. `) and unordered lists (`- `)
  - Actionable task lists with completion status (`[ ]`, `[x]`, `[X]`)
  - Symmetric inline formatting: `*bold*`, `_emphasis_`, `` `code` ``, `~strikethrough~`, `[links](url)`, and `[media](img:url)`
- **Editor Queries**:
  - `queries/highlights.scm`: Fine-grained syntax highlight captures conformant with Neovim, Helix, and Zed standard capture namespaces.
  - `queries/injections.scm`: Dynamic nested language injections inside fenced code blocks (`go`, `rust`, `python`, `json`, `yaml`, `bash`, etc.).
  - `queries/folds.scm`: Structural code folding for code fences, tables, callouts, blockquotes, and list trees.
- **Polyglot Bindings**: Automated bindings generated for Node.js, Rust, Go, Python, C, and Swift.

---

## Installation

### Via npm (Node.js / WebAssembly)

```bash
npm install tree-sitter-vexil
```

### From Source (Git)

Clone the repository and install the development dependencies:

```bash
git clone https://github.com/brycebudd/vexil.git
cd vexil/tree-sitter-vexil
npm install
```

### Building the Parser

To regenerate the C parser and compile the native Node.js addon:

```bash
# Generate C parser from grammar.js
npx tree-sitter generate

# Compile native binding
npm run build
```

Requirements for native compilation:
- A working C/C++ compiler (`gcc`, `clang`, or MSVC)
- Node.js >= 18.0.0
- `node-gyp`

---

## Editor Integration

`tree-sitter-vexil` provides standard query files for effortless integration into modern modal and graphical editors.

### 1. Helix

Helix natively uses Tree-sitter for syntax highlighting, indentation, and code folding.

#### Configuration (`languages.toml`)

Add the following language and grammar definition to `~/.config/helix/languages.toml` (or `.helix/languages.toml` in your project root):

```toml
[[grammar]]
name = "vexil"
source = { git = "https://github.com/brycebudd/vexil", rev = "main", subpath = "tree-sitter-vexil" }
# Or for local development:
# source = { path = "/absolute/path/to/vexil/tree-sitter-vexil" }

[[language]]
name = "vexil"
scope = "source.vexil"
injection-regex = "^(vxl|vexil)$"
file-types = ["vxl", "vexil"]
roots = [".git"]
comment-token = "@"
indent = { tab-width = 2, unit = "  " }
grammar = "vexil"
```

#### Query Files Placement

Install the query files into the Helix runtime queries directory:

```bash
mkdir -p ~/.config/helix/runtime/queries/vexil

# Symlink or copy queries
ln -sfn /path/to/vexil/tree-sitter-vexil/queries/highlights.scm ~/.config/helix/runtime/queries/vexil/highlights.scm
ln -sfn /path/to/vexil/tree-sitter-vexil/queries/injections.scm ~/.config/helix/runtime/queries/vexil/injections.scm
ln -sfn /path/to/vexil/tree-sitter-vexil/queries/folds.scm ~/.config/helix/runtime/queries/vexil/folds.scm
```

#### Fetch & Compile

Fetch and build the grammar in Helix:

```bash
hx --grammar fetch
hx --grammar build
```

---

### 2. Neovim (`nvim-treesitter`)

Neovim supports custom Tree-sitter grammars through `nvim-treesitter`.

#### Configuration (`init.lua`)

Register the Vexil parser configuration and filetype detection in your Neovim configuration (e.g. `~/.config/nvim/init.lua` or `~/.config/nvim/lua/plugins/treesitter.lua`):

```lua
-- 1. Register filetype association for .vxl files
vim.filetype.add({
  extension = {
    vxl = "vexil",
    vexil = "vexil",
  },
})

-- 2. Register parser configuration in nvim-treesitter
local parser_config = require("nvim-treesitter.parsers").get_parser_configs()

parser_config.vexil = {
  install_info = {
    url = "https://github.com/brycebudd/vexil", -- or local path to repository (e.g. "~/repos/vexil")
    files = { "tree-sitter-vexil/src/parser.c" },
    branch = "main",
    generate_requires_npm = false,
    requires_generate_from_grammar = false,
  },
  filetype = "vexil",
}
```

#### Query Files Placement

Symlink the query files into Neovim's runtime query path:

```bash
mkdir -p ~/.config/nvim/queries/vexil

ln -sfn /path/to/vexil/tree-sitter-vexil/queries/highlights.scm ~/.config/nvim/queries/vexil/highlights.scm
ln -sfn /path/to/vexil/tree-sitter-vexil/queries/injections.scm ~/.config/nvim/queries/vexil/injections.scm
ln -sfn /path/to/vexil/tree-sitter-vexil/queries/folds.scm ~/.config/nvim/queries/vexil/folds.scm
```

#### Installation in Neovim

Open Neovim and run:

```vim
:TSInstall vexil
```

Syntax highlighting, folding, and language injections will activate automatically for all `.vxl` files.

---

### 3. Zed

Zed supports custom grammars through Zed language extensions.

#### Extension Manifest (`extension.toml`)

Create or update your Zed extension manifest with the Vexil grammar definition:

```toml
id = "vexil"
name = "Vexil"
version = "0.1.0"
schema_version = 1
authors = ["Bryce Budd <bbudd@agileparadigm.com>"]
description = "Vexil structured markup language support for Zed"
repository = "https://github.com/brycebudd/vexil"

[grammars.vexil]
repository = "https://github.com/brycebudd/vexil"
rev = "main"
path = "tree-sitter-vexil"
```

#### Language Configuration (`languages/vexil/config.toml`)

Define the language behavior in `languages/vexil/config.toml`:

```toml
name = "Vexil"
grammar = "vexil"
path_suffixes = ["vxl", "vexil"]
line_comments = ["# "]
tab_size = 2
hard_tabs = false
```

#### Query Placement

Copy query files into the extension directory:
```
languages/vexil/highlights.scm
languages/vexil/injections.scm
languages/vexil/folds.scm
```

---

### 4. Visual Studio Code

VS Code supports Tree-sitter grammars either through native Tree-sitter integrations or extensions.

#### Method A: Tree-sitter Extension (`vscode-tree-sitter`)

1. Install the `vscode-tree-sitter` extension.
2. In VS Code settings (`settings.json`), declare the language mapping:

```json
{
  "files.associations": {
    "*.vxl": "vexil"
  },
  "tree-sitter.grammars": {
    "vexil": {
      "path": "/path/to/vexil/tree-sitter-vexil",
      "highlights": "/path/to/vexil/tree-sitter-vexil/queries/highlights.scm"
    }
  }
}
```

#### Method B: Standalone VS Code Extension

To package `tree-sitter-vexil` as a first-class VS Code extension:
1. Compile the grammar to WebAssembly:
   ```bash
   npx tree-sitter build --wasm
   ```
2. In your extension's `package.json`, contribute the language:
   ```json
   {
     "contributes": {
       "languages": [
         {
           "id": "vexil",
           "aliases": ["Vexil", "vxl"],
           "extensions": [".vxl", ".vexil"],
           "configuration": "./language-configuration.json"
         }
       ]
     }
   }
   ```
3. Load the WebAssembly parser in your extension activation hook using `web-tree-sitter` and attach decorations using `queries/highlights.scm`.

---

## CLI Commands & Tooling

The Tree-sitter CLI enables testing, validation, and AST inspection.

### 1. Generating Parser Code

Regenerate the parser C code after modifying `grammar.js`:

```bash
npx tree-sitter generate
```

### 2. Running Corpus Tests

Execute the comprehensive corpus test suite (`test/corpus/`):

```bash
npx tree-sitter test
```

To run a specific test suite or filter by name:

```bash
npx tree-sitter test -f 'spec'
npx tree-sitter test -f 'blocks'
npx tree-sitter test -f 'inline'
```

### 3. Parsing Vexil Documents

Parse any Vexil file and display its concrete syntax tree (CST / AST):

```bash
# Parse canonical specification example
npx tree-sitter parse ../examples/spec.vxl

# Parse without byte offset ranges
npx tree-sitter parse --no-ranges ../examples/spec.vxl
```

### 4. Syntax Highlighting in Terminal

Render syntax-highlighted terminal output utilizing `queries/highlights.scm`:

```bash
npx tree-sitter highlight ../examples/spec.vxl
```

### 5. Compiling to WebAssembly

Compile the grammar to `.wasm` for browser environments or web-based IDEs:

```bash
npx tree-sitter build --wasm
```

---

## Corpus Test Suites

The test suite covers full language syntax across three modular suites:

1. **`test/corpus/blocks.txt`**: Unit tests for all block-level prefixes:
   - Metadata directives (`@key: value`, `@draft`)
   - Headings (levels 1 through 4)
   - Dividers (`---`)
   - Fenced code blocks (plain, tagged, colon identifiers)
   - Tables (pipe-separated and comma-separated)
   - Callouts with and without titles
   - Blockquotes (single and multi-line)
   - Lists (unordered `- `, ordered `1. `, automatic counters `. `, task states `[ ]` / `[x]`)
2. **`test/corpus/inline.txt`**: Unit tests for inline delimiters:
   - Bold (`*text*`), italic (`_text_`), inline code (`` `code` ``), strikethrough (`~text~`)
   - Hyperlinks (`[label](url)`) and media embeds (`[alt](img:url)`)
   - Backslash escape sequences
   - Nested formatting and inline nodes inside headings, lists, tables, callouts, and blockquotes
3. **`test/corpus/spec.txt`**: End-to-end integration tests representing real-world Vexil specifications:
   - Full canonical specification document (`examples/spec.vxl`)
   - Discrete canonical sections covering document headers, list hierarchies, advisory callouts, code fences, and tabular data

---

## References

- [Vexil Language Specification v0.1.0](../docs/spec.md)
- [Canonical Specification Example](../examples/spec.vxl)
- [Tree-sitter Documentation](https://tree-sitter.github.io/tree-sitter/)
