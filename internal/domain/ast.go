package domain

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Canonical AST node type constants.
const (
	TypeDocument      = "document"
	TypeHeading       = "heading"
	TypeParagraph     = "paragraph"
	TypeUnorderedList = "unordered_list"
	TypeOrderedList   = "ordered_list"
	TypeListItem      = "list_item"
	TypeTaskList      = "task_list"
	TypeTaskItem      = "task_item"
	TypeCallout       = "callout"
	TypeBlockquote    = "blockquote"
	TypeCodeFence     = "code_fence"
	TypeTable         = "table"
	TypeDivider       = "divider"

	TypeText          = "text"
	TypeStrong        = "strong"
	TypeEmphasis      = "emphasis"
	TypeInlineCode    = "inline_code"
	TypeStrikethrough = "strikethrough"
	TypeLink          = "link"
	TypeMedia         = "media"
)

var (
	// ErrUnknownNodeType is returned when deserializing an unrecognized node type.
	ErrUnknownNodeType = errors.New("unknown node type")
	// ErrNotAnInlineNode is returned when expecting an InlineNode but finding a block node.
	ErrNotAnInlineNode = errors.New("node is not an inline node")
)

// Node is the common interface implemented by all AST nodes in Vexil.
type Node interface {
	Type() string
}

// InlineNode is implemented by all inline-level AST nodes.
type InlineNode interface {
	Node
	isInlineNode()
}

// Document represents the top-level Vexil root AST node.
type Document struct {
	Meta     map[string]string `json:"meta,omitempty"`
	Children []Node            `json:"children,omitempty"`
}

func (d *Document) Type() string { return TypeDocument }

func (d *Document) MarshalJSON() ([]byte, error) {
	if d == nil {
		return []byte("null"), nil
	}
	type Alias Document
	return json.Marshal(&struct {
		Type string `json:"type"`
		*Alias
	}{
		Type:  d.Type(),
		Alias: (*Alias)(d),
	})
}

func (d *Document) UnmarshalJSON(data []byte) error {
	var raw struct {
		Meta     map[string]string `json:"meta"`
		Children []json.RawMessage `json:"children"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("unmarshal document raw fields: %w", err)
	}
	d.Meta = raw.Meta
	d.Children = make([]Node, 0, len(raw.Children))
	for _, childRaw := range raw.Children {
		node, err := UnmarshalNode(childRaw)
		if err != nil {
			return fmt.Errorf("unmarshal document child: %w", err)
		}
		d.Children = append(d.Children, node)
	}
	return nil
}

// Heading represents a section header (#, ##, ###, ####).
type Heading struct {
	Level    int               `json:"level"`
	Meta     map[string]string `json:"meta,omitempty"`
	Children []InlineNode      `json:"children,omitempty"`
}

func (h *Heading) Type() string { return TypeHeading }

func (h *Heading) MarshalJSON() ([]byte, error) {
	if h == nil {
		return []byte("null"), nil
	}
	type Alias Heading
	return json.Marshal(&struct {
		Type string `json:"type"`
		*Alias
	}{
		Type:  h.Type(),
		Alias: (*Alias)(h),
	})
}

func (h *Heading) UnmarshalJSON(data []byte) error {
	var raw struct {
		Level    int               `json:"level"`
		Meta     map[string]string `json:"meta"`
		Children []json.RawMessage `json:"children"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("unmarshal heading raw fields: %w", err)
	}
	h.Level = raw.Level
	h.Meta = raw.Meta
	inlines, err := unmarshalInlineNodes(raw.Children)
	if err != nil {
		return fmt.Errorf("unmarshal heading inlines: %w", err)
	}
	h.Children = inlines
	return nil
}

// Paragraph represents a block of running prose.
type Paragraph struct {
	Meta     map[string]string `json:"meta,omitempty"`
	Children []InlineNode      `json:"children,omitempty"`
}

func (p *Paragraph) Type() string { return TypeParagraph }

func (p *Paragraph) MarshalJSON() ([]byte, error) {
	if p == nil {
		return []byte("null"), nil
	}
	type Alias Paragraph
	return json.Marshal(&struct {
		Type string `json:"type"`
		*Alias
	}{
		Type:  p.Type(),
		Alias: (*Alias)(p),
	})
}

