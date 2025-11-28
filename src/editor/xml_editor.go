package editor

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// XMLEditor manages XML DOM style editing with undo/redo support.
type XMLEditor struct {
	path      string
	root      *XMLNode
	index     map[string]*XMLNode
	modified  bool
	undoStack []*xmlCommand
	redoStack []*xmlCommand
}

// XMLNode represents a DOM element.
type XMLNode struct {
	Tag        string
	ID         string
	Attributes []XMLAttribute
	attrIndex  map[string]int
	Text       string
	Children   []*XMLNode
	Parent     *XMLNode
}

// XMLAttribute retains attribute order.
type XMLAttribute struct {
	Name  string
	Value string
}

type xmlCommand struct {
	description string
	before      *XMLNode
	after       *XMLNode
}

// NewXMLEditor constructs an editor for the provided root.
func NewXMLEditor(path string, root *XMLNode, modified bool) *XMLEditor {
	index := map[string]*XMLNode{}
	rebuildIndex(root, index)
	return &XMLEditor{
		path:     path,
		root:     root,
		index:    index,
		modified: modified,
	}
}

// NewDefaultXMLDocument builds the lab default root.
func NewDefaultXMLDocument(withLog bool) *XMLNode {
	attrs := []XMLAttribute{{Name: "id", Value: "root"}}
	attrIndex := map[string]int{"id": 0}
	if withLog {
		attrs = append(attrs, XMLAttribute{Name: "log", Value: "true"})
		attrIndex["log"] = 1
	}
	return &XMLNode{
		Tag:        "root",
		ID:         "root",
		Attributes: attrs,
		attrIndex:  attrIndex,
	}
}

