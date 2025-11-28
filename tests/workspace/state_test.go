package workspace_test

import (
	"testing"

	"softwaredesign/src/workspace"
)

func TestStateKeeperSaveLoad(t *testing.T) {
	dir := t.TempDir()
	keeper := workspace.NewStateKeeper(dir)
	state := workspace.WorkspaceState{
		Editors: []workspace.EditorState{
			{Path: dir + "/a.txt", Modified: true},
		},
		Active:  dir + "/a.txt",
		Logging: []string{dir + "/a.txt"},
	}
	if err := keeper.Save(state); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	loaded, err := keeper.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(loaded.Editors) != 1 || loaded.Editors[0].Path != state.Editors[0].Path {
		t.Fatalf("loaded state mismatch: %+v", loaded)
	}
	if loaded.Active != state.Active {
		t.Fatalf("active mismatch")
	}
	if len(loaded.Logging) != 1 || loaded.Logging[0] != state.Logging[0] {
		t.Fatalf("logging mismatch")
	}
}

func TestStateKeeperEmptyState(t *testing.T) {
	dir := t.TempDir()
	keeper := workspace.NewStateKeeper(dir)
	state := workspace.WorkspaceState{}
	if err := keeper.Save(state); err != nil {
		t.Fatalf("save empty state failed: %v", err)
	}
	loaded, err := keeper.Load()
	if err != nil {
		t.Fatalf("load empty state failed: %v", err)
	}
	if len(loaded.Editors) != 0 || loaded.Active != "" {
		t.Fatalf("empty state should be empty: %+v", loaded)
	}
}

