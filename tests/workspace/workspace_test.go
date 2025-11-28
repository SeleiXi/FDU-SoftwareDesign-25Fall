package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"softwaredesign/src/events"
	"softwaredesign/src/logging"
	"softwaredesign/src/workspace"
)

func TestWorkspaceLoadSaveCycle(t *testing.T) {
	dir := t.TempDir()
	bus := events.NewBus()
	logger := logging.NewManager()
	keeper := workspace.NewStateKeeper(dir)
	ws := workspace.NewWorkspace(dir, bus, keeper, logger, nil)

	file := filepath.Join(dir, "sample.txt")
	ed, err := ws.Load(file)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if !ed.IsModified() {
		t.Fatalf("new file should be marked modified")
	}
	if err := ed.Append("hello"); err != nil {
		t.Fatalf("append failed: %v", err)
	}
	if err := ws.Save(""); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read file failed: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("unexpected file content: %s", string(data))
	}
	if err := ws.Persist(); err != nil {
		t.Fatalf("persist failed: %v", err)
	}
	state, err := keeper.Load()
	if err != nil {
		t.Fatalf("load state failed: %v", err)
	}
	if len(state.Editors) != 1 || state.Editors[0].Path != file {
		t.Fatalf("state missing editor info: %+v", state)
	}
}

func TestWorkspaceMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	bus := events.NewBus()
	logger := logging.NewManager()
	keeper := workspace.NewStateKeeper(dir)
	ws := workspace.NewWorkspace(dir, bus, keeper, logger, nil)

	file1 := filepath.Join(dir, "file1.txt")
	file2 := filepath.Join(dir, "file2.txt")

	ed1, err := ws.Load(file1)
	if err != nil {
		t.Fatalf("load file1 failed: %v", err)
	}
	ed1.Append("content1")

	ed2, err := ws.Load(file2)
	if err != nil {
		t.Fatalf("load file2 failed: %v", err)
	}
	ed2.Append("content2")

	active2, _ := ws.ActiveEditor()
	if active2 != ed2 {
		t.Fatalf("active editor should be file2")
	}

	if err := ws.Edit(file1); err != nil {
		t.Fatalf("edit file1 failed: %v", err)
	}
	active1, _ := ws.ActiveEditor()
	if active1 != ed1 {
		t.Fatalf("active editor should be file1 after edit")
	}
}

func TestWorkspaceClose(t *testing.T) {
	dir := t.TempDir()
	bus := events.NewBus()
	logger := logging.NewManager()
	keeper := workspace.NewStateKeeper(dir)
	ws := workspace.NewWorkspace(dir, bus, keeper, logger, nil)

	file := filepath.Join(dir, "test.txt")
	ws.Load(file)
	active, _ := ws.ActiveEditor()
	if active == nil {
		t.Fatalf("active editor should exist")
	}

	if err := ws.Close(""); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	activeAfter, _ := ws.ActiveEditor()
	if activeAfter != nil {
		t.Fatalf("active editor should be nil after close")
	}
}

