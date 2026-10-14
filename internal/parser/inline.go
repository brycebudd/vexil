package parser

import (
	"strings"
	"unicode"

	"vexil/internal/domain"
)

type delimiterKind int

const (
	delimRoot delimiterKind = iota
	delimStrong        // *
	delimEmphasis      // _
	delimStrikethrough // ~
	delimLinkOpen      // [
)

type frame struct {
	kind     delimiterKind
	pos      int
	raw      string
	children []domain.InlineNode
}

// InlineParser provides single-pass inline lexical scanning with a symmetric delimiter stack.
type InlineParser struct{}

// NewInlineParser constructs a reusable InlineParser instance.
func NewInlineParser() *InlineParser {
	return &InlineParser{}
}

// ParseInline parses raw inline markdown prose into an AST slice of domain.InlineNode.
func ParseInline(input string) []domain.InlineNode {
	return NewInlineParser().Parse(input)
}

// Parse parses inline text into a slice of InlineNode using a symmetric delimiter stack.
func (p *InlineParser) Parse(input string) []domain.InlineNode {
	if len(input) == 0 {
		return nil
	}

	runes := []rune(input)
	stack := []frame{
		{kind: delimRoot},
	}

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		// 1. Backslash escaping
		if r == '\\' {
			if i+1 < len(runes) && isEscapableDelimiter(runes[i+1]) {
				appendNode(&stack[len(stack)-1].children, &domain.Text{Value: string(runes[i+1])})
				i++
				continue
			}
			appendNode(&stack[len(stack)-1].children, &domain.Text{Value: "\\"})
			continue
		}

		// 2. InlineCode: `code`
		if r == '`' {
			// Opening delimiter must NOT be followed by whitespace
			if i+1 < len(runes) && !unicode.IsSpace(runes[i+1]) {
				closingIdx := -1
				for j := i + 2; j < len(runes); j++ {
					if runes[j] == '`' && !unicode.IsSpace(runes[j-1]) {
						closingIdx = j
						break
					}
				}
				if closingIdx != -1 {
					codeVal := string(runes[i+1 : closingIdx])
					appendNode(&stack[len(stack)-1].children, &domain.InlineCode{Value: codeVal})
					i = closingIdx
					continue
				}
			}
			// Unmatched backtick degrades to literal text
			appendNode(&stack[len(stack)-1].children, &domain.Text{Value: "`"})
			continue
		}

		// 3. Link open: [
		if r == '[' {
			stack = append(stack, frame{
				kind: delimLinkOpen,
				pos:  i,
				raw:  "[",
			})
			continue
		}

		// 4. Link close candidate: ]
		if r == ']' {
			linkIdx := -1
			for s := len(stack) - 1; s >= 1; s-- {
				if stack[s].kind == delimLinkOpen {
					linkIdx = s
					break
				}
			}

			if linkIdx != -1 && i+1 < len(runes) && runes[i+1] == '(' {
				targetEnd := -1
				for j := i + 2; j < len(runes); j++ {
					if runes[j] == '\\' && j+1 < len(runes) {
						j++
						continue
					}
					if runes[j] == ')' {
						targetEnd = j
						break
					}
				}

				if targetEnd != -1 {
					rawTarget := string(runes[i+2 : targetEnd])
					target := strings.TrimSpace(unescapeTarget(rawTarget))

					// Unwind any unclosed delimiters inside this link label
					for k := len(stack) - 1; k > linkIdx; k-- {
						unwindFrame(stack[k], &stack[k-1])
					}
					stack = stack[:linkIdx+1]

					labelNodes := stack[linkIdx].children
					stack = stack[:linkIdx] // pop delimLinkOpen

					if isMediaTarget(target) {
						alt := extractPlainText(labelNodes)
						appendNode(&stack[len(stack)-1].children, &domain.Media{
							Alt:    alt,
							Target: target,
						})
					} else {
						appendNode(&stack[len(stack)-1].children, &domain.Link{
							Label:  labelNodes,
							Target: target,
						})
					}

					i = targetEnd
					continue
				}
			}

			// Not a valid link: treat ']' as literal text
			appendNode(&stack[len(stack)-1].children, &domain.Text{Value: "]"})
			continue
		}

		// 5. Symmetric delimiters: *, _, ~
		var kind delimiterKind
		switch r {
		case '*':
			kind = delimStrong
		case '_':
			kind = delimEmphasis
		case '~':
			kind = delimStrikethrough
		default:
			appendNode(&stack[len(stack)-1].children, &domain.Text{Value: string(r)})
			continue
		}

		// Closing delimiter must NOT be preceded by whitespace.
		canClose := false
		matchIdx := -1
		if i > 0 && !unicode.IsSpace(runes[i-1]) {
			for s := len(stack) - 1; s >= 1; s-- {
				if stack[s].kind == kind && stack[s].pos != i-1 {
					canClose = true
					matchIdx = s
					break
				}
			}
		}

		// Opening delimiter must NOT be followed by whitespace.
		canOpen := false
		if i+1 < len(runes) && !unicode.IsSpace(runes[i+1]) {
			canOpen = true
		}

		if canClose {
			for k := len(stack) - 1; k > matchIdx; k-- {
				unwindFrame(stack[k], &stack[k-1])
			}
			stack = stack[:matchIdx+1]

			top := stack[matchIdx]
			stack = stack[:matchIdx]

			var closedNode domain.InlineNode
			switch top.kind {
			case delimStrong:
				closedNode = &domain.Strong{Children: top.children}
			case delimEmphasis:
				closedNode = &domain.Emphasis{Children: top.children}
			case delimStrikethrough:
				closedNode = &domain.Strikethrough{Children: top.children}
			}
			appendNode(&stack[len(stack)-1].children, closedNode)
			continue
		}

		if canOpen {
			stack = append(stack, frame{
				kind: kind,
				pos:  i,
				raw:  string(r),
			})
			continue
		}

		// Delimiter neither closes nor opens: literal text
		appendNode(&stack[len(stack)-1].children, &domain.Text{Value: string(r)})
	}

	// End of input: unwind all unclosed frames into root
	for k := len(stack) - 1; k >= 1; k-- {
		unwindFrame(stack[k], &stack[k-1])
	}

	return stack[0].children
}

