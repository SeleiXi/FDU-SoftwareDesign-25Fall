package editor

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// TextEditor manages in-memory text lines with undo/redo support.
type TextEditor struct {
	path      string
	lines     []string
	modified  bool
	undoStack []*editCommand
	redoStack []*editCommand
}

// NewTextEditor constructs an editor for the provided path.
func NewTextEditor(path string, lines []string, modified bool) *TextEditor {
	copied := cloneLines(lines)
	return &TextEditor{
		path:     path,
		lines:    copied,
		modified: modified,
	}
}

// Path returns the backing file path.
func (e *TextEditor) Path() string {
	return e.path
}

// Name returns the file name for display.
func (e *TextEditor) Name() string {
	return filepath.Base(e.path)
}

// Type returns the editor kind.
func (e *TextEditor) Type() Type {
	return TypeText
}

// Lines returns a copy of editor lines.
func (e *TextEditor) Lines() []string {
	return cloneLines(e.lines)
}

// SetLines replaces editor content.
func (e *TextEditor) SetLines(lines []string) {
	e.lines = cloneLines(lines)
}

// IsModified reports whether editor has unsaved changes.
func (e *TextEditor) IsModified() bool {
	return e.modified
}

// SetModified forces modified flag.
func (e *TextEditor) SetModified(value bool) {
	e.modified = value
}

// Append adds a new line at the end.
func (e *TextEditor) Append(text string) error {
	return e.execute("append", func() error {
		for _, line := range splitWithKeep(text) {
			e.lines = append(e.lines, line)
		}
		return nil
	})
}

// Insert adds text at the specified 1-based line and column.
func (e *TextEditor) Insert(line, col int, text string) error {
	return e.execute("insert", func() error {
		return e.insertSpan(line, col, text)
	})
}

// Delete removes len characters starting at line:col.
func (e *TextEditor) Delete(line, col, length int) error {
	return e.execute("delete", func() error {
		return e.deleteSpan(line, col, length)
	})
}

// Replace swaps len characters with text starting at line:col.
func (e *TextEditor) Replace(line, col, length int, text string) error {
	return e.execute("replace", func() error {
		if err := e.deleteSpan(line, col, length); err != nil {
			return err
		}
		return e.insertSpan(line, col, text)
	})
}

// Show returns lines within the inclusive range (1-based).
func (e *TextEditor) Show(start, end int) ([]string, error) {
	if len(e.lines) == 0 {
		return []string{}, nil
	}
	if start < 1 || start > len(e.lines) {
		return nil, fmt.Errorf("起始行越界: %d", start)
	}
	if end == 0 {
		end = len(e.lines)
	}
	if end < start || end > len(e.lines) {
		return nil, fmt.Errorf("结束行越界: %d", end)
	}
	view := make([]string, 0, end-start+1)
	for i := start - 1; i < end; i++ {
		view = append(view, e.lines[i])
	}
	return view, nil
}

// Undo reverts the last command.
func (e *TextEditor) Undo() error {
	if len(e.undoStack) == 0 {
		return errors.New("没有可撤销的操作")
	}
	last := e.undoStack[len(e.undoStack)-1]
	e.undoStack = e.undoStack[:len(e.undoStack)-1]
	if err := last.undo(e); err != nil {
		return err
	}
	e.redoStack = append(e.redoStack, last)
	e.modified = true
	return nil
}

// Redo reapplies the last undone command.
func (e *TextEditor) Redo() error {
	if len(e.redoStack) == 0 {
		return errors.New("没有可重做的操作")
	}
	last := e.redoStack[len(e.redoStack)-1]
	e.redoStack = e.redoStack[:len(e.redoStack)-1]
	if err := last.redo(e); err != nil {
		return err
	}
	e.undoStack = append(e.undoStack, last)
	e.modified = true
	return nil
}

func (e *TextEditor) execute(desc string, mutate func() error) error {
	before := cloneLines(e.lines)
	if err := mutate(); err != nil {
		return err
	}
	after := cloneLines(e.lines)
	cmd := &editCommand{description: desc, before: before, after: after}
	e.undoStack = append(e.undoStack, cmd)
	e.redoStack = nil
	e.modified = true
	return nil
}

func (e *TextEditor) ensureLinePosition(line, col int, allowEOF bool) error {
	if len(e.lines) == 0 {
		if allowEOF && line == 1 && col == 1 {
			return nil
		}
		return errors.New("空文件只能在1:1位置插入")
	}
	if line < 1 || line > len(e.lines) {
		return fmt.Errorf("行号越界: %d", line)
	}
	lineLen := utf8.RuneCountInString(e.lines[line-1])
	maxCol := lineLen + 1
	if !allowEOF {
		maxCol = lineLen
	}
	if col < 1 || col > maxCol {
		return fmt.Errorf("列号越界: %d", col)
	}
	return nil
}

type editCommand struct {
	description string
	before      []string
	after       []string
}

func (c *editCommand) undo(e *TextEditor) error {
	e.lines = cloneLines(c.before)
	return nil
}

func (c *editCommand) redo(e *TextEditor) error {
	e.lines = cloneLines(c.after)
	return nil
}

func cloneLines(src []string) []string {
	dst := make([]string, len(src))
	copy(dst, src)
	return dst
}

// Content returns the serialized text data.
func (e *TextEditor) Content() (string, error) {
	return strings.Join(e.lines, "\n"), nil
}

func splitWithKeep(text string) []string {
	if text == "" {
		return []string{""}
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	chunks := strings.Split(text, "\n")
	return chunks
}

func splitLineAtColumn(line string, col int) (string, string, error) {
	runes := []rune(line)
	if col < 1 || col > len(runes)+1 {
		return "", "", fmt.Errorf("列号越界: %d", col)
	}
	return string(runes[:col-1]), string(runes[col-1:]), nil
}

func (e *TextEditor) insertSpan(line, col int, text string) error {
	if err := e.ensureLinePosition(line, col, true); err != nil {
		return err
	}
	if len(e.lines) == 0 {
		e.lines = []string{""}
	}
	lineIdx := line - 1
	left, right, err := splitLineAtColumn(e.lines[lineIdx], col)
	if err != nil {
		return err
	}
	newLines := splitWithKeep(text)
	if len(newLines) == 1 {
		e.lines[lineIdx] = left + newLines[0] + right
		return nil
	}
	composed := make([]string, 0, len(e.lines)+len(newLines)-1)
	composed = append(composed, e.lines[:lineIdx]...)
	composed = append(composed, left+newLines[0])
	if len(newLines) > 2 {
		composed = append(composed, newLines[1:len(newLines)-1]...)
	}
	composed = append(composed, newLines[len(newLines)-1]+right)
	composed = append(composed, e.lines[lineIdx+1:]...)
	e.lines = composed
	return nil
}

func (e *TextEditor) deleteSpan(line, col, length int) error {
	if err := e.ensureLinePosition(line, col, false); err != nil {
		return err
	}
	if length < 1 {
		return errors.New("删除长度必须大于0")
	}
	lineIdx := line - 1
	runes := []rune(e.lines[lineIdx])
	if col-1+length > len(runes) {
		return errors.New("删除长度超出行尾")
	}
	newLine := string(runes[:col-1]) + string(runes[col-1+length:])
	e.lines[lineIdx] = newLine
	return nil
}