func (p *Paragraph) UnmarshalJSON(data []byte) error {
	var raw struct {
		Meta     map[string]string `json:"meta"`
		Children []json.RawMessage `json:"children"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("unmarshal paragraph raw fields: %w", err)
	}
	p.Meta = raw.Meta
	inlines, err := unmarshalInlineNodes(raw.Children)
	if err != nil {
		return fmt.Errorf("unmarshal paragraph inlines: %w", err)
	}
	p.Children = inlines
	return nil
}

// ListItem represents an item in an ordered or unordered list.
type ListItem struct {
	Level    int          `json:"level"`
	Children []InlineNode `json:"children,omitempty"`
}

func (l *ListItem) Type() string { return TypeListItem }

func (l *ListItem) MarshalJSON() ([]byte, error) {
	if l == nil {
		return []byte("null"), nil
	}
	type Alias ListItem
	return json.Marshal(&struct {
		Type string `json:"type"`
		*Alias
	}{
		Type:  l.Type(),
		Alias: (*Alias)(l),
	})
}

func (l *ListItem) UnmarshalJSON(data []byte) error {
	var raw struct {
		Level    int               `json:"level"`
		Children []json.RawMessage `json:"children"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("unmarshal list item raw fields: %w", err)
	}
	l.Level = raw.Level
	inlines, err := unmarshalInlineNodes(raw.Children)
	if err != nil {
		return fmt.Errorf("unmarshal list item inlines: %w", err)
	}
	l.Children = inlines
	return nil
}

// UnorderedList represents a bulleted list.
type UnorderedList struct {
	Items []*ListItem       `json:"items,omitempty"`
	Meta  map[string]string `json:"meta,omitempty"`
}

func (u *UnorderedList) Type() string { return TypeUnorderedList }

func (u *UnorderedList) MarshalJSON() ([]byte, error) {
	if u == nil {
		return []byte("null"), nil
	}
	type Alias UnorderedList
	return json.Marshal(&struct {
		Type string `json:"type"`
		*Alias
	}{
		Type:  u.Type(),
		Alias: (*Alias)(u),
	})
}

// OrderedList represents a numbered list.
type OrderedList struct {
	Items []*ListItem       `json:"items,omitempty"`
	Meta  map[string]string `json:"meta,omitempty"`
}

func (o *OrderedList) Type() string { return TypeOrderedList }

func (o *OrderedList) MarshalJSON() ([]byte, error) {
	if o == nil {
		return []byte("null"), nil
	}
	type Alias OrderedList
	return json.Marshal(&struct {
		Type string `json:"type"`
		*Alias
	}{
		Type:  o.Type(),
		Alias: (*Alias)(o),
	})
}

// TaskItem represents an individual item in a task checklist.
type TaskItem struct {
	Checked  bool         `json:"checked"`
	Level    int          `json:"level"`
	Children []InlineNode `json:"children,omitempty"`
}

func (t *TaskItem) Type() string { return TypeTaskItem }

func (t *TaskItem) MarshalJSON() ([]byte, error) {
	if t == nil {
		return []byte("null"), nil
	}
	type Alias TaskItem
	return json.Marshal(&struct {
		Type string `json:"type"`
		*Alias
	}{
		Type:  t.Type(),
		Alias: (*Alias)(t),
	})
}

func (t *TaskItem) UnmarshalJSON(data []byte) error {
	var raw struct {
		Checked  bool              `json:"checked"`
		Level    int               `json:"level"`
		Children []json.RawMessage `json:"children"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("unmarshal task item raw fields: %w", err)
	}
	t.Checked = raw.Checked
	t.Level = raw.Level
	inlines, err := unmarshalInlineNodes(raw.Children)
	if err != nil {
		return fmt.Errorf("unmarshal task item inlines: %w", err)
	}
	t.Children = inlines
	return nil
}

// TaskList represents a checklist of actionable items.
type TaskList struct {
	Items []*TaskItem       `json:"items,omitempty"`
	Meta  map[string]string `json:"meta,omitempty"`
}

func (t *TaskList) Type() string { return TypeTaskList }

func (t *TaskList) MarshalJSON() ([]byte, error) {
	if t == nil {
		return []byte("null"), nil
	}
	type Alias TaskList
	return json.Marshal(&struct {
		Type string `json:"type"`
		*Alias
	}{
		Type:  t.Type(),
		Alias: (*Alias)(t),
	})
}

// Callout represents a distinguished message box (!warning, !note, etc.).
type Callout struct {
	CalloutType string            `json:"callout_type"`
	Title       string            `json:"title,omitempty"`
	Meta        map[string]string `json:"meta,omitempty"`
	Children    []InlineNode      `json:"children,omitempty"`
}

func (c *Callout) Type() string { return TypeCallout }

func (c *Callout) MarshalJSON() ([]byte, error) {
	if c == nil {
		return []byte("null"), nil
	}
	type Alias Callout
	return json.Marshal(&struct {
		Type string `json:"type"`
		*Alias
	}{
		Type:  c.Type(),
		Alias: (*Alias)(c),
	})
}

func (c *Callout) UnmarshalJSON(data []byte) error {
	var raw struct {
		CalloutType string            `json:"callout_type"`
		Title       string            `json:"title"`
		Meta        map[string]string `json:"meta"`
		Children    []json.RawMessage `json:"children"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("unmarshal callout raw fields: %w", err)
	}
	c.CalloutType = raw.CalloutType
	c.Title = raw.Title
	c.Meta = raw.Meta
	inlines, err := unmarshalInlineNodes(raw.Children)
	if err != nil {
		return fmt.Errorf("unmarshal callout inlines: %w", err)
	}
	c.Children = inlines
	return nil
}

