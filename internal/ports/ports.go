package ports

import (
	"io"

	"vexil/internal/domain"
)

// Chunk represents a semantic segment of a Vexil document partitioned for RAG retrieval.
type Chunk struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Breadcrumbs []string          `json:"breadcrumbs"`
	Content     string            `json:"content"`
	Meta        map[string]string `json:"meta,omitempty"`
	CharCount   int               `json:"char_count"`
	TokenCount  int               `json:"token_count"`
}

// Parser defines the contract for compiling full Vexil streams into an in-memory AST.
type Parser interface {
	Parse(r io.Reader) (*domain.Document, error)
}

// StreamParser defines the contract for single-pass streaming line-by-line block emission.
type StreamParser interface {
	ParseStream(r io.Reader) (<-chan domain.Node, <-chan error)
}

// Renderer defines the contract for serializing a Vexil Document into a target format.
type Renderer interface {
	Render(doc *domain.Document, w io.Writer) error
}

// Chunker defines the contract for segmenting a Vexil Document into semantic RAG chunks.
type Chunker interface {
	Chunk(doc *domain.Document) ([]Chunk, error)
}

// Formatter defines the contract for canonically formatting Vexil source documents.
type Formatter interface {
	Format(r io.Reader, w io.Writer) error
}
