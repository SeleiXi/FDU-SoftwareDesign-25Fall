package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const stateFile = ".editor_workspace"

// EditorState stores lightweight editor metadata.
type EditorState struct {
	Path     string `json:"path"`
	Modified bool   `json:"modified"`
}

// WorkspaceState captures persisted workspace info.
type WorkspaceState struct {
	Editors []EditorState `json:"editors"`
	Active  string        `json:"active"`
	Logging []string      `json:"logging"`
}

// StateKeeper reads/writes workspace state.
type StateKeeper struct {
	path string
}

// NewStateKeeper builds a state keeper rooted at baseDir.
func NewStateKeeper(baseDir string) *StateKeeper {
	return &StateKeeper{
		path: filepath.Join(baseDir, stateFile),
	}
}

// Save persists workspace state to disk.
func (s *StateKeeper) Save(state WorkspaceState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}

// Load restores workspace state if present.
func (s *StateKeeper) Load() (WorkspaceState, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return WorkspaceState{}, err
	}
	var state WorkspaceState
	if err := json.Unmarshal(data, &state); err != nil {
		return WorkspaceState{}, err
	}
	return state, nil
}