// Blockquote represents an indented quotation (>).
type Blockquote struct {
	Meta     map[string]string `json:"meta,omitempty"`
	Children []InlineNode      `json:"children,omitempty"`
}

func (b *Blockquote) Type() string { return TypeBlockquote }

func (b *Blockquote) MarshalJSON() ([]byte, error) {
	if b == nil {
		return []byte("null"), nil
	}
	type Alias Blockquote
	return json.Marshal(&struct {
		Type string `json:"type"`
		*Alias
	}{
		Type:  b.Type(),
		Alias: (*Alias)(b),
	})
}

func (b *Blockquote) UnmarshalJSON(data []byte) error {
	var raw struct {
		Meta     map[string]string `json:"meta"`
		Children []json.RawMessage `json:"children"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("unmarshal blockquote raw fields: %w", err)
	}
	b.Meta = raw.Meta
	inlines, err := unmarshalInlineNodes(raw.Children)
	if err != nil {
		return fmt.Errorf("unmarshal blockquote inlines: %w", err)
	}
	b.Children = inlines
	return nil
}

// CodeFence represents a fenced block of source code or verbatim text.
type CodeFence struct {
	Lang       string            `json:"lang,omitempty"`
	Identifier string            `json:"identifier,omitempty"`
	Content    string            `json:"content"`
	Meta       map[string]string `json:"meta,omitempty"`
}

func (c *CodeFence) Type() string { return TypeCodeFence }

func (c *CodeFence) MarshalJSON() ([]byte, error) {
	if c == nil {
		return []byte("null"), nil
	}
	type Alias CodeFence
	return json.Marshal(&struct {
		Type string `json:"type"`
		*Alias
	}{
		Type:  c.Type(),
		Alias: (*Alias)(c),
	})
}

// Table represents a 2D structured table (::table ... ::).
type Table struct {
	Headers []string          `json:"headers,omitempty"`
	Rows    [][][]InlineNode  `json:"rows,omitempty"`
	Meta    map[string]string `json:"meta,omitempty"`
}

func (t *Table) Type() string { return TypeTable }

func (t *Table) MarshalJSON() ([]byte, error) {
	if t == nil {
		return []byte("null"), nil
	}
	type Alias Table
	return json.Marshal(&struct {
		Type string `json:"type"`
		*Alias
	}{
		Type:  t.Type(),
		Alias: (*Alias)(t),
	})
}

func (t *Table) UnmarshalJSON(data []byte) error {
	var raw struct {
		Headers []string              `json:"headers"`
		Rows    [][][]json.RawMessage `json:"rows"`
		Meta    map[string]string     `json:"meta"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("unmarshal table raw fields: %w", err)
	}
	t.Headers = raw.Headers
	t.Meta = raw.Meta
	t.Rows = make([][][]InlineNode, len(raw.Rows))
	for r, rowRaw := range raw.Rows {
		t.Rows[r] = make([][]InlineNode, len(rowRaw))
		for c, cellRaw := range rowRaw {
			cellInlines, err := unmarshalInlineNodes(cellRaw)
			if err != nil {
				return fmt.Errorf("unmarshal table cell (%d,%d): %w", r, c, err)
			}
			t.Rows[r][c] = cellInlines
		}
	}
	return nil
}

// Divider represents a horizontal thematic break (---).
type Divider struct {
	Meta map[string]string `json:"meta,omitempty"`
}

func (d *Divider) Type() string { return TypeDivider }

