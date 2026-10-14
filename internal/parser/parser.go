package parser

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"vexil/internal/domain"
	"vexil/internal/ports"
)

// Ensure Parser implements ports.Parser and ports.StreamParser.
var (
	_ ports.Parser       = (*Parser)(nil)
	_ ports.StreamParser = (*Parser)(nil)
)

// Parser compiles Vexil source documents into fully resolved ASTs with inline nodes.
type Parser struct {
	inlineParser *InlineParser
}

// NewParser creates a new integrated Parser instance.
func NewParser() *Parser {
	return &Parser{
		inlineParser: NewInlineParser(),
	}
}

// Parse parses a Vexil source stream into a *domain.Document AST.
func (p *Parser) Parse(r io.Reader) (*domain.Document, error) {
	scanner := NewBlockScanner(r)
	doc := &domain.Document{
		Children: make([]domain.Node, 0),
	}

	for {
		node, err := p.nextBlock(scanner)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("parse document: %w", err)
		}
		if node != nil {
			doc.Children = append(doc.Children, node)
		}
	}

	doc.Meta = scanner.DocumentMeta()
	return doc, nil
}

// ParseString parses a Vexil source string into a *domain.Document AST.
func (p *Parser) ParseString(s string) (*domain.Document, error) {
	return p.Parse(strings.NewReader(s))
}

// ParseBytes parses a Vexil source byte slice into a *domain.Document AST.
func (p *Parser) ParseBytes(b []byte) (*domain.Document, error) {
	return p.Parse(bytes.NewReader(b))
}

// ParseStream streams top-level AST block nodes line-by-line via Go channels.
func (p *Parser) ParseStream(r io.Reader) (<-chan domain.Node, <-chan error) {
	nodeCh := make(chan domain.Node, 16)
	errCh := make(chan error, 1)

	go func() {
		defer close(nodeCh)
		defer close(errCh)

		scanner := NewBlockScanner(r)
		for {
			node, err := p.nextBlock(scanner)
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				errCh <- fmt.Errorf("stream block: %w", err)
				return
			}
			if node != nil {
				nodeCh <- node
			}
		}
	}()

	return nodeCh, errCh
}

// Parse parses a Vexil source stream into a *domain.Document AST using default Parser.
func Parse(r io.Reader) (*domain.Document, error) {
	return NewParser().Parse(r)
}

// ParseString parses a Vexil source string into a *domain.Document AST using default Parser.
func ParseString(s string) (*domain.Document, error) {
	return NewParser().ParseString(s)
}

// ParseBytes parses a Vexil source byte slice into a *domain.Document AST using default Parser.
func ParseBytes(b []byte) (*domain.Document, error) {
	return NewParser().ParseBytes(b)
}

// ParseStream streams top-level AST block nodes line-by-line using default StreamParser.
func ParseStream(r io.Reader) (<-chan domain.Node, <-chan error) {
	return NewStreamParser().ParseStream(r)
}

// nextBlock parses and returns the next top-level block with resolved inline AST nodes.
func (p *Parser) nextBlock(s *BlockScanner) (domain.Node, error) {
	for {
		line, eof, err := s.readLine()
		if err != nil {
			return nil, fmt.Errorf("read line: %w", err)
		}
		if eof {
			if !s.hasEmittedBlocks && len(s.pendingMeta) > 0 {
				if s.docMeta == nil {
					s.docMeta = make(map[string]string)
				}
				for k, v := range s.pendingMeta {
					s.docMeta[k] = v
				}
				s.pendingMeta = make(map[string]string)
			}
			return nil, io.EOF
		}

		// Blank line handling
		if strings.TrimSpace(line) == "" {
			if !s.hasEmittedBlocks && len(s.pendingMeta) > 0 {
				if s.docMeta == nil {
					s.docMeta = make(map[string]string)
				}
				for k, v := range s.pendingMeta {
					s.docMeta[k] = v
				}
				s.pendingMeta = make(map[string]string)
			}
			continue
		}

		// Check for @key: value metadata line
		norm := normalizeTabs(line)
		leadingSpaces := countLeadingSpaces(norm)
		content := norm[leadingSpaces:]

		if strings.HasPrefix(content, "@") && strings.Contains(content, ":") {
			colonIdx := strings.Index(content, ":")
			key := strings.TrimSpace(content[1:colonIdx])
			val := strings.TrimSpace(content[colonIdx+1:])
			s.pendingMeta[key] = val
			continue
		}

		// Content line starting a block: unread and parse
		s.unreadLine(line)
		return p.parseBlockWithInlines(s)
	}
}

