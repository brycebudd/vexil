package formatter

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"vexil/internal/parser"
	"vexil/internal/ports"
)

// Ensure Formatter implements ports.Formatter at compile time.
var _ ports.Formatter = (*Formatter)(nil)

// Option defines a functional option for configuring a Formatter instance.
type Option func(*Formatter)

// WithAlignTableColumns configures whether table columns should be padded to aligned widths.
func WithAlignTableColumns(align bool) Option {
	return func(f *Formatter) {
		f.alignTableColumns = align
	}
}

// Formatter formats Vexil source documents according to canonical rules.
type Formatter struct {
	alignTableColumns bool
}

// NewFormatter constructs a new Formatter with the provided functional options.
func NewFormatter(opts ...Option) *Formatter {
	f := &Formatter{}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// Format formats a Vexil source stream from r and writes canonical output to w.
func (f *Formatter) Format(r io.Reader, w io.Writer) error {
	reader := bufio.NewReader(r)

	var (
		inCodeFence       bool
		codeFenceIndent   string
		inTable           bool
		tableIndent       string
		tableHeader       string
		tableRows         [][]string
		hasEmittedContent bool
		pendingBlankLine  bool
	)

	for {
		rawLine, readErr := reader.ReadString('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return fmt.Errorf("formatter: read line: %w", readErr)
		}

		line := strings.TrimSuffix(rawLine, "\n")
		line = strings.TrimSuffix(line, "\r")

		isEOF := errors.Is(readErr, io.EOF)

		// 1. Inside code fence: preserve verbatim
		if inCodeFence {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "```") {
				// Closing fence
				if err := f.writeLine(w, codeFenceIndent+"```"); err != nil {
					return err
				}
				inCodeFence = false
			} else {
				// Body line: preserved verbatim
				if err := f.writeLine(w, line); err != nil {
					return err
				}
			}
			if isEOF {
				break
			}
			continue
		}

		// 2. Inside table: buffer rows until :: or EOF or another block
		if inTable {
			trimmed := strings.TrimSpace(line)
			if trimmed == "::" {
				if err := f.flushTable(w, tableIndent, tableHeader, tableRows); err != nil {
					return err
				}
				inTable = false
				tableRows = nil
				tableHeader = ""
				if isEOF {
					break
				}
				continue
			}

			if strings.HasPrefix(trimmed, "-") {
				rowContent := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
				cells := parser.SplitTableCells(rowContent)
				tableRows = append(tableRows, cells)
				if isEOF {
					if err := f.flushTable(w, tableIndent, tableHeader, tableRows); err != nil {
						return err
					}
					inTable = false
					break
				}
				continue
			}

			if trimmed == "" {
				// Blank line inside table
				if isEOF {
					if err := f.flushTable(w, tableIndent, tableHeader, tableRows); err != nil {
						return err
					}
					inTable = false
					break
				}
				continue
			}

			// If unexpected block start inside table, recover gracefully by flushing table
			normLine := normalizeTabs(line)
			ls := countLeadingSpaces(normLine)
			c := normLine[ls:]
			if isBlockStart(c) {
				if err := f.flushTable(w, tableIndent, tableHeader, tableRows); err != nil {
					return err
				}
				inTable = false
				tableRows = nil
				tableHeader = ""
				// Continue processing this line outside table below
			} else {
				// Fallback cell row without dash
				cells := parser.SplitTableCells(trimmed)
				tableRows = append(tableRows, cells)
				if isEOF {
					if err := f.flushTable(w, tableIndent, tableHeader, tableRows); err != nil {
						return err
					}
					inTable = false
					break
				}
				continue
			}
		}

		// 3. Normal block line processing
		// Trim trailing whitespace from non-code-fence line
		line = strings.TrimRight(line, " \t\r")

		if line == "" {
			if hasEmittedContent {
				pendingBlankLine = true
			}
			if isEOF {
				break
			}
			continue
		}

		// Non-blank line to emit
		if pendingBlankLine {
			if err := f.writeBlankLine(w); err != nil {
				return err
			}
			pendingBlankLine = false
		}

		norm := normalizeTabs(line)
		leadingSpaces := countLeadingSpaces(norm)
		indent := leadingSpaces / 2
		prefix := strings.Repeat("  ", indent)
		content := norm[leadingSpaces:]

		// A. Code Fence start
		if strings.HasPrefix(content, "```") {
			inCodeFence = true
			codeFenceIndent = prefix
			info := strings.TrimSpace(content[3:])
			var lineOut string
			if info != "" {
				lineOut = prefix + "```" + info
			} else {
				lineOut = prefix + "```"
			}
			if err := f.writeLine(w, lineOut); err != nil {
				return err
			}
			hasEmittedContent = true
			if isEOF {
				break
			}
			continue
		}

		// B. Table start
		if strings.HasPrefix(content, "::table") {
			inTable = true
			tableIndent = prefix
			tableHeader = formatTableHeader(content, prefix)
			tableRows = nil
			hasEmittedContent = true
			if isEOF {
				if err := f.flushTable(w, tableIndent, tableHeader, tableRows); err != nil {
					return err
				}
				inTable = false
				break
			}
			continue
		}

		// C. Metadata @key: value
		if strings.HasPrefix(content, "@") && strings.Contains(content, ":") {
			colonIdx := strings.Index(content, ":")
			key := strings.TrimSpace(content[1:colonIdx])
			val := strings.TrimSpace(content[colonIdx+1:])
			var lineOut string
			if val != "" {
				lineOut = prefix + "@" + key + ": " + val
			} else {
				lineOut = prefix + "@" + key + ":"
			}
			if err := f.writeLine(w, lineOut); err != nil {
				return err
			}
			hasEmittedContent = true
			if isEOF {
				break
			}
			continue
		}

		// D. Heading #{1,4}
		if isHeading(content) {
			level := countLeadingHashes(content)
			headingText := ""
			if level < len(content) {
				headingText = strings.TrimSpace(content[level:])
			}
			var lineOut string
			if headingText != "" {
				lineOut = prefix + strings.Repeat("#", level) + " " + headingText
			} else {
				lineOut = prefix + strings.Repeat("#", level)
			}
			if err := f.writeLine(w, lineOut); err != nil {
				return err
			}
			hasEmittedContent = true
			if isEOF {
				break
			}
			continue
		}

		// E. Divider ---
		if content == "---" {
			if err := f.writeLine(w, prefix+"---"); err != nil {
				return err
			}
			hasEmittedContent = true
			if isEOF {
				break
			}
			continue
		}

		// F. Task list [ ], [x], [X]
		if ok, mark, taskText := isTask(content); ok {
			var lineOut string
			if taskText != "" {
				lineOut = prefix + "[" + mark + "] " + taskText
			} else {
				lineOut = prefix + "[" + mark + "]"
			}
			if err := f.writeLine(w, lineOut); err != nil {
				return err
			}
			hasEmittedContent = true
			if isEOF {
				break
			}
			continue
		}

		// G. Unordered list -
		if ok, itemText := isUnordered(content); ok {
			var lineOut string
			if itemText != "" {
				lineOut = prefix + "- " + itemText
			} else {
				lineOut = prefix + "-"
			}
			if err := f.writeLine(w, lineOut); err != nil {
				return err
			}
			hasEmittedContent = true
			if isEOF {
				break
			}
			continue
		}

		// H. Ordered list 1. or .
		if ok, num, itemText := isOrdered(content); ok {
			var lineOut string
			if itemText != "" {
				lineOut = prefix + num + " " + itemText
			} else {
				lineOut = prefix + num
			}
			if err := f.writeLine(w, lineOut); err != nil {
				return err
			}
			hasEmittedContent = true
			if isEOF {
				break
			}
			continue
		}

		// I. Callout !
		if strings.HasPrefix(content, "!") {
			cType, title := parseCallout(content)
			var lineOut string
			if title != "" {
				lineOut = prefix + "!" + cType + ": " + title
			} else {
				lineOut = prefix + "!" + cType
			}
			if err := f.writeLine(w, lineOut); err != nil {
				return err
			}
			hasEmittedContent = true
			if isEOF {
				break
			}
			continue
		}

		// J. Blockquote >
		if strings.HasPrefix(content, ">") {
			quoteText := strings.TrimSpace(strings.TrimPrefix(content, ">"))
			var lineOut string
			if quoteText != "" {
				lineOut = prefix + "> " + quoteText
			} else {
				lineOut = prefix + ">"
			}
			if err := f.writeLine(w, lineOut); err != nil {
				return err
			}
			hasEmittedContent = true
			if isEOF {
				break
			}
			continue
		}

		// K. Paragraph / prose
		lineOut := prefix + content
		if err := f.writeLine(w, lineOut); err != nil {
			return err
		}
		hasEmittedContent = true

		if isEOF {
			break
		}
	}

	// Flush unclosed table at EOF if any
	if inTable {
		if err := f.flushTable(w, tableIndent, tableHeader, tableRows); err != nil {
			return err
		}
	}

	return nil
}