func (d *Divider) MarshalJSON() ([]byte, error) {
	if d == nil {
		return []byte("null"), nil
	}
	type Alias Divider
	return json.Marshal(&struct {
		Type string `json:"type"`
		*Alias
	}{
		Type:  d.Type(),
		Alias: (*Alias)(d),
	})
}

// --- Inline AST Nodes ---

// Text represents a leaf plain-text inline segment.
type Text struct {
	Value string `json:"value"`
}

func (t *Text) Type() string  { return TypeText }
func (t *Text) isInlineNode() {}

func (t *Text) MarshalJSON() ([]byte, error) {
	if t == nil {
		return []byte("null"), nil
	}
	type Alias Text
	return json.Marshal(&struct {
		Type string `json:"type"`
		*Alias
	}{
		Type:  t.Type(),
		Alias: (*Alias)(t),
	})
}

// Strong represents bold inline text (*bold*).
type Strong struct {
	Children []InlineNode `json:"children,omitempty"`
}

func (s *Strong) Type() string  { return TypeStrong }
func (s *Strong) isInlineNode() {}

func (s *Strong) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("null"), nil
	}
	type Alias Strong
	return json.Marshal(&struct {
		Type string `json:"type"`
		*Alias
	}{
		Type:  s.Type(),
		Alias: (*Alias)(s),
	})
}

func (s *Strong) UnmarshalJSON(data []byte) error {
	var raw struct {
		Children []json.RawMessage `json:"children"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("unmarshal strong raw fields: %w", err)
	}
	inlines, err := unmarshalInlineNodes(raw.Children)
	if err != nil {
		return fmt.Errorf("unmarshal strong inlines: %w", err)
	}
	s.Children = inlines
	return nil
}

// Emphasis represents italicized inline text (_italic_).
type Emphasis struct {
	Children []InlineNode `json:"children,omitempty"`
}

func (e *Emphasis) Type() string  { return TypeEmphasis }
func (e *Emphasis) isInlineNode() {}

func (e *Emphasis) MarshalJSON() ([]byte, error) {
	if e == nil {
		return []byte("null"), nil
	}
	type Alias Emphasis
	return json.Marshal(&struct {
		Type string `json:"type"`
		*Alias
	}{
		Type:  e.Type(),
		Alias: (*Alias)(e),
	})
}

func (e *Emphasis) UnmarshalJSON(data []byte) error {
	var raw struct {
		Children []json.RawMessage `json:"children"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("unmarshal emphasis raw fields: %w", err)
	}
	inlines, err := unmarshalInlineNodes(raw.Children)
	if err != nil {
		return fmt.Errorf("unmarshal emphasis inlines: %w", err)
	}
	e.Children = inlines
	return nil
}

// InlineCode represents verbatim monospaced inline text (`code`).
type InlineCode struct {
	Value string `json:"value"`
}

func (c *InlineCode) Type() string  { return TypeInlineCode }
func (c *InlineCode) isInlineNode() {}

func (c *InlineCode) MarshalJSON() ([]byte, error) {
	if c == nil {
		return []byte("null"), nil
	}
	type Alias InlineCode
	return json.Marshal(&struct {
		Type string `json:"type"`
		*Alias
	}{
		Type:  c.Type(),
		Alias: (*Alias)(c),
	})
}

// Strikethrough represents struck-through inline text (~strike~).
type Strikethrough struct {
	Children []InlineNode `json:"children,omitempty"`
}

func (s *Strikethrough) Type() string  { return TypeStrikethrough }
func (s *Strikethrough) isInlineNode() {}

func (s *Strikethrough) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("null"), nil
	}
	type Alias Strikethrough
	return json.Marshal(&struct {
		Type string `json:"type"`
		*Alias
	}{
		Type:  s.Type(),
		Alias: (*Alias)(s),
	})
}

func (s *Strikethrough) UnmarshalJSON(data []byte) error {
	var raw struct {
		Children []json.RawMessage `json:"children"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("unmarshal strikethrough raw fields: %w", err)
	}
	inlines, err := unmarshalInlineNodes(raw.Children)
	if err != nil {
		return fmt.Errorf("unmarshal strikethrough inlines: %w", err)
	}
	s.Children = inlines
	return nil
}

// Link represents an inline hyperlink ([label](target)).
type Link struct {
	Label  []InlineNode `json:"label,omitempty"`
	Target string       `json:"target"`
}

func (l *Link) Type() string  { return TypeLink }
func (l *Link) isInlineNode() {}

func (l *Link) MarshalJSON() ([]byte, error) {
	if l == nil {
		return []byte("null"), nil
	}
	type Alias Link
	return json.Marshal(&struct {
		Type string `json:"type"`
		*Alias
	}{
		Type:  l.Type(),
		Alias: (*Alias)(l),
	})
}

func (l *Link) UnmarshalJSON(data []byte) error {
	var raw struct {
		Label  []json.RawMessage `json:"label"`
		Target string            `json:"target"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("unmarshal link raw fields: %w", err)
	}
	l.Target = raw.Target
	inlines, err := unmarshalInlineNodes(raw.Label)
	if err != nil {
		return fmt.Errorf("unmarshal link label: %w", err)
	}
	l.Label = inlines
	return nil
}

