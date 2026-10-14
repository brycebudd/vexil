; ==============================================================================
; Vexil Tree-sitter Injections Query
; Injects embedded language highlighters into code fence contents
; ==============================================================================

(code_fence
  language: (identifier) @injection.language
  content: (code_fence_content) @injection.content)