// FormatString formats a Vexil source string into a canonical formatted string.
func (f *Formatter) FormatString(s string) (string, error) {
	var buf bytes.Buffer
	if err := f.Format(strings.NewReader(s), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// FormatBytes formats a Vexil source byte slice into a canonical formatted byte slice.
func (f *Formatter) FormatBytes(b []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := f.Format(bytes.NewReader(b), &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Format formats a Vexil source stream from r to w using default settings.
func Format(r io.Reader, w io.Writer) error {
	return NewFormatter().Format(r, w)
}

// FormatString formats a Vexil source string using default settings.
func FormatString(s string) (string, error) {
	return NewFormatter().FormatString(s)
}

// FormatBytes formats a Vexil source byte slice using default settings.
func FormatBytes(b []byte) ([]byte, error) {
	return NewFormatter().FormatBytes(b)
}

func (f *Formatter) writeLine(w io.Writer, s string) error {
	if _, err := io.WriteString(w, s+"\n"); err != nil {
		return fmt.Errorf("formatter: write line: %w", err)
	}
	return nil
}

func (f *Formatter) writeBlankLine(w io.Writer) error {
	if _, err := io.WriteString(w, "\n"); err != nil {
		return fmt.Errorf("formatter: write blank line: %w", err)
	}
	return nil
}

func (f *Formatter) flushTable(w io.Writer, tableIndent string, header string, rows [][]string) error {
	if header != "" {
		if err := f.writeLine(w, header); err != nil {
			return err
		}
	}

	if len(rows) > 0 {
		if f.alignTableColumns {
			maxCols := 0
			for _, r := range rows {
				if len(r) > maxCols {
					maxCols = len(r)
				}
			}

			colWidths := make([]int, maxCols)
			for _, r := range rows {
				for c, cell := range r {
					if len(cell) > colWidths[c] {
						colWidths[c] = len(cell)
					}
				}
			}

			for _, r := range rows {
				if len(r) == 0 {
					if err := f.writeLine(w, tableIndent+"-"); err != nil {
						return err
					}
					continue
				}

				var sb strings.Builder
				sb.WriteString(tableIndent)
				sb.WriteString("- ")
				for c := 0; c < len(r); c++ {
					cell := r[c]
					if c > 0 {
						sb.WriteString(" | ")
					}
					if c < len(r)-1 {
						sb.WriteString(cell)
						if pad := colWidths[c] - len(cell); pad > 0 {
							sb.WriteString(strings.Repeat(" ", pad))
						}
					} else {
						sb.WriteString(cell)
					}
				}
				if err := f.writeLine(w, sb.String()); err != nil {
					return err
				}
			}
		} else {
			for _, r := range rows {
				if len(r) == 0 {
					if err := f.writeLine(w, tableIndent+"-"); err != nil {
						return err
					}
					continue
				}

				var sb strings.Builder
				sb.WriteString(tableIndent)
				sb.WriteString("- ")
				sb.WriteString(strings.Join(r, " | "))
				if err := f.writeLine(w, sb.String()); err != nil {
					return err
				}
			}
		}
	}

	return f.writeLine(w, tableIndent+"::")
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

func isUnordered(s string) (bool, string) {
	if s == "-" {
		return true, ""
	}
	if strings.HasPrefix(s, "- ") || strings.HasPrefix(s, "-\t") {
		return true, strings.TrimSpace(strings.TrimPrefix(s, "-"))
	}
	return false, ""
}

func isOrdered(s string) (bool, string, string) {
	if s == "." {
		return true, ".", ""
	}
	if strings.HasPrefix(s, ". ") || strings.HasPrefix(s, ".\t") {
		return true, ".", strings.TrimSpace(s[2:])
	}
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i > 0 && i < len(s) && s[i] == '.' {
		numPrefix := s[:i+1]
		if i+1 == len(s) {
			return true, numPrefix, ""
		}
		if s[i+1] == ' ' || s[i+1] == '\t' {
			return true, numPrefix, strings.TrimSpace(s[i+2:])
		}
	}
	return false, "", ""
}

func isTask(s string) (bool, string, string) {
	if len(s) >= 3 && s[0] == '[' && (s[1] == ' ' || s[1] == 'x' || s[1] == 'X') && s[2] == ']' {
		mark := string(s[1])
		if len(s) == 3 {
			return true, mark, ""
		}
		if s[3] == ' ' || s[3] == '\t' {
			return true, mark, strings.TrimSpace(s[4:])
		}
		return true, mark, strings.TrimSpace(s[3:])
	}
	return false, "", ""
}

func parseCallout(s string) (string, string) {
	rest := s[1:]
	if colonIdx := strings.Index(rest, ":"); colonIdx != -1 {
		cType := strings.TrimSpace(rest[:colonIdx])
		title := strings.TrimSpace(rest[colonIdx+1:])
		return cType, title
	}
	return strings.TrimSpace(rest), ""
}

func formatTableHeader(content string, prefix string) string {
	remainder := strings.TrimSpace(strings.TrimPrefix(content, "::table"))
	if openBracket := strings.Index(remainder, "["); openBracket != -1 {
		if closeBracket := strings.LastIndex(remainder, "]"); closeBracket > openBracket {
			headerStr := remainder[openBracket+1 : closeBracket]
			var parts []string
			if strings.Contains(headerStr, "|") {
				parts = parser.SplitTableCells(headerStr)
			} else if strings.Contains(headerStr, ",") {
				rawParts := strings.Split(headerStr, ",")
				for _, p := range rawParts {
					pTrim := strings.TrimSpace(p)
					if len(pTrim) > 0 {
						parts = append(parts, pTrim)
					}
				}
			} else if strings.TrimSpace(headerStr) != "" {
				parts = []string{strings.TrimSpace(headerStr)}
			}
			if len(parts) > 0 {
				return prefix + "::table [" + strings.Join(parts, ", ") + "]"
			}
			return prefix + "::table []"
		}
	}
	if remainder != "" {
		return prefix + "::table " + remainder
	}
	return prefix + "::table"
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
	if ok, _, _ := isTask(s); ok {
		return true
	}
	if ok, _ := isUnordered(s); ok {
		return true
	}
	if ok, _, _ := isOrdered(s); ok {
		return true
	}
	return false
}