// ParseXMLEditor parses XML content into an editor.
func ParseXMLEditor(path string, data []byte) (*XMLEditor, error) {
	root, err := parseXML(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return NewXMLEditor(path, root, false), nil
}

// Path returns the backing file path.
func (e *XMLEditor) Path() string {
	return e.path
}

// Name returns the file name for display.
func (e *XMLEditor) Name() string {
	return filepath.Base(e.path)
}

// Type returns the editor type.
func (e *XMLEditor) Type() Type {
	return TypeXML
}

// IsModified reports whether the editor has unsaved changes.
func (e *XMLEditor) IsModified() bool {
	return e.modified
}

// SetModified overrides the modified state.
func (e *XMLEditor) SetModified(value bool) {
	e.modified = value
}

// Content serializes the XML tree.
func (e *XMLEditor) Content() (string, error) {
	if e.root == nil {
		return "", errors.New("缺少根元素")
	}
	var buf bytes.Buffer
	buf.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	writeNode(&buf, e.root, 0)
	return buf.String(), nil
}

// Undo reverts the last operation.
func (e *XMLEditor) Undo() error {
	if len(e.undoStack) == 0 {
		return errors.New("没有可撤销的操作")
	}
	last := e.undoStack[len(e.undoStack)-1]
	e.undoStack = e.undoStack[:len(e.undoStack)-1]
	e.applySnapshot(last.before)
	e.redoStack = append(e.redoStack, last)
	e.modified = true
	return nil
}

// Redo reapplies the last undone operation.
func (e *XMLEditor) Redo() error {
	if len(e.redoStack) == 0 {
		return errors.New("没有可重做的操作")
	}
	last := e.redoStack[len(e.redoStack)-1]
	e.redoStack = e.redoStack[:len(e.redoStack)-1]
	e.applySnapshot(last.after)
	e.undoStack = append(e.undoStack, last)
	e.modified = true
	return nil
}

// InsertBefore inserts a sibling element before the target.
func (e *XMLEditor) InsertBefore(tag, newID, targetID string, text *string) error {
	return e.execute("insert-before", func() error {
		if _, exists := e.index[newID]; exists {
			return fmt.Errorf("元素 ID 已存在: %s", newID)
		}
		target, ok := e.index[targetID]
		if !ok {
			return fmt.Errorf("目标元素不存在: %s", targetID)
		}
		if target.Parent == nil {
			return errors.New("不能在根元素前插入元素")
		}
		parent := target.Parent
		if strings.TrimSpace(parent.Text) != "" {
			return errors.New("该元素已有文本内容，不支持混合内容")
		}
		node := createXMLNode(tag, newID, text)
		node.Parent = parent
		idx := indexOfChild(parent, target)
		parent.Children = append(parent.Children[:idx], append([]*XMLNode{node}, parent.Children[idx:]...)...)
		registerNode(node, e.index)
		return nil
	})
}

// AppendChild appends a child element to the parent.
func (e *XMLEditor) AppendChild(tag, newID, parentID string, text *string) error {
	return e.execute("append-child", func() error {
		if _, exists := e.index[newID]; exists {
			return fmt.Errorf("元素 ID 已存在: %s", newID)
		}
		parent, ok := e.index[parentID]
		if !ok {
			return fmt.Errorf("父元素不存在: %s", parentID)
		}
		if strings.TrimSpace(parent.Text) != "" {
			return errors.New("该元素已有文本内容，不支持混合内容")
		}
		node := createXMLNode(tag, newID, text)
		node.Parent = parent
		parent.Children = append(parent.Children, node)
		registerNode(node, e.index)
		return nil
	})
}

// EditID renames an element id.
func (e *XMLEditor) EditID(oldID, newID string) error {
	return e.execute("edit-id", func() error {
		node, ok := e.index[oldID]
		if !ok {
			return fmt.Errorf("元素不存在: %s", oldID)
		}
		if node.Parent == nil {
			return errors.New("不允许修改根元素 ID")
		}
		if _, exists := e.index[newID]; exists {
			return fmt.Errorf("目标 ID 已存在: %s", newID)
		}
		delete(e.index, oldID)
		node.ID = newID
		if node.attrIndex == nil {
			node.attrIndex = map[string]int{}
		}
		if idx, ok := node.attrIndex["id"]; ok {
			node.Attributes[idx].Value = newID
		} else {
			node.attrIndex["id"] = len(node.Attributes)
			node.Attributes = append(node.Attributes, XMLAttribute{Name: "id", Value: newID})
		}
		e.index[newID] = node
		return nil
	})
}

// EditText updates the text content of an element.
func (e *XMLEditor) EditText(elementID string, text string) error {
	return e.execute("edit-text", func() error {
		node, ok := e.index[elementID]
		if !ok {
			return fmt.Errorf("元素不存在: %s", elementID)
		}
		if len(node.Children) > 0 {
			return errors.New("该元素有子元素，不支持混合内容")
		}
		node.Text = text
		return nil
	})
}

// DeleteElement removes the specified element and its subtree.
func (e *XMLEditor) DeleteElement(elementID string) error {
	return e.execute("delete-element", func() error {
		node, ok := e.index[elementID]
		if !ok {
			return fmt.Errorf("元素不存在: %s", elementID)
		}
		if node.Parent == nil {
			return errors.New("不能删除根元素")
		}
		parent := node.Parent
		idx := indexOfChild(parent, node)
		parent.Children = append(parent.Children[:idx], parent.Children[idx+1:]...)
		removeFromIndex(node, e.index)
		return nil
	})
}

// TreeString renders the XML tree for display.
func (e *XMLEditor) TreeString() string {
	if e.root == nil {
		return ""
	}
	lines := []string{formatNodeLabel(e.root)}
	renderTree(e.root, "", &lines)
	return strings.Join(lines, "\n")
}

// TextNodes collects element texts for spell checking.
func (e *XMLEditor) TextNodes() []XMLTextNode {
	var result []XMLTextNode
	collectTextNodes(e.root, &result)
	return result
}

// RootAttributes exposes the root attribute map.
func (e *XMLEditor) RootAttributes() map[string]string {
	attrs := map[string]string{}
	if e.root == nil {
		return attrs
	}
	for _, attr := range e.root.Attributes {
		attrs[attr.Name] = attr.Value
	}
	return attrs
}

func (e *XMLEditor) execute(desc string, mutate func() error) error {
	before := cloneTree(e.root, nil)
	if err := mutate(); err != nil {
		return err
	}
	after := cloneTree(e.root, nil)
	cmd := &xmlCommand{description: desc, before: before, after: after}
	e.undoStack = append(e.undoStack, cmd)
	e.redoStack = nil
	e.modified = true
	return nil
}

func (e *XMLEditor) applySnapshot(snapshot *XMLNode) {
	cloned := cloneTree(snapshot, nil)
	index := map[string]*XMLNode{}
	rebuildIndex(cloned, index)
	e.root = cloned
	e.index = index
}

func createXMLNode(tag, id string, text *string) *XMLNode {
	attrIndex := map[string]int{"id": 0}
	attrs := []XMLAttribute{{Name: "id", Value: id}}
	node := &XMLNode{
		Tag:        tag,
		ID:         id,
		Attributes: attrs,
		attrIndex:  attrIndex,
	}
	if text != nil {
		node.Text = *text
	}
	return node
}

func registerNode(node *XMLNode, index map[string]*XMLNode) {
	if index == nil || node == nil {
		return
	}
	if node.attrIndex == nil {
		node.attrIndex = map[string]int{}
		for idx, attr := range node.Attributes {
			node.attrIndex[attr.Name] = idx
		}
	}
	index[node.ID] = node
}

func removeFromIndex(node *XMLNode, index map[string]*XMLNode) {
	if node == nil {
		return
	}
	delete(index, node.ID)
	for _, child := range node.Children {
		removeFromIndex(child, index)
	}
}

func indexOfChild(parent *XMLNode, child *XMLNode) int {
	for i, candidate := range parent.Children {
		if candidate == child {
			return i
		}
	}
	return -1
}

func renderTree(node *XMLNode, prefix string, lines *[]string) {
	if node == nil {
		return
	}
	textPresent := strings.TrimSpace(node.Text) != ""
	childCount := len(node.Children)
	for i, child := range node.Children {
		last := i == childCount-1 && !textPresent
		connector := "├── "
		nextPrefix := prefix + "│   "
		if last {
			connector = "└── "
			nextPrefix = prefix + "    "
		}
		*lines = append(*lines, prefix+connector+formatNodeLabel(child))
		renderTree(child, nextPrefix, lines)
	}
	if textPresent {
		textLine := fmt.Sprintf("\"%s\"", node.Text)
		connector := "└── "
		*lines = append(*lines, prefix+connector+textLine)
	}
}

func collectTextNodes(node *XMLNode, acc *[]XMLTextNode) {
	if node == nil {
		return
	}
	if strings.TrimSpace(node.Text) != "" {
		*acc = append(*acc, XMLTextNode{ElementID: node.ID, Text: strings.TrimSpace(node.Text)})
	}
	for _, child := range node.Children {
		collectTextNodes(child, acc)
	}
}

func formatNodeLabel(node *XMLNode) string {
	if node == nil {
		return ""
	}
	if len(node.Attributes) == 0 {
		return node.Tag
	}
	parts := make([]string, len(node.Attributes))
	for i, attr := range node.Attributes {
		parts[i] = fmt.Sprintf("%s=\"%s\"", attr.Name, attr.Value)
	}
	return fmt.Sprintf("%s [%s]", node.Tag, strings.Join(parts, ", "))
}

func writeNode(buf *bytes.Buffer, node *XMLNode, depth int) {
	if node == nil {
		return
	}
	indent := strings.Repeat("    ", depth)
	attrText := formatAttributes(node.Attributes)
	if len(node.Children) == 0 {
		if strings.TrimSpace(node.Text) == "" {
			fmt.Fprintf(buf, "%s<%s%s></%s>\n", indent, node.Tag, attrText, node.Tag)
			return
		}
		fmt.Fprintf(buf, "%s<%s%s>%s</%s>\n", indent, node.Tag, attrText, escapeText(node.Text), node.Tag)
		return
	}
	fmt.Fprintf(buf, "%s<%s%s>\n", indent, node.Tag, attrText)
	for _, child := range node.Children {
		writeNode(buf, child, depth+1)
	}
	fmt.Fprintf(buf, "%s</%s>\n", indent, node.Tag)
}

func formatAttributes(attrs []XMLAttribute) string {
	if len(attrs) == 0 {
		return ""
	}
	parts := make([]string, len(attrs))
	for i, attr := range attrs {
		parts[i] = fmt.Sprintf("%s=\"%s\"", attr.Name, escapeAttribute(attr.Value))
	}
	return " " + strings.Join(parts, " ")
}

func escapeText(text string) string {
	var buf bytes.Buffer
	if err := xml.EscapeText(&buf, []byte(text)); err != nil {
		return text
	}
	return buf.String()
}

func escapeAttribute(value string) string {
	return escapeText(value)
}

func cloneTree(node *XMLNode, parent *XMLNode) *XMLNode {
	if node == nil {
		return nil
	}
	cloned := &XMLNode{
		Tag:        node.Tag,
		ID:         node.ID,
		Text:       node.Text,
		Attributes: make([]XMLAttribute, len(node.Attributes)),
		attrIndex:  make(map[string]int, len(node.attrIndex)),
		Parent:     parent,
	}
	copy(cloned.Attributes, node.Attributes)
	for k, v := range node.attrIndex {
		cloned.attrIndex[k] = v
	}
	cloned.Children = make([]*XMLNode, len(node.Children))
	for i, child := range node.Children {
		clonedChild := cloneTree(child, cloned)
		cloned.Children[i] = clonedChild
	}
	return cloned
}

func rebuildIndex(node *XMLNode, index map[string]*XMLNode) {
	if node == nil {
		return
	}
	if node.attrIndex == nil {
		node.attrIndex = map[string]int{}
		for idx, attr := range node.Attributes {
			node.attrIndex[attr.Name] = idx
		}
	}
	index[node.ID] = node
	for _, child := range node.Children {
		child.Parent = node
		rebuildIndex(child, index)
	}
}

func parseXML(reader io.Reader) (*XMLNode, error) {
	decoder := xml.NewDecoder(reader)
	var stack []*XMLNode
	var root *XMLNode
	ids := map[string]struct{}{}

	for {
		token, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		switch tok := token.(type) {
		case xml.StartElement:
			node := &XMLNode{
				Tag:        tok.Name.Local,
				Attributes: make([]XMLAttribute, 0, len(tok.Attr)),
				attrIndex:  map[string]int{},
			}
			for idx, attr := range tok.Attr {
				name := attr.Name.Local
				value := attr.Value
				node.Attributes = append(node.Attributes, XMLAttribute{Name: name, Value: value})
				node.attrIndex[name] = idx
				if name == "id" {
					node.ID = value
				}
			}
			if node.ID == "" {
				return nil, fmt.Errorf("元素缺少 id 属性: %s", node.Tag)
			}
			if _, exists := ids[node.ID]; exists {
				return nil, fmt.Errorf("元素 ID 已存在: %s", node.ID)
			}
			ids[node.ID] = struct{}{}

			if len(stack) == 0 {
				root = node
			} else {
				parent := stack[len(stack)-1]
				if strings.TrimSpace(parent.Text) != "" {
					return nil, errors.New("该元素已有文本内容，不支持混合内容")
				}
				node.Parent = parent
				parent.Children = append(parent.Children, node)
			}
			stack = append(stack, node)
		case xml.EndElement:
			if len(stack) == 0 {
				return nil, errors.New("XML 结构不匹配")
			}
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) == 0 {
				continue
			}
			current := stack[len(stack)-1]
			data := string([]byte(tok))
			if strings.TrimSpace(data) == "" {
				continue
			}
			if len(current.Children) > 0 {
				return nil, errors.New("该元素已有子元素，不支持混合内容")
			}
			if strings.TrimSpace(current.Text) == "" {
				current.Text = strings.TrimSpace(data)
			} else {
				current.Text += " " + strings.TrimSpace(data)
			}
		case xml.Comment:
			continue
		default:
			continue
		}
	}

	if root == nil {
		return nil, errors.New("未找到根元素")
	}
	return root, nil
}
