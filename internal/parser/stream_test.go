package parser_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"vexil/internal/domain"
	"vexil/internal/parser"
	"vexil/internal/ports"
)

// chunkedReader is an io.Reader that returns at most chunkSize bytes per Read call.
type chunkedReader struct {
	r         io.Reader
	chunkSize int
}

func (c *chunkedReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	limit := c.chunkSize
	if limit > len(p) {
		limit = len(p)
	}
	return c.r.Read(p[:limit])
}

// faultyReader simulates an I/O failure after reading a set number of bytes.
type faultyReader struct {
	r         io.Reader
	bytesLeft int
	errToEmit error
}

func (f *faultyReader) Read(p []byte) (int, error) {
	if f.bytesLeft <= 0 {
		return 0, f.errToEmit
	}
	limit := f.bytesLeft
	if limit > len(p) {
		limit = len(p)
	}
	n, err := f.r.Read(p[:limit])
	f.bytesLeft -= n
	if err != nil {
		return n, err
	}
	if f.bytesLeft <= 0 {
		return n, f.errToEmit
	}
	return n, nil
}

func sampleDocument() string {
	return `@title: Vexil Full Integration Document
@version: 1.0.0
@author: Vexil Team

# Section 1: Overview
This is a comprehensive document for testing *streaming* line-by-line parsing.
It includes consecutive paragraph lines that must merge properly.

## Sub-section 1.1: Lists
- Unordered item 1 with ` + "`inline code`" + `
  - Nested unordered item 1.1 with *bold* text
    - Deep nested 1.1.1
- Unordered item 2 with [link](https://vexil.org)

1. First ordered step
  2. Nested ordered sub-step
3. Third ordered step

[ ] Initial pending task
  [x] Completed subtask
[X] Another completed task

!warning: Security Notice
Do not expose private credentials.
Always sanitize incoming *user input*.

> This is a blockquote.
> It spans across multiple lines with _emphasis_.

::table [Service, Port, Protocol]
- Web Server | 8080 | HTTP/1.1
- DB Cluster \| Primary | 5432 | TCP
- Cache Node | ` + "`6379 | redis`" + ` | TCP
::

` + "```go server.go\npackage main\n\nfunc main() {}\n```" + `

---

Final concluding paragraph with ~strike~ text.
`
}

func TestStreamParserContract(t *testing.T) {
	var _ ports.StreamParser = parser.NewStreamParser()
	var _ ports.StreamParser = parser.NewParser()

	sp := parser.NewStreamParser()
	input := sampleDocument()

	nodeCh, errCh := sp.ParseStream(strings.NewReader(input))

	var nodes []domain.Node
	for node := range nodeCh {
		nodes = append(nodes, node)
	}

	for err := range errCh {
		if err != nil {
			t.Fatalf("unexpected stream error: %v", err)
		}
	}

	if len(nodes) == 0 {
		t.Fatal("expected nodes emitted from stream, got none")
	}
}

func TestStreamChunked5BytesIdenticalAST(t *testing.T) {
	docStr := sampleDocument()
	docBytes := []byte(docStr)

	// 1. Parse entire document from a single bytes.Buffer
	fullBuf := bytes.NewBuffer(docBytes)
	fullDoc, err := parser.Parse(fullBuf)
	if err != nil {
		t.Fatalf("full buffer Parse failed: %v", err)
	}

	// 2. Parse using chunkedReader feeding 5 bytes at a time
	chunked := &chunkedReader{
		r:         bytes.NewReader(docBytes),
		chunkSize: 5,
	}
	chunkedDoc, err := parser.Parse(chunked)
	if err != nil {
		t.Fatalf("5-byte chunked Parse failed: %v", err)
	}

	// 3. Verify Document Metadata is identical
	if !reflect.DeepEqual(fullDoc.Meta, chunkedDoc.Meta) {
		t.Fatalf("metadata mismatch:\nfull:    %#v\nchunked: %#v", fullDoc.Meta, chunkedDoc.Meta)
	}

	// 4. Verify AST Children count
	if len(fullDoc.Children) != len(chunkedDoc.Children) {
		t.Fatalf("children count mismatch: full=%d, chunked=%d", len(fullDoc.Children), len(chunkedDoc.Children))
	}

	// 5. Verify AST JSON serialization equivalence
	fullJSON, err := json.Marshal(fullDoc)
	if err != nil {
		t.Fatalf("full JSON marshal failed: %v", err)
	}
	chunkedJSON, err := json.Marshal(chunkedDoc)
	if err != nil {
		t.Fatalf("chunked JSON marshal failed: %v", err)
	}

	if !bytes.Equal(fullJSON, chunkedJSON) {
		t.Fatalf("JSON AST mismatch between full buffer and 5-byte chunked reader:\nfull:    %s\nchunked: %s",
			string(fullJSON), string(chunkedJSON))
	}

	// 6. Deep equal AST check
	if !reflect.DeepEqual(fullDoc, chunkedDoc) {
		t.Fatal("fullDoc and chunkedDoc ASTs are not deeply equal")
	}
}

