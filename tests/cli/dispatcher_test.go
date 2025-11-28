package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"softwaredesign/src/cli"
	"softwaredesign/src/events"
	"softwaredesign/src/logging"
	"softwaredesign/src/workspace"
)

func TestDispatcherLoadCommand(t *testing.T) {
	dir := t.TempDir()
	bus := events.NewBus()
	logger := logging.NewManager()
	keeper := workspace.NewStateKeeper(dir)
	output := bytes.NewBuffer(nil)
	console := cli.NewConsole(bytes.NewBufferString(""), output)
	ws := workspace.NewWorkspace(dir, bus, keeper, logger, console)
	dispatcher := cli.NewDispatcher(ws, console, logger)

	file := dir + "/test.txt"
	err := dispatcher.Execute("load " + file)
	if err != nil {
		t.Fatalf("load command failed: %v", err)
	}
	active, err := ws.ActiveEditor()
	if err != nil || active == nil {
		t.Fatalf("active editor should exist after load")
	}
}

func TestDispatcherEditorList(t *testing.T) {
	dir := t.TempDir()
	bus := events.NewBus()
	logger := logging.NewManager()
	keeper := workspace.NewStateKeeper(dir)
	output := bytes.NewBuffer(nil)
	console := cli.NewConsole(bytes.NewBufferString(""), output)
	ws := workspace.NewWorkspace(dir, bus, keeper, logger, console)
	dispatcher := cli.NewDispatcher(ws, console, logger)

	file := dir + "/test.txt"
	dispatcher.Execute("load " + file)
	err := dispatcher.Execute("editor-list")
	if err != nil {
		t.Fatalf("editor-list command failed: %v", err)
	}
	outputStr := output.String()
	if !strings.Contains(outputStr, "test.txt") {
		t.Fatalf("editor-list should contain test.txt, output: %s", outputStr)
	}
}
