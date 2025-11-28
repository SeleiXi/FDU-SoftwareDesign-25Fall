package editor_test

import (
	"strings"
	"testing"

	"softwaredesign/src/editor"
)

func TestXMLEditorCommands(t *testing.T) {
	root := editor.NewDefaultXMLDocument(false)
	ed := editor.NewXMLEditor("test.xml", root, true)

	if err := ed.AppendChild("book", "book1", "root", nil); err != nil {
		t.Fatalf("append-child failed: %v", err)
	}
	text := "Everyday Italian"
	if err := ed.AppendChild("title", "title1", "book1", &text); err != nil {
		t.Fatalf("append-child title failed: %v", err)
	}
	if err := ed.EditText("title1", "Everyday Italian Updated"); err != nil {
		t.Fatalf("edit-text failed: %v", err)
	}
	if err := ed.EditID("title1", "title-main"); err != nil {
		t.Fatalf("edit-id failed: %v", err)
	}
	tree := ed.TreeString()
	if !strings.Contains(tree, "book [id=\"book1\"]") {
		t.Fatalf("tree missing book node: %s", tree)
	}
	if !strings.Contains(tree, "\"Everyday Italian Updated\"") {
		t.Fatalf("tree missing updated text: %s", tree)
	}
	if err := ed.DeleteElement("title-main"); err != nil {
		t.Fatalf("delete-element failed: %v", err)
	}
	tree = ed.TreeString()
	if strings.Contains(tree, "title") {
		t.Fatalf("title should be removed: %s", tree)
	}
}

func TestXMLEditorUndoRedo(t *testing.T) {
	root := editor.NewDefaultXMLDocument(false)
	ed := editor.NewXMLEditor("demo.xml", root, true)

	if err := ed.AppendChild("item", "item1", "root", nil); err != nil {
		t.Fatalf("append failed: %v", err)
	}
	if err := ed.Undo(); err != nil {
		t.Fatalf("undo failed: %v", err)
	}
	tree := ed.TreeString()
	if strings.Contains(tree, "item") {
		t.Fatalf("undo should remove child")
	}
	if err := ed.Redo(); err != nil {
		t.Fatalf("redo failed: %v", err)
	}
	tree = ed.TreeString()
	if !strings.Contains(tree, "item [id=\"item1\"]") {
		t.Fatalf("redo should restore child: %s", tree)
	}
}