func TestStreamParserEventsChunked5Bytes(t *testing.T) {
	docStr := sampleDocument()
	docBytes := []byte(docStr)

	// 1. Collect nodes from full stream
	fullNodeCh, fullErrCh := parser.ParseStream(bytes.NewBuffer(docBytes))
	var fullNodes []domain.Node
	for node := range fullNodeCh {
		fullNodes = append(fullNodes, node)
	}
	for err := range fullErrCh {
		if err != nil {
			t.Fatalf("full stream error: %v", err)
		}
	}

	// 2. Collect nodes from 5-byte chunked stream
	chunkedNodeCh, chunkedErrCh := parser.ParseStream(&chunkedReader{
		r:         bytes.NewReader(docBytes),
		chunkSize: 5,
	})
	var chunkedNodes []domain.Node
	for node := range chunkedNodeCh {
		chunkedNodes = append(chunkedNodes, node)
	}
	for err := range chunkedErrCh {
		if err != nil {
			t.Fatalf("chunked stream error: %v", err)
		}
	}

	// 3. Compare stream node counts
	if len(fullNodes) != len(chunkedNodes) {
		t.Fatalf("stream event count mismatch: full=%d, chunked=%d", len(fullNodes), len(chunkedNodes))
	}

	// 4. Compare each node in sequence
	for i := 0; i < len(fullNodes); i++ {
		fnJSON, _ := json.Marshal(fullNodes[i])
		cnJSON, _ := json.Marshal(chunkedNodes[i])
		if !bytes.Equal(fnJSON, cnJSON) {
			t.Fatalf("node %d mismatch in stream events:\nfull:    %s\nchunked: %s", i, string(fnJSON), string(cnJSON))
		}
	}
}

func TestStreamParserErrorPropagation(t *testing.T) {
	simulatedErr := errors.New("simulated network read failure")
	faulty := &faultyReader{
		r:         strings.NewReader(sampleDocument()),
		bytesLeft: 50,
		errToEmit: simulatedErr,
	}

	nodeCh, errCh := parser.ParseStream(faulty)

	for range nodeCh {
		// drain any initial nodes emitted before failure
	}

	var receivedErr error
	for err := range errCh {
		if err != nil {
			receivedErr = err
		}
	}

	if receivedErr == nil {
		t.Fatal("expected error from faulty stream, got nil")
	}
	if !errors.Is(receivedErr, simulatedErr) && !strings.Contains(receivedErr.Error(), "simulated") {
		t.Fatalf("expected simulated error, got: %v", receivedErr)
	}
}

func TestStreamParserEmpty(t *testing.T) {
	nodeCh, errCh := parser.ParseStream(strings.NewReader(""))

	var nodes []domain.Node
	for node := range nodeCh {
		nodes = append(nodes, node)
	}

	for err := range errCh {
		if err != nil {
			t.Fatalf("unexpected error on empty stream: %v", err)
		}
	}

	if len(nodes) != 0 {
		t.Fatalf("expected 0 nodes from empty stream, got %d", len(nodes))
	}
}
