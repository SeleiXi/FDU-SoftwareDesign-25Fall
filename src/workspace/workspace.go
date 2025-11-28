package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"softwaredesign/src/editor"
	"softwaredesign/src/events"
	"softwaredesign/src/fs"
	"softwaredesign/src/logging"
)

// SaveDecider asks user whether to save modifications.
type SaveDecider interface {
	ConfirmSave(path string) (bool, error)
}

// Info describes an open editor.
type Info struct {
	Path     string
	Name     string
	Modified bool
	Active   bool
}

// Workspace coordinates editors, persistence, and observers.
type Workspace struct {
	baseDir string
	editors map[string]*editor.TextEditor
	active  string
	history []string

	bus     *events.Bus
	keeper  *StateKeeper
	logger  *logging.Manager
	decider SaveDecider
}

// NewWorkspace builds a workspace.
func NewWorkspace(baseDir string, bus *events.Bus, keeper *StateKeeper, logger *logging.Manager, decider SaveDecider) *Workspace {
	return &Workspace{
		baseDir: baseDir,
		editors: map[string]*editor.TextEditor{},
		bus:     bus,
		keeper:  keeper,
		logger:  logger,
		decider: decider,
	}
}

// SetDecider overrides the save decider.
func (w *Workspace) SetDecider(decider SaveDecider) {
	w.decider = decider
}

// BaseDir exposes the root directory.
func (w *Workspace) BaseDir() string {
	return w.baseDir
}

// Load opens or activates a file.
func (w *Workspace) Load(path string) (*editor.TextEditor, error) {
	abs, err := w.resolvePath(path)
	if err != nil {
		return nil, err
	}
	if ed, ok := w.editors[abs]; ok {
		w.setActive(abs)
		return ed, nil
	}
	lines := []string{}
	modified := false
	info, err := os.Stat(abs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			modified = true
		} else {
			return nil, err
		}
	} else if info.IsDir() {
		return nil, fmt.Errorf("无法打开目录: %s", abs)
	} else {
		data, err := os.ReadFile(abs)
		if err != nil {
			return nil, err
		}
		lines = splitLines(string(data))
	}
	ed := editor.NewTextEditor(abs, lines, modified)
	w.editors[abs] = ed
	w.setActive(abs)
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "# log" {
		_ = w.logger.Enable(abs)
	}
	return ed, nil
}

// Init creates an unsaved buffer.
func (w *Workspace) Init(path string, withLog bool) (*editor.TextEditor, error) {
	abs, err := w.resolvePath(path)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(abs); err == nil {
		return nil, fmt.Errorf("文件已存在: %s", abs)
	}
	lines := []string{}
	if withLog {
		lines = []string{"# log"}
	}
	ed := editor.NewTextEditor(abs, lines, true)
	w.editors[abs] = ed
	w.setActive(abs)
	if withLog {
		_ = w.logger.Enable(abs)
	}
	return ed, nil
}

// Save writes the specified file (empty path means active).
func (w *Workspace) Save(path string) error {
	target := path
	if target == "" {
		target = w.active
	}
	if target == "" {
		return errors.New("没有活动文件")
	}
	abs, err := w.resolvePath(target)
	if err != nil {
		return err
	}
	ed, ok := w.editors[abs]
	if !ok {
		return fmt.Errorf("文件未打开: %s", target)
	}
	if err := w.saveEditor(ed); err != nil {
		return err
	}
	ed.SetModified(false)
	return nil
}

// SaveAll writes every open editor.
func (w *Workspace) SaveAll() error {
	for _, ed := range w.editors {
		if err := w.saveEditor(ed); err != nil {
			return err
		}
		ed.SetModified(false)
	}
	return nil
}

// Close removes an editor, prompting when necessary.
func (w *Workspace) Close(path string) error {
	target := path
	if target == "" {
		target = w.active
	}
	if target == "" {
		return errors.New("没有活动文件")
	}
	abs, err := w.resolvePath(target)
	if err != nil {
		return err
	}
	ed, ok := w.editors[abs]
	if !ok {
		return fmt.Errorf("文件未打开: %s", target)
	}
	if ed.IsModified() && w.decider != nil {
		save, err := w.decider.ConfirmSave(abs)
		if err != nil {
			return err
		}
		if save {
			if err := w.saveEditor(ed); err != nil {
				return err
			}
			ed.SetModified(false)
		}
	}
	delete(w.editors, abs)
	w.removeFromHistory(abs)
	if w.active == abs {
		w.active = ""
		if len(w.history) > 0 {
			w.active = w.history[0]
		}
	}
	return nil
}

