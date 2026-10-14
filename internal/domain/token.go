package domain

import "fmt"

// TokenType represents lexical token categories in Vexil documents.
type TokenType string

const (
	// Special tokens
	TokenEOF     TokenType = "EOF"
	TokenError   TokenType = "ERROR"
	TokenNewline TokenType = "NEWLINE"

	// Block-level line prefix tokens
	TokenMeta           TokenType = "META"             // @key: value
	TokenHeading        TokenType = "HEADING"          // #, ##, ###, ####
	TokenBullet         TokenType = "BULLET"           // -
	TokenOrdered        TokenType = "ORDERED"          // 1., 2. or .
	TokenTask           TokenType = "TASK"             // [ ] or [x]
	TokenCallout        TokenType = "CALLOUT"          // !warning, !note, etc.
	TokenBlockquote     TokenType = "BLOCKQUOTE"       // >
	TokenCodeFenceOpen  TokenType = "CODE_FENCE_OPEN"  // ```lang:id
	TokenCodeFenceClose TokenType = "CODE_FENCE_CLOSE" // ```
	TokenTableOpen      TokenType = "TABLE_OPEN"       // ::table
	TokenTableClose     TokenType = "TABLE_CLOSE"      // ::
	TokenTableRow       TokenType = "TABLE_ROW"        // | col1 | col2 |
	TokenDivider        TokenType = "DIVIDER"          // ---
	TokenParagraph      TokenType = "PARAGRAPH"        // Raw paragraph line

	// Inline delimiter tokens
	TokenText          TokenType = "TEXT"          // Plain text
	TokenStrong        TokenType = "STRONG"        // *
	TokenEmphasis      TokenType = "EMPHASIS"      // _
	TokenInlineCode    TokenType = "INLINE_CODE"   // `
	TokenStrikethrough TokenType = "STRIKETHROUGH" // ~
	TokenLinkOpen      TokenType = "LINK_OPEN"     // [
	TokenLinkClose     TokenType = "LINK_CLOSE"    // ]
	TokenTargetOpen    TokenType = "TARGET_OPEN"   // (
	TokenTargetClose   TokenType = "TARGET_CLOSE"  // )
	TokenPipe          TokenType = "PIPE"          // |
	TokenEscape        TokenType = "ESCAPE"        // \
)

// Token represents a scanned lexical unit in a Vexil source document.
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

// String returns a human-readable representation of a token.
func (t Token) String() string {
	return fmt.Sprintf("Token(%s, %q, %d:%d)", t.Type, t.Literal, t.Line, t.Column)
}

// NewToken constructs a Token with the given properties.
func NewToken(tokenType TokenType, literal string, line, column int) Token {
	return Token{
		Type:    tokenType,
		Literal: literal,
		Line:    line,
		Column:  column,
	}
}
