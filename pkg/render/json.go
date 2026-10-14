package render

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"vexil/internal/domain"
	"vexil/internal/ports"
)

var (
	// ErrNilDocument is returned when a nil Document is passed for rendering.
	ErrNilDocument = errors.New("document is nil")
	// ErrNilWriter is returned when a nil io.Writer is provided.
	ErrNilWriter = errors.New("writer is nil")
)

// Ensure JSONRenderer implements ports.Renderer.
var _ ports.Renderer = (*JSONRenderer)(nil)

// JSONOption configures a JSONRenderer instance.
type JSONOption func(*JSONRenderer)

// WithIndent configures the indentation string for JSON output.
// Pass an empty string "" for compact single-line JSON.
func WithIndent(indent string) JSONOption {
	return func(r *JSONRenderer) {
		r.indent = indent
	}
}

// JSONRenderer serializes a Vexil AST into canonical JSON.
type JSONRenderer struct {
	indent string
}

// NewJSONRenderer constructs a new JSONRenderer with optional configurations.
// By default, JSON is pretty-printed with two spaces indentation.
func NewJSONRenderer(opts ...JSONOption) *JSONRenderer {
	r := &JSONRenderer{
		indent: "  ",
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Render serializes the given Document AST to the provided io.Writer in JSON.
func (r *JSONRenderer) Render(doc *domain.Document, w io.Writer) error {
	if doc == nil {
		return fmt.Errorf("render json: %w", ErrNilDocument)
	}
	if w == nil {
		return fmt.Errorf("render json: %w", ErrNilWriter)
	}

	var data []byte
	var err error

	if r.indent != "" {
		data, err = json.MarshalIndent(doc, "", r.indent)
		if err != nil {
			return fmt.Errorf("marshal indented json: %w", err)
		}
		data = append(data, '\n')
	} else {
		data, err = json.Marshal(doc)
		if err != nil {
			return fmt.Errorf("marshal compact json: %w", err)
		}
	}

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("write json output: %w", err)
	}

	return nil
}

// RenderToString serializes the document into a JSON string.
func (r *JSONRenderer) RenderToString(doc *domain.Document) (string, error) {
	var buf bytes.Buffer
	if err := r.Render(doc, &buf); err != nil {
		return "", fmt.Errorf("render to string: %w", err)
	}
	return buf.String(), nil
}
