package logging

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"softwaredesign/src/events"
)

const timeLayout = "20060102 15:04:05"

// Manager coordinates file-based logging as an observer.
type Manager struct {
	mu             sync.Mutex
	enabled        map[string]bool
	sessionStarted map[string]bool
}

// NewManager builds a Manager.
func NewManager() *Manager {
	return &Manager{
		enabled:        map[string]bool{},
		sessionStarted: map[string]bool{},
	}
}

// Handle consumes command events for logging.
func (m *Manager) Handle(evt events.Event) {
	if evt.Type != events.EventCommandExecuted || evt.File == "" {
		return
	}
	m.mu.Lock()
	enabled := m.enabled[evt.File]
	m.mu.Unlock()
	if !enabled {
		return
	}
	if err := m.append(evt.File, fmt.Sprintf("%s %s", evt.Timestamp.Format(timeLayout), evt.Raw)); err != nil {
		fmt.Printf("[log warning] %v\n", err)
	}
}

// Enable activates logging for a file.
func (m *Manager) Enable(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled[abs] = true
	if !m.sessionStarted[abs] {
		if err := m.append(abs, fmt.Sprintf("session start at %s", time.Now().Format(timeLayout))); err != nil {
			fmt.Printf("[log warning] %v\n", err)
		} else {
			m.sessionStarted[abs] = true
		}
	}
	return nil
}

// Disable turns off logging for a file.
func (m *Manager) Disable(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.enabled, abs)
	return nil
}

// Enabled returns whether logging is active for a path.
func (m *Manager) Enabled(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.enabled[abs]
}

// Restore hydrates manager state from saved entries.
func (m *Manager) Restore(paths []string) {
	for _, p := range paths {
		_ = m.Enable(p)
	}
}

// ActivePaths lists currently enabled files.
func (m *Manager) ActivePaths() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, 0, len(m.enabled))
	for path := range m.enabled {
		result = append(result, path)
	}
	sort.Strings(result)
	return result
}

// LogFilePath resolves the log file path for a given file.
func LogFilePath(source string) (string, error) {
	abs, err := filepath.Abs(source)
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(abs)
	name := filepath.Base(abs)
	return filepath.Join(dir, fmt.Sprintf(".%s.log", name)), nil
}

// Show prints the log contents for a file.
func (m *Manager) Show(path string) (string, error) {
	logPath, err := LogFilePath(path)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (m *Manager) append(sourcePath, line string) error {
	logPath, err := LogFilePath(sourcePath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	writer := bufio.NewWriter(f)
	if _, err := writer.WriteString(strings.TrimSpace(line) + "\n"); err != nil {
		return err
	}
	return writer.Flush()
}