// Media represents an inline image or video asset ([alt](target)).
type Media struct {
	Alt    string `json:"alt,omitempty"`
	Target string `json:"target"`
}

func (m *Media) Type() string  { return TypeMedia }
func (m *Media) isInlineNode() {}

func (m *Media) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	type Alias Media
	return json.Marshal(&struct {
		Type string `json:"type"`
		*Alias
	}{
		Type:  m.Type(),
		Alias: (*Alias)(m),
	})
}

// --- JSON Helpers ---

func unmarshalInlineNodes(raws []json.RawMessage) ([]InlineNode, error) {
	if raws == nil {
		return nil, nil
	}
	res := make([]InlineNode, 0, len(raws))
	for _, raw := range raws {
		inode, err := UnmarshalInlineNode(raw)
		if err != nil {
			return nil, fmt.Errorf("unmarshal inline node element: %w", err)
		}
		res = append(res, inode)
	}
	return res, nil
}

// UnmarshalNode parses a JSON object with a "type" field into the corresponding concrete Node.
func UnmarshalNode(data []byte) (Node, error) {
	var env struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("unmarshal node envelope: %w", err)
	}

	switch env.Type {
	case TypeDocument:
		var doc Document
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("unmarshal Document: %w", err)
		}
		return &doc, nil
	case TypeHeading:
		var h Heading
		if err := json.Unmarshal(data, &h); err != nil {
			return nil, fmt.Errorf("unmarshal Heading: %w", err)
		}
		return &h, nil
	case TypeParagraph:
		var p Paragraph
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("unmarshal Paragraph: %w", err)
		}
		return &p, nil
	case TypeUnorderedList:
		var ul UnorderedList
		type ulAlias UnorderedList
		if err := json.Unmarshal(data, (*ulAlias)(&ul)); err != nil {
			return nil, fmt.Errorf("unmarshal UnorderedList: %w", err)
		}
		return &ul, nil
	case TypeOrderedList:
		var ol OrderedList
		type olAlias OrderedList
		if err := json.Unmarshal(data, (*olAlias)(&ol)); err != nil {
			return nil, fmt.Errorf("unmarshal OrderedList: %w", err)
		}
		return &ol, nil
	case TypeListItem:
		var li ListItem
		if err := json.Unmarshal(data, &li); err != nil {
			return nil, fmt.Errorf("unmarshal ListItem: %w", err)
		}
		return &li, nil
	case TypeTaskList:
		var tl TaskList
		type tlAlias TaskList
		if err := json.Unmarshal(data, (*tlAlias)(&tl)); err != nil {
			return nil, fmt.Errorf("unmarshal TaskList: %w", err)
		}
		return &tl, nil
	case TypeTaskItem:
		var ti TaskItem
		if err := json.Unmarshal(data, &ti); err != nil {
			return nil, fmt.Errorf("unmarshal TaskItem: %w", err)
		}
		return &ti, nil
	case TypeCallout:
		var c Callout
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, fmt.Errorf("unmarshal Callout: %w", err)
		}
		return &c, nil
	case TypeBlockquote:
		var b Blockquote
		if err := json.Unmarshal(data, &b); err != nil {
			return nil, fmt.Errorf("unmarshal Blockquote: %w", err)
		}
		return &b, nil
	case TypeCodeFence:
		var cf CodeFence
		type cfAlias CodeFence
		if err := json.Unmarshal(data, (*cfAlias)(&cf)); err != nil {
			return nil, fmt.Errorf("unmarshal CodeFence: %w", err)
		}
		return &cf, nil
	case TypeTable:
		var tb Table
		if err := json.Unmarshal(data, &tb); err != nil {
			return nil, fmt.Errorf("unmarshal Table: %w", err)
		}
		return &tb, nil
	case TypeDivider:
		var d Divider
		type dAlias Divider
		if err := json.Unmarshal(data, (*dAlias)(&d)); err != nil {
			return nil, fmt.Errorf("unmarshal Divider: %w", err)
		}
		return &d, nil
	case TypeText:
		var t Text
		type tAlias Text
		if err := json.Unmarshal(data, (*tAlias)(&t)); err != nil {
			return nil, fmt.Errorf("unmarshal Text: %w", err)
		}
		return &t, nil
	case TypeStrong:
		var s Strong
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, fmt.Errorf("unmarshal Strong: %w", err)
		}
		return &s, nil
	case TypeEmphasis:
		var e Emphasis
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, fmt.Errorf("unmarshal Emphasis: %w", err)
		}
		return &e, nil
	case TypeInlineCode:
		var ic InlineCode
		type icAlias InlineCode
		if err := json.Unmarshal(data, (*icAlias)(&ic)); err != nil {
			return nil, fmt.Errorf("unmarshal InlineCode: %w", err)
		}
		return &ic, nil
	case TypeStrikethrough:
		var st Strikethrough
		if err := json.Unmarshal(data, &st); err != nil {
			return nil, fmt.Errorf("unmarshal Strikethrough: %w", err)
		}
		return &st, nil
	case TypeLink:
		var l Link
		if err := json.Unmarshal(data, &l); err != nil {
			return nil, fmt.Errorf("unmarshal Link: %w", err)
		}
		return &l, nil
	case TypeMedia:
		var m Media
		type mAlias Media
		if err := json.Unmarshal(data, (*mAlias)(&m)); err != nil {
			return nil, fmt.Errorf("unmarshal Media: %w", err)
		}
		return &m, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownNodeType, env.Type)
	}
}

