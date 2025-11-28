package logging_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"softwaredesign/src/events"
	"softwaredesign/src/logging"
)

func TestManagerLoggingFlow(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "demo.txt")
	if err := os.WriteFile(file, []byte("demo"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	mgr := logging.NewManager()
	if err := mgr.Enable(file); err != nil {
		t.Fatalf("enable failed: %v", err)
	}
	event := events.Event{
		Type:      events.EventCommandExecuted,
		Command:   "append",
		Raw:       "append \"text\"",
		File:      file,
		Timestamp: time.Now(),
	}
	mgr.Handle(event)
	content, err := mgr.Show(file)
	if err != nil {
		t.Fatalf("show failed: %v", err)
	}
	if !strings.Contains(content, "append \"text\"") {
		t.Fatalf("log missing command, content: %s", content)
	}
	if len(mgr.ActivePaths()) != 1 {
		t.Fatalf("expected one active path")
	}
}

func TestManagerEnableDisable(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.txt")
	os.WriteFile(file, []byte("test"), 0o644)

	mgr := logging.NewManager()
	if err := mgr.Enable(file); err != nil {
		t.Fatalf("enable failed: %v", err)
	}
	if len(mgr.ActivePaths()) != 1 {
		t.Fatalf("should have one active path")
	}

	if err := mgr.Disable(file); err != nil {
		t.Fatalf("disable failed: %v", err)
	}
	if len(mgr.ActivePaths()) != 0 {
		t.Fatalf("should have no active paths after disable")
	}
}

func TestManagerSessionStart(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.txt")
	os.WriteFile(file, []byte("test"), 0o644)

	mgr := logging.NewManager()
	if err := mgr.Enable(file); err != nil {
		t.Fatalf("enable failed: %v", err)
	}

	content, err := mgr.Show(file)
	if err != nil {
		t.Fatalf("show failed: %v", err)
	}
	if !strings.Contains(content, "session start at") {
		t.Fatalf("log should contain session start, content: %s", content)
	}
}

