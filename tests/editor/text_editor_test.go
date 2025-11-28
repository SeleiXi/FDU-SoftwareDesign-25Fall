package editor_test

import (
	"testing"

	"softwaredesign/src/editor"
)

func TestInsertDeleteReplace(t *testing.T) {
	ed := editor.NewTextEditor("test.txt", []string{"hello"}, false)
	if err := ed.Insert(1, 6, " world"); err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	lines, err := ed.Show(1, 1)
	if err != nil {
		t.Fatalf("show failed: %v", err)
	}
	if lines[0] != "hello world" {
		t.Fatalf("unexpected content: %v", lines[0])
	}

	if err := ed.Delete(1, 7, 5); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	lines, _ = ed.Show(1, 1)
	if lines[0] != "hello " {
		t.Fatalf("unexpected delete result: %v", lines[0])
	}

	if err := ed.Replace(1, 1, 5, "bye"); err != nil {
		t.Fatalf("replace failed: %v", err)
	}
	lines, _ = ed.Show(1, 1)
	if lines[0] != "bye " {
		t.Fatalf("unexpected replace result: %v", lines[0])
	}
}

func TestUndoRedo(t *testing.T) {
	ed := editor.NewTextEditor("demo.txt", []string{"first"}, false)
	if err := ed.Append("second"); err != nil {
		t.Fatalf("append failed: %v", err)
	}
	if len(ed.Lines()) != 2 {
		t.Fatalf("append not applied")
	}
	if err := ed.Undo(); err != nil {
		t.Fatalf("undo failed: %v", err)
	}
	if len(ed.Lines()) != 1 {
		t.Fatalf("undo did not revert")
	}
	if err := ed.Redo(); err != nil {
		t.Fatalf("redo failed: %v", err)
	}
	if len(ed.Lines()) != 2 {
		t.Fatalf("redo did not reapply")
	}
}

func TestAppend(t *testing.T) {
	ed := editor.NewTextEditor("test.txt", []string{}, false)
	if err := ed.Append("line1"); err != nil {
		t.Fatalf("append failed: %v", err)
	}
	if err := ed.Append("line2"); err != nil {
		t.Fatalf("append failed: %v", err)
	}
	lines := ed.Lines()
	if len(lines) != 2 || lines[0] != "line1" || lines[1] != "line2" {
		t.Fatalf("unexpected lines: %v", lines)
	}
}

func TestInsertWithNewline(t *testing.T) {
	ed := editor.NewTextEditor("test.txt", []string{"abc"}, false)
	if err := ed.Insert(1, 2, "x\ny"); err != nil {
		t.Fatalf("insert with newline failed: %v", err)
	}
	lines := ed.Lines()
	if len(lines) != 2 || lines[0] != "ax" || lines[1] != "ybc" {
		t.Fatalf("unexpected lines after newline insert: %v", lines)
	}
}

func TestShowRange(t *testing.T) {
	ed := editor.NewTextEditor("test.txt", []string{"line1", "line2", "line3"}, false)
	lines, err := ed.Show(1, 2)
	if err != nil {
		t.Fatalf("show range failed: %v", err)
	}
	if len(lines) != 2 || lines[0] != "line1" || lines[1] != "line2" {
		t.Fatalf("unexpected show range: %v", lines)
	}
}