func (p *Parser) parseBlockWithInlines(s *BlockScanner) (domain.Node, error) {
	line, eof, err := s.readLine()
	if err != nil {
		return nil, fmt.Errorf("read block start line: %w", err)
	}
	if eof {
		return nil, io.EOF
	}

	norm := normalizeTabs(line)
	leadingSpaces := countLeadingSpaces(norm)
	indent := leadingSpaces / 2
	content := norm[leadingSpaces:]

	// 1. Heading: #{1,4} Text
	if isHeading(content) {
		hashCount := countLeadingHashes(content)
		text := ""
		if hashCount < len(content) {
			text = strings.TrimSpace(content[hashCount:])
		}
		s.hasEmittedBlocks = true
		return &domain.Heading{
			Level:    hashCount,
			Meta:     s.consumeMeta(),
			Children: ParseInline(text),
		}, nil
	}

	// 2. Divider: --- (exact 3 hyphens)
	if content == "---" {
		s.hasEmittedBlocks = true
		return &domain.Divider{
			Meta: s.consumeMeta(),
		}, nil
	}

	// 3. CodeFence: ```lang identifier
	if strings.HasPrefix(content, "```") {
		meta := s.consumeMeta()
		info := strings.TrimSpace(content[3:])
		var lang, id string
		if colonIdx := strings.Index(info, ":"); colonIdx != -1 {
			lang = strings.TrimSpace(info[:colonIdx])
			id = strings.TrimSpace(info[colonIdx+1:])
		} else {
			fields := strings.Fields(info)
			if len(fields) > 0 {
				lang = fields[0]
			}
			if len(fields) > 1 {
				id = fields[1]
			}
		}

		var codeLines []string
		for {
			raw, codeEOF, codeErr := s.readLine()
			if codeErr != nil {
				return nil, fmt.Errorf("read code fence line: %w", codeErr)
			}
			if codeEOF {
				break // Graceful auto-close at EOF
			}
			if strings.HasPrefix(strings.TrimSpace(raw), "```") {
				break // Closing fence
			}
			codeLines = append(codeLines, raw)
		}

		s.hasEmittedBlocks = true
		return &domain.CodeFence{
			Lang:       lang,
			Identifier: id,
			Content:    strings.Join(codeLines, "\n"),
			Meta:       meta,
		}, nil
	}

	// 4. Table: ::table [Col1, Col2]
	if strings.HasPrefix(content, "::table") {
		meta := s.consumeMeta()
		remainder := strings.TrimSpace(strings.TrimPrefix(content, "::table"))
		var headers []string
		if openBracket := strings.Index(remainder, "["); openBracket != -1 {
			if closeBracket := strings.LastIndex(remainder, "]"); closeBracket > openBracket {
				headerStr := remainder[openBracket+1 : closeBracket]
				var parts []string
				if strings.Contains(headerStr, "|") {
					parts = SplitTableCells(headerStr)
				} else if strings.Contains(headerStr, ",") {
					parts = strings.Split(headerStr, ",")
				} else if strings.TrimSpace(headerStr) != "" {
					parts = []string{headerStr}
				}
				for _, p := range parts {
					pTrim := strings.TrimSpace(p)
					if len(pTrim) > 0 {
						headers = append(headers, pTrim)
					}
				}
			}
		}

		var rows [][][]domain.InlineNode
		for {
			raw, tblEOF, tblErr := s.readLine()
			if tblErr != nil {
				return nil, fmt.Errorf("read table line: %w", tblErr)
			}
			if tblEOF {
				break // Graceful EOF recovery on unclosed table fence
			}
			trimLine := strings.TrimSpace(raw)
			if trimLine == "::" {
				break // Closing table fence
			}
			if strings.HasPrefix(trimLine, "-") {
				rowContent := strings.TrimSpace(strings.TrimPrefix(trimLine, "-"))
				cellParts := SplitTableCells(rowContent)
				var rowCells [][]domain.InlineNode
				for _, cell := range cellParts {
					rowCells = append(rowCells, ParseInline(cell))
				}
				rows = append(rows, rowCells)
			}
		}

		s.hasEmittedBlocks = true
		return &domain.Table{
			Headers: headers,
			Rows:    rows,
			Meta:    meta,
		}, nil
	}

	// 5. Callout: !type: Optional Title
	if strings.HasPrefix(content, "!") {
		meta := s.consumeMeta()
		rest := content[1:]
		var calloutType, title string
		if colonIdx := strings.Index(rest, ":"); colonIdx != -1 {
			calloutType = strings.TrimSpace(rest[:colonIdx])
			title = strings.TrimSpace(rest[colonIdx+1:])
		} else {
			calloutType = strings.TrimSpace(rest)
		}

		var bodyLines []string
		for {
			raw, calloutEOF, calloutErr := s.readLine()
			if calloutErr != nil {
				return nil, fmt.Errorf("read callout line: %w", calloutErr)
			}
			if calloutEOF {
				break
			}
			if strings.TrimSpace(raw) == "" {
				break // Blank line terminates callout body
			}
			bodyLines = append(bodyLines, raw)
		}

		var inlines []domain.InlineNode
		if len(bodyLines) > 0 {
			inlines = ParseInline(strings.Join(bodyLines, "\n"))
		}

		s.hasEmittedBlocks = true
		return &domain.Callout{
			CalloutType: calloutType,
			Title:       title,
			Meta:        meta,
			Children:    inlines,
		}, nil
	}

	// 6. Blockquote: > Quoted text
	if strings.HasPrefix(content, ">") {
		meta := s.consumeMeta()
		firstText := strings.TrimPrefix(content, ">")
		firstText = strings.TrimPrefix(firstText, " ")
		lines := []string{firstText}

		for {
			raw, bqEOF, bqErr := s.readLine()
			if bqErr != nil {
				return nil, fmt.Errorf("read blockquote line: %w", bqErr)
			}
			if bqEOF {
				break
			}
			if strings.TrimSpace(raw) == "" {
				break // Blank line terminates blockquote
			}
			normNext := normalizeTabs(raw)
			lsNext := countLeadingSpaces(normNext)
			contentNext := normNext[lsNext:]
			if strings.HasPrefix(contentNext, ">") {
				nextText := strings.TrimPrefix(contentNext, ">")
				nextText = strings.TrimPrefix(nextText, " ")
				lines = append(lines, nextText)
			} else {
				s.unreadLine(raw)
				break
			}
		}

		s.hasEmittedBlocks = true
		return &domain.Blockquote{
			Meta:     meta,
			Children: ParseInline(strings.Join(lines, "\n")),
		}, nil
	}

	// 7. TaskList: [ ] or [x] or [X]
	if isTaskPrefix(content) {
		meta := s.consumeMeta()
		checked, taskText := parseTask(content)
		items := []*domain.TaskItem{
			{
				Checked:  checked,
				Level:    indent,
				Children: ParseInline(taskText),
			},
		}

		for {
			raw, taskEOF, taskErr := s.readLine()
			if taskErr != nil {
				return nil, fmt.Errorf("read task list line: %w", taskErr)
			}
			if taskEOF {
				break
			}
			if strings.TrimSpace(raw) == "" {
				break // Blank line terminates task list
			}
			normNext := normalizeTabs(raw)
			lsNext := countLeadingSpaces(normNext)
			contentNext := normNext[lsNext:]
			if isTaskPrefix(contentNext) {
				cNext, tNext := parseTask(contentNext)
				items = append(items, &domain.TaskItem{
					Checked:  cNext,
					Level:    lsNext / 2,
					Children: ParseInline(tNext),
				})
			} else {
				s.unreadLine(raw)
				break
			}
		}

		s.hasEmittedBlocks = true
		return &domain.TaskList{
			Items: items,
			Meta:  meta,
		}, nil
	}

	// 8. UnorderedList: - Item
	if isUnorderedPrefix(content) {
		meta := s.consumeMeta()
		itemText := parseUnordered(content)
		items := []*domain.ListItem{
			{
				Level:    indent,
				Children: ParseInline(itemText),
			},
		}

		for {
			raw, ulEOF, ulErr := s.readLine()
			if ulErr != nil {
				return nil, fmt.Errorf("read unordered list line: %w", ulErr)
			}
			if ulEOF {
				break
			}
			if strings.TrimSpace(raw) == "" {
				break // Blank line terminates list
			}
			normNext := normalizeTabs(raw)
			lsNext := countLeadingSpaces(normNext)
			contentNext := normNext[lsNext:]
			if isUnorderedPrefix(contentNext) {
				items = append(items, &domain.ListItem{
					Level:    lsNext / 2,
					Children: ParseInline(parseUnordered(contentNext)),
				})
			} else {
				s.unreadLine(raw)
				break
			}
		}

		s.hasEmittedBlocks = true
		return &domain.UnorderedList{
			Items: items,
			Meta:  meta,
		}, nil
	}

	// 9. OrderedList: 1. Item or . Item
	if ok, itemText := isOrderedPrefix(content); ok {
		meta := s.consumeMeta()
		items := []*domain.ListItem{
			{
				Level:    indent,
				Children: ParseInline(itemText),
			},
		}

		for {
			raw, olEOF, olErr := s.readLine()
			if olErr != nil {
				return nil, fmt.Errorf("read ordered list line: %w", olErr)
			}
			if olEOF {
				break
			}
			if strings.TrimSpace(raw) == "" {
				break // Blank line terminates list
			}
			normNext := normalizeTabs(raw)
			lsNext := countLeadingSpaces(normNext)
			contentNext := normNext[lsNext:]
			if okNext, textNext := isOrderedPrefix(contentNext); okNext {
				items = append(items, &domain.ListItem{
					Level:    lsNext / 2,
					Children: ParseInline(textNext),
				})
			} else {
				s.unreadLine(raw)
				break
			}
		}

		s.hasEmittedBlocks = true
		return &domain.OrderedList{
			Items: items,
			Meta:  meta,
		}, nil
	}

	// 10. Default: Paragraph (consecutive unmarked lines merge)
	meta := s.consumeMeta()
	paraLines := []string{line}

	for {
		raw, pEOF, pErr := s.readLine()
		if pErr != nil {
			return nil, fmt.Errorf("read paragraph line: %w", pErr)
		}
		if pEOF {
			break
		}
		if strings.TrimSpace(raw) == "" {
			break // Blank line terminates paragraph
		}
		normNext := normalizeTabs(raw)
		lsNext := countLeadingSpaces(normNext)
		contentNext := normNext[lsNext:]

		if isBlockStart(contentNext) {
			s.unreadLine(raw)
			break
		}

		paraLines = append(paraLines, raw)
	}

	s.hasEmittedBlocks = true
	return &domain.Paragraph{
		Meta:     meta,
		Children: ParseInline(strings.Join(paraLines, "\n")),
	}, nil
}
