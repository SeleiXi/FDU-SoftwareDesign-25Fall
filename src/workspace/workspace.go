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
	"softwaredesign/src/spellcheck"
	"softwaredesign/src/statistics"
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
	Duration time.Duration
}

// Workspace coordinates editors, persistence, and observers.
type Workspace struct {
	baseDir string
	editors map[string]editor.Editor
	active  string
	history []string

	bus     *events.Bus
	keeper  *StateKeeper
	logger  *logging.Manager
	decider SaveDecider
	stats   *statistics.Tracker
	speller *spellcheck.Service
}

// NewWorkspace builds a workspace.
func NewWorkspace(baseDir string, bus *events.Bus, keeper *StateKeeper, logger *logging.Manager, decider SaveDecider) *Workspace {
	return &Workspace{
		baseDir: baseDir,
		editors: map[string]editor.Editor{},
		bus:     bus,
		keeper:  keeper,
		logger:  logger,
		decider: decider,
		stats:   statistics.NewTracker(),
		speller: spellcheck.NewService(spellcheck.NewThirdPartyAdapter()),
	}
}

// SetDecider overrides the save decider.
func (w *Workspace) SetDecider(decider SaveDecider) {
	w.decider = decider
}

// SetSpellService overrides the spell check service (used in tests).
func (w *Workspace) SetSpellService(service *spellcheck.Service) {
	w.speller = service
}

// SetClock overrides the tracker clock for deterministic testing.
func (w *Workspace) SetClock(clock statistics.Clock) {
	w.stats.WithClock(clock)
}

// BaseDir exposes the root directory.
func (w *Workspace) BaseDir() string {
	return w.baseDir
}

// Load opens or activates a file.
func (w *Workspace) Load(path string) (editor.Editor, error) {
	abs, err := w.resolvePath(path)
	if err != nil {
		return nil, err
	}
	if ed, ok := w.editors[abs]; ok {
		w.setActive(abs)
		return ed, nil
	}
	ext := strings.ToLower(filepath.Ext(abs))
	var ed editor.Editor
	switch ext {
	case ".xml":
		info, statErr := os.Stat(abs)
		if statErr != nil {
			if errors.Is(statErr, os.ErrNotExist) {
				return nil, fmt.Errorf("XML 文件不存在: %s", abs)
			}
			return nil, statErr
		}
		if info.IsDir() {
			return nil, fmt.Errorf("无法打开目录: %s", abs)
		}
		data, readErr := os.ReadFile(abs)
		if readErr != nil {
			return nil, readErr
		}
		parsed, parseErr := editor.ParseXMLEditor(abs, data)
		if parseErr != nil {
			return nil, parseErr
		}
		ed = parsed
	default:
		lines := []string{}
		modified := false
		info, statErr := os.Stat(abs)
		if statErr != nil {
			if errors.Is(statErr, os.ErrNotExist) {
				modified = true
			} else {
				return nil, statErr
			}
		} else if info.IsDir() {
			return nil, fmt.Errorf("无法打开目录: %s", abs)
		} else {
			data, readErr := os.ReadFile(abs)
			if readErr != nil {
				return nil, readErr
			}
			lines = splitLines(string(data))
		}
		ed = editor.NewTextEditor(abs, lines, modified)
	}
	w.editors[abs] = ed
	w.setActive(abs)
	w.applyAutoLog(ed)
	return ed, nil
}

// Init creates an unsaved buffer.
func (w *Workspace) Init(kind, path string, withLog bool) (editor.Editor, error) {
	abs, err := w.resolvePath(path)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(abs); err == nil {
		return nil, fmt.Errorf("文件已存在: %s", abs)
	}
	var ed editor.Editor
	switch strings.ToLower(kind) {
	case "text":
		lines := []string{}
		if withLog {
			lines = []string{"# log"}
		}
		ed = editor.NewTextEditor(abs, lines, true)
	case "xml":
		root := editor.NewDefaultXMLDocument(withLog)
		ed = editor.NewXMLEditor(abs, root, true)
	default:
		return nil, fmt.Errorf("未知的编辑器类型: %s", kind)
	}
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
		save, decErr := w.decider.ConfirmSave(abs)
		if decErr != nil {
			return decErr
		}
		if save {
			if err := w.saveEditor(ed); err != nil {
				return err
			}
			ed.SetModified(false)
		}
	}
	w.stats.Close(abs)
	delete(w.editors, abs)
	w.removeFromHistory(abs)
	next := ""
	if w.active == abs {
		if len(w.history) > 0 {
			next = w.history[0]
		}
	} else {
		next = w.active
	}
	w.setActive(next)
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
			Duration: w.stats.Duration(path),
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
func (w *Workspace) ActiveEditor() (editor.Editor, error) {
	if w.active == "" {
		return nil, errors.New("没有活动文件")
	}
	ed, ok := w.editors[w.active]
	if !ok {
		return nil, errors.New("活动文件不存在")
	}
	return ed, nil
}