// SplitTableCells splits a row by '|' while preserving escaped pipes and pipes inside inline code.
func SplitTableCells(row string) []string {
	if strings.TrimSpace(row) == "" {
		return nil
	}

	runes := []rune(row)
	var cells []string
	var current strings.Builder
	inCode := false

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		// Escaped pipe or character
		if r == '\\' && i+1 < len(runes) {
			current.WriteRune(r)
			current.WriteRune(runes[i+1])
			i++
			continue
		}

		// Toggle code span on backtick
		if r == '`' {
			inCode = !inCode
			current.WriteRune(r)
			continue
		}

		// Delimiter pipe outside code
		if r == '|' && !inCode {
			cells = append(cells, current.String())
			current.Reset()
			continue
		}

		current.WriteRune(r)
	}

	cells = append(cells, current.String())

	// Strip leading/trailing empty cells from border pipes
	if len(cells) > 1 && strings.TrimSpace(cells[0]) == "" && strings.HasPrefix(strings.TrimSpace(row), "|") {
		cells = cells[1:]
	}
	if len(cells) > 0 && strings.TrimSpace(cells[len(cells)-1]) == "" && strings.HasSuffix(strings.TrimSpace(row), "|") {
		cells = cells[:len(cells)-1]
	}

	if len(cells) == 0 {
		return nil
	}

	res := make([]string, len(cells))
	for i, c := range cells {
		res[i] = strings.TrimSpace(c)
	}
	return res
}

func isEscapableDelimiter(r rune) bool {
	switch r {
	case '*', '_', '~', '`', '[', ']', '(', ')', '\\', '|':
		return true
	default:
		return false
	}
}

func unescapeTarget(target string) string {
	runes := []rune(target)
	var sb strings.Builder
	for i := 0; i < len(runes); i++ {
		if runes[i] == '\\' && i+1 < len(runes) && (runes[i+1] == ')' || runes[i+1] == '\\') {
			sb.WriteRune(runes[i+1])
			i++
			continue
		}
		sb.WriteRune(runes[i])
	}
	return sb.String()
}

func isMediaTarget(target string) bool {
	if strings.HasPrefix(target, "img:") {
		return true
	}

	mediaExts := []string{
		".png",
		".jpg",
		".jpeg",
		".svg",
		".webp",
		".mp4",
		".gif",
	}

	lower := strings.ToLower(target)
	for _, ext := range mediaExts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}

	// Also check target URL path without query string or hash
	clean := target
	if idx := strings.IndexAny(clean, "?#"); idx != -1 {
		clean = clean[:idx]
	}
	lowerClean := strings.ToLower(clean)
	for _, ext := range mediaExts {
		if strings.HasSuffix(lowerClean, ext) {
			return true
		}
	}

	return false
}

func extractPlainText(nodes []domain.InlineNode) string {
	var sb strings.Builder
	for _, node := range nodes {
		switch n := node.(type) {
		case *domain.Text:
			sb.WriteString(n.Value)
		case *domain.Strong:
			sb.WriteString(extractPlainText(n.Children))
		case *domain.Emphasis:
			sb.WriteString(extractPlainText(n.Children))
		case *domain.InlineCode:
			sb.WriteString(n.Value)
		case *domain.Strikethrough:
			sb.WriteString(extractPlainText(n.Children))
		case *domain.Link:
			sb.WriteString(extractPlainText(n.Label))
		case *domain.Media:
			sb.WriteString(n.Alt)
		}
	}
	return sb.String()
}

func appendNode(children *[]domain.InlineNode, node domain.InlineNode) {
	if node == nil {
		return
	}
	if text, ok := node.(*domain.Text); ok {
		if text.Value == "" {
			return
		}
		if len(*children) > 0 {
			if lastText, ok := (*children)[len(*children)-1].(*domain.Text); ok {
				lastText.Value += text.Value
				return
			}
		}
	}
	*children = append(*children, node)
}

func unwindFrame(from frame, into *frame) {
	if from.raw != "" {
		appendNode(&into.children, &domain.Text{Value: from.raw})
	}
	for _, child := range from.children {
		appendNode(&into.children, child)
	}
}
