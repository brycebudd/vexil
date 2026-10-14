package parser

import (
	"io"

	"vexil/internal/domain"
	"vexil/internal/ports"
)

// Ensure StreamParser implements ports.StreamParser.
var _ ports.StreamParser = (*StreamParser)(nil)

// StreamParser provides single-pass streaming line-by-line block emission for Vexil streams.
type StreamParser struct {
	parser *Parser
}

// NewStreamParser creates a new streaming block parser.
func NewStreamParser() *StreamParser {
	return &StreamParser{
		parser: NewParser(),
	}
}

// ParseStream emits domain.Node blocks as they are parsed from r, sending any error to errCh.
func (sp *StreamParser) ParseStream(r io.Reader) (<-chan domain.Node, <-chan error) {
	return sp.parser.ParseStream(r)
}
