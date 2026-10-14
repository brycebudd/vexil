; ==============================================================================
; Vexil Tree-sitter Highlights Query
; Conforms to standard capture groups for Neovim, Helix, Zed, and GitHub
; ==============================================================================

; ------------------------------------------------------------------------------
; Structural Blocks
; ------------------------------------------------------------------------------

; Headings
(heading) @markup.heading

(heading_marker) @markup.heading.marker

(heading
  level: (heading_marker) @_h1
  (#eq? @_h1 "#")) @markup.heading.1

(heading
  level: (heading_marker) @_h2
  (#eq? @_h2 "##")) @markup.heading.2

(heading
  level: (heading_marker) @_h3
  (#eq? @_h3 "###")) @markup.heading.3

(heading
  level: (heading_marker) @_h4
  (#eq? @_h4 "####")) @markup.heading.4

; Metadata directives
(meta_directive
  "@" @punctuation.delimiter
  key: (identifier) @keyword.directive)

(meta_directive
  ":" @punctuation.delimiter)

(meta_directive
  value: (meta_value) @string)

; Dividers
(divider) @punctuation.delimiter

; Blockquotes
(blockquote) @markup.quote

(blockquote
  ">" @markup.quote)

; Callouts
(callout
  "!" @punctuation.delimiter
  type: (callout_type) @keyword)

(callout
  ":" @punctuation.delimiter)

(callout
  title: (callout_title) @markup.heading)

; Code fences
(code_fence) @markup.raw.block
(code_fence_content) @markup.raw.block

(code_fence
  "```" @punctuation.delimiter)

(code_fence
  language: (identifier) @constant.character)

(code_fence
  ":" @punctuation.delimiter)

(code_fence
  identifier: (identifier) @string.special.path)

; Tables
(table
  "::" @punctuation.delimiter)

(table_header
  "::table" @punctuation.delimiter)

(table_header
  "[" @punctuation.bracket)

(table_header
  "]" @punctuation.bracket)

(table_header
  "|" @punctuation.delimiter)

(table_header
  "," @punctuation.delimiter)

(table_header
  (table_cell) @markup.heading)

(table_row
  "-" @punctuation.delimiter)

(table_row
  "|" @punctuation.delimiter)

; Lists
(unordered_list_item
  "-" @markup.list)

(ordered_list_item) @markup.list

(ordered_list_item
  "." @markup.list)

(task_list_item
  status: (task_status) @markup.list.unchecked
  (#eq? @markup.list.unchecked "[ ]"))

(task_list_item
  status: (task_status) @markup.list.checked
  (#match? @markup.list.checked "^\\[[xX]\\]$"))

; ------------------------------------------------------------------------------
; Inline Formatting
; ------------------------------------------------------------------------------

(strong) @markup.strong
(strong
  "*" @punctuation.delimiter)

(emphasis) @markup.italic
(emphasis
  "_" @punctuation.delimiter)

(inline_code) @markup.raw.inline
(inline_code
  "`" @punctuation.delimiter)

(strikethrough) @markup.strikethrough
(strikethrough
  "~" @punctuation.delimiter)

(escape_sequence) @string.escape

(link
  "[" @punctuation.bracket)
(link
  label: (link_label) @markup.link.label)
(link
  "]" @punctuation.bracket)
(link
  "(" @punctuation.bracket)
(link
  destination: (link_destination) @markup.link.url)
(link
  ")" @punctuation.bracket)

(media
  "[" @punctuation.bracket)
(media
  alt: (media_alt) @markup.link.label)
(media
  "]" @punctuation.bracket)
(media
  "(" @punctuation.bracket)
(media
  source: (media_source) @markup.link.url)
(media
  ")" @punctuation.bracket)