// UnmarshalInlineNode parses a JSON object into a concrete InlineNode.
func UnmarshalInlineNode(data []byte) (InlineNode, error) {
	node, err := UnmarshalNode(data)
	if err != nil {
		return nil, fmt.Errorf("unmarshal node: %w", err)
	}
	inlineNode, ok := node.(InlineNode)
	if !ok {
		return nil, fmt.Errorf("%w: %T is not an InlineNode", ErrNotAnInlineNode, node)
	}
	return inlineNode, nil
}

// --- AST Traversal (Visitor Pattern) ---

// Visitor defines the interface for traversing an AST.
// If Visit returns a non-nil Visitor, Walk continues with the returned Visitor for children.
type Visitor interface {
	Visit(node Node) (w Visitor)
}

// Walk traverses an AST in depth-first order.
func Walk(v Visitor, node Node) {
	if node == nil {
		return
	}
	v = v.Visit(node)
	if v == nil {
		return
	}

	switch n := node.(type) {
	case *Document:
		for _, child := range n.Children {
			Walk(v, child)
		}
	case *Heading:
		for _, child := range n.Children {
			Walk(v, child)
		}
	case *Paragraph:
		for _, child := range n.Children {
			Walk(v, child)
		}
	case *UnorderedList:
		for _, item := range n.Items {
			Walk(v, item)
		}
	case *OrderedList:
		for _, item := range n.Items {
			Walk(v, item)
		}
	case *ListItem:
		for _, child := range n.Children {
			Walk(v, child)
		}
	case *TaskList:
		for _, item := range n.Items {
			Walk(v, item)
		}
	case *TaskItem:
		for _, child := range n.Children {
			Walk(v, child)
		}
	case *Callout:
		for _, child := range n.Children {
			Walk(v, child)
		}
	case *Blockquote:
		for _, child := range n.Children {
			Walk(v, child)
		}
	case *Table:
		for _, row := range n.Rows {
			for _, cell := range row {
				for _, in := range cell {
					Walk(v, in)
				}
			}
		}
	case *Strong:
		for _, child := range n.Children {
			Walk(v, child)
		}
	case *Emphasis:
		for _, child := range n.Children {
			Walk(v, child)
		}
	case *Strikethrough:
		for _, child := range n.Children {
			Walk(v, child)
		}
	case *Link:
		for _, child := range n.Label {
			Walk(v, child)
		}
	case *CodeFence, *Divider, *Text, *InlineCode, *Media:
		// Leaf nodes
	}
}

type inspector func(Node) bool

func (f inspector) Visit(node Node) Visitor {
	if f(node) {
		return f
	}
	return nil
}

// Inspect traverses an AST in depth-first order. If f returns true, Inspect
// continues to visit children; if false, it skips children of that node.
func Inspect(node Node, f func(Node) bool) {
	Walk(inspector(f), node)
}