// Edit switches the active editor.
func (w *Workspace) Edit(path string) error {
	abs, err := w.resolvePath(path)
	if err != nil {
		return err
	}
	if _, ok := w.editors[abs]; !ok {
		return fmt.Errorf("文件未打开: %s", path)
	}
	w.setActive(abs)
	return nil
}

// List returns info for editors.
func (w *Workspace) List() []Info {
	result := make([]Info, 0, len(w.editors))
	for path, ed := range w.editors {
		result = append(result, Info{
			Path:     path,
			Name:     ed.Name(),
			Modified: ed.IsModified(),
			Active:   path == w.active,
		})
	}
	return result
}

// DirTree prints a directory tree.
func (w *Workspace) DirTree(path string) (string, error) {
	target := path
	if target == "" {
		target = w.baseDir
	}
	return fs.Tree(target)
}

// Undo reverts an edit.
func (w *Workspace) Undo() error {
	ed, err := w.ActiveEditor()
	if err != nil {
		return err
	}
	return ed.Undo()
}

// Redo reapplies an edit.
func (w *Workspace) Redo() error {
	ed, err := w.ActiveEditor()
	if err != nil {
		return err
	}
	return ed.Redo()
}

// ActiveEditor returns the current editor.
func (w *Workspace) ActiveEditor() (*editor.TextEditor, error) {
	if w.active == "" {
		return nil, errors.New("没有活动文件")
	}
	ed, ok := w.editors[w.active]
	if !ok {
		return nil, errors.New("活动文件不存在")
	}
	return ed, nil
}

// PublishCommand notifies observers about a command.
func (w *Workspace) PublishCommand(name, raw, file string) {
	if w.bus == nil {
		return
	}
	w.bus.Publish(events.Event{
		Type:      events.EventCommandExecuted,
		Timestamp: time.Now(),
		Command:   name,
		Raw:       raw,
		File:      file,
	})
}

// Persist saves workspace metadata.
func (w *Workspace) Persist() error {
	state := WorkspaceState{
		Active: w.active,
	}
	for path, ed := range w.editors {
		state.Editors = append(state.Editors, EditorState{
			Path:     path,
			Modified: ed.IsModified(),
		})
	}
	state.Logging = w.logger.ActivePaths()
	return w.keeper.Save(state)
}

// Restore hydrates workspace from disk.
func (w *Workspace) Restore() error {
	state, err := w.keeper.Load()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, entry := range state.Editors {
		if _, err := os.Stat(entry.Path); err != nil {
			continue
		}
		ed, err := w.Load(entry.Path)
		if err != nil {
			continue
		}
		ed.SetModified(entry.Modified)
	}
	if state.Active != "" {
		if _, ok := w.editors[state.Active]; ok {
			w.setActive(state.Active)
		}
	}
	w.logger.Restore(state.Logging)
	return nil
}

func (w *Workspace) resolvePath(path string) (string, error) {
	if path == "" {
		return "", errors.New("路径不能为空")
	}
	expanded := path
	if !filepath.IsAbs(path) {
		expanded = filepath.Join(w.baseDir, path)
	}
	return filepath.Abs(expanded)
}

func (w *Workspace) setActive(path string) {
	w.active = path
	w.touchHistory(path)
}

func (w *Workspace) touchHistory(path string) {
	w.removeFromHistory(path)
	w.history = append([]string{path}, w.history...)
}

func (w *Workspace) removeFromHistory(path string) {
	next := w.history[:0]
	for _, item := range w.history {
		if item != path {
			next = append(next, item)
		}
	}
	w.history = next
}

func (w *Workspace) saveEditor(ed *editor.TextEditor) error {
	if err := os.MkdirAll(filepath.Dir(ed.Path()), 0o755); err != nil {
		return err
	}
	content := strings.Join(ed.Lines(), "\n")
	return os.WriteFile(ed.Path(), []byte(content), 0o644)
}

func splitLines(data string) []string {
	if data == "" {
		return []string{}
	}
	data = strings.ReplaceAll(data, "\r\n", "\n")
	lines := strings.Split(data, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