// EditorByPath returns an opened editor by path.
func (w *Workspace) EditorByPath(path string) (editor.Editor, error) {
	abs, err := w.resolvePath(path)
	if err != nil {
		return nil, err
	}
	ed, ok := w.editors[abs]
	if !ok {
		return nil, fmt.Errorf("文件未打开: %s", path)
	}
	return ed, nil
}

// SpellCheck runs the configured spell checker on the target file.
func (w *Workspace) SpellCheck(path string) (string, error) {
	if w.speller == nil {
		return "", errors.New("未配置拼写检查器")
	}
	target := path
	if target == "" {
		target = w.active
	}
	if target == "" {
		return "", errors.New("没有活动文件")
	}
	abs, err := w.resolvePath(target)
	if err != nil {
		return "", err
	}
	ed, ok := w.editors[abs]
	if !ok {
		return "", fmt.Errorf("文件未打开: %s", target)
	}
	switch doc := ed.(type) {
	case editor.TextDocument:
		issues := w.speller.CheckLines(doc.Lines())
		return formatTextIssues(issues), nil
	case editor.XMLTreeEditor:
		raw := doc.TextNodes()
		entries := make([]spellcheck.XMLText, len(raw))
		for i, entry := range raw {
			entries[i] = spellcheck.XMLText{ElementID: entry.ElementID, Text: entry.Text}
		}
		issues := w.speller.CheckXMLText(entries)
		return formatXMLIssues(issues), nil
	default:
		return "", errors.New("当前文件不支持拼写检查")
	}
}

// PublishCommand notifies observers about a command.
func (w *Workspace) PublishCommand(name, raw, file string) {
	if w.bus == nil {
		return
	}
	metadata := map[string]string{}
	if w.active != "" {
		metadata["active"] = w.active
	}
	w.bus.Publish(events.Event{
		Type:      events.EventCommandExecuted,
		Timestamp: time.Now(),
		Command:   name,
		Raw:       raw,
		File:      file,
		Metadata:  metadata,
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
	w.stats.StopAll()
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
		if _, statErr := os.Stat(entry.Path); statErr != nil {
			continue
		}
		ed, loadErr := w.Load(entry.Path)
		if loadErr != nil {
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
	prev := w.active
	if prev == path {
		return
	}
	w.active = path
	w.stats.Switch(prev, path)
	if path != "" {
		w.touchHistory(path)
	}
}

func (w *Workspace) touchHistory(path string) {
	if path == "" {
		return
	}
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

func (w *Workspace) saveEditor(ed editor.Editor) error {
	if err := os.MkdirAll(filepath.Dir(ed.Path()), 0o755); err != nil {
		return err
	}
	content, err := ed.Content()
	if err != nil {
		return err
	}
	return os.WriteFile(ed.Path(), []byte(content), 0o644)
}

func (w *Workspace) applyAutoLog(ed editor.Editor) {
	switch doc := ed.(type) {
	case editor.TextDocument:
		lines := doc.Lines()
		if len(lines) > 0 && strings.TrimSpace(lines[0]) == "# log" {
			_ = w.logger.Enable(ed.Path())
		}
	case editor.XMLTreeEditor:
		attrs := doc.RootAttributes()
		if strings.EqualFold(attrs["log"], "true") {
			_ = w.logger.Enable(ed.Path())
		}
	}
}

func formatTextIssues(issues []spellcheck.TextIssue) string {
	var builder strings.Builder
	builder.WriteString("拼写检查结果:\n")
	if len(issues) == 0 {
		builder.WriteString("未发现拼写错误")
		return builder.String()
	}
	for i, issue := range issues {
		suggestions := "无"
		if len(issue.Suggestions) > 0 {
			suggestions = strings.Join(issue.Suggestions, ", ")
		}
		builder.WriteString(fmt.Sprintf("第%d行，第%d列: \"%s\" -> 建议: %s", issue.Line, issue.Column, issue.Word, suggestions))
		if i != len(issues)-1 {
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func formatXMLIssues(issues []spellcheck.XMLIssue) string {
	var builder strings.Builder
	builder.WriteString("拼写检查结果:\n")
	if len(issues) == 0 {
		builder.WriteString("未发现拼写错误")
		return builder.String()
	}
	for i, issue := range issues {
		suggestions := "无"
		if len(issue.Suggestions) > 0 {
			suggestions = strings.Join(issue.Suggestions, ", ")
		}
		builder.WriteString(fmt.Sprintf("元素 %s: \"%s\" -> 建议: %s", issue.ElementID, issue.Word, suggestions))
		if i != len(issues)-1 {
			builder.WriteString("\n")
		}
	}
	return builder.String()
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
