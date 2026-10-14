package parser

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"vexil/internal/domain"
)

// BlockScanner scans Vexil source documents into structural block-level AST nodes.
type BlockScanner struct {
	reader           *bufio.Reader
	peekLine         *string
	hasPeek          bool
	eof              bool
	docMeta          map[string]string
	pendingMeta      map[string]string
	hasEmittedBlocks bool
}

// NewBlockScanner creates a new block scanner reading from r.
func NewBlockScanner(r io.Reader) *BlockScanner {
	return &BlockScanner{
		reader:      bufio.NewReader(r),
		pendingMeta: make(map[string]string),
	}
}

// ParseBlocks parses a Vexil source stream into a domain.Document using BlockScanner.
func ParseBlocks(r io.Reader) (*domain.Document, error) {
	scanner := NewBlockScanner(r)
	doc, err := scanner.ScanDocument()
	if err != nil {
		return nil, fmt.Errorf("parse blocks: %w", err)
	}
	return doc, nil
}

// ParseBlocksString parses a Vexil source string into a domain.Document using BlockScanner.
func ParseBlocksString(s string) (*domain.Document, error) {
	return ParseBlocks(strings.NewReader(s))
}

// ScanDocument parses the entire input stream and returns the complete domain.Document AST.
func (s *BlockScanner) ScanDocument() (*domain.Document, error) {
	doc := &domain.Document{
		Children: make([]domain.Node, 0),
	}

	for {
		node, err := s.NextBlock()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("scan document: %w", err)
		}
		if node != nil {
			doc.Children = append(doc.Children, node)
		}
	}

	doc.Meta = s.DocumentMeta()
	return doc, nil
}

// DocumentMeta returns a copy of top-of-document metadata parsed so far.
func (s *BlockScanner) DocumentMeta() map[string]string {
	if len(s.docMeta) == 0 {
		return nil
	}
	res := make(map[string]string, len(s.docMeta))
	for k, v := range s.docMeta {
		res[k] = v
	}
	return res
}

// NextBlock parses and returns the next top-level block node, or io.EOF when done.
func (s *BlockScanner) NextBlock() (domain.Node, error) {
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
		return s.parseBlock()
	}
}

func (s *BlockScanner) parseBlock() (domain.Node, error) {
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
			Children: []domain.InlineNode{&domain.Text{Value: text}},
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
				parts := strings.Split(headerStr, ",")
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
				cellParts := strings.Split(rowContent, "|")
				var rowCells [][]domain.InlineNode
				for _, cell := range cellParts {
					cellVal := strings.TrimSpace(cell)
					rowCells = append(rowCells, []domain.InlineNode{&domain.Text{Value: cellVal}})
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
			inlines = append(inlines, &domain.Text{Value: strings.Join(bodyLines, "\n")})
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
			Children: []domain.InlineNode{&domain.Text{Value: strings.Join(lines, "\n")}},
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
				Children: []domain.InlineNode{&domain.Text{Value: taskText}},
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
					Children: []domain.InlineNode{&domain.Text{Value: tNext}},
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
				Children: []domain.InlineNode{&domain.Text{Value: itemText}},
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
					Children: []domain.InlineNode{&domain.Text{Value: parseUnordered(contentNext)}},
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
				Children: []domain.InlineNode{&domain.Text{Value: itemText}},
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
					Children: []domain.InlineNode{&domain.Text{Value: textNext}},
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
		Children: []domain.InlineNode{&domain.Text{Value: strings.Join(paraLines, "\n")}},
	}, nil
}

func (s *BlockScanner) consumeMeta() map[string]string {
	if len(s.pendingMeta) == 0 {
		return nil
	}
	res := make(map[string]string, len(s.pendingMeta))
	for k, v := range s.pendingMeta {
		res[k] = v
	}
	s.pendingMeta = make(map[string]string)
	return res
}

func (s *BlockScanner) readLine() (string, bool, error) {
	if s.hasPeek {
		line := *s.peekLine
		s.hasPeek = false
		s.peekLine = nil
		return line, false, nil
	}
	if s.eof {
		return "", true, nil
	}

	line, err := s.reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			s.eof = true
			if len(line) == 0 {
				return "", true, nil
			}
			line = strings.TrimRight(line, "\r\n")
			return line, false, nil
		}
		return "", false, fmt.Errorf("read line buffer: %w", err)
	}
	line = strings.TrimRight(line, "\r\n")
	return line, false, nil
}

func (s *BlockScanner) unreadLine(line string) {
	s.peekLine = &line
	s.hasPeek = true
}

func normalizeTabs(s string) string {
	return strings.ReplaceAll(s, "\t", "  ")
}

func countLeadingSpaces(s string) int {
	count := 0
	for count < len(s) && s[count] == ' ' {
		count++
	}
	return count
}

func countLeadingHashes(s string) int {
	count := 0
	for count < len(s) && s[count] == '#' {
		count++
	}
	return count
}

func isHeading(s string) bool {
	hCount := countLeadingHashes(s)
	if hCount >= 1 && hCount <= 4 {
		return hCount == len(s) || s[hCount] == ' ' || s[hCount] == '\t'
	}
	return false
}

func isUnorderedPrefix(s string) bool {
	return s == "-" || strings.HasPrefix(s, "- ")
}

func parseUnordered(s string) string {
	if s == "-" {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(s, "-"))
}

func isTaskPrefix(s string) bool {
	if len(s) >= 4 && s[0] == '[' && (s[1] == ' ' || s[1] == 'x' || s[1] == 'X') && s[2] == ']' {
		return s[3] == ' ' || s[3] == '\t'
	}
	return false
}

func parseTask(s string) (bool, string) {
	checked := s[1] == 'x' || s[1] == 'X'
	var text string
	if len(s) > 4 {
		text = strings.TrimSpace(s[4:])
	}
	return checked, text
}

func isOrderedPrefix(s string) (bool, string) {
	if s == "." {
		return true, ""
	}
	if strings.HasPrefix(s, ". ") {
		return true, strings.TrimSpace(s[2:])
	}
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i > 0 && i < len(s) && s[i] == '.' {
		if i+1 == len(s) {
			return true, ""
		}
		if s[i+1] == ' ' || s[i+1] == '\t' {
			return true, strings.TrimSpace(s[i+2:])
		}
	}
	return false, ""
}

func isBlockStart(s string) bool {
	if strings.HasPrefix(s, "@") && strings.Contains(s, ":") {
		return true
	}
	if isHeading(s) {
		return true
	}
	if s == "---" {
		return true
	}
	if strings.HasPrefix(s, "```") {
		return true
	}
	if strings.HasPrefix(s, "::table") {
		return true
	}
	if strings.HasPrefix(s, "!") {
		return true
	}
	if strings.HasPrefix(s, ">") {
		return true
	}
	if isTaskPrefix(s) {
		return true
	}
	if isUnorderedPrefix(s) {
		return true
	}
	if ok, _ := isOrderedPrefix(s); ok {
		return true
	}
	return false
}
