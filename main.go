package main

import (
	"fmt"
	"os"

	"softwaredesign/src/cli"
	"softwaredesign/src/events"
	"softwaredesign/src/logging"
	"softwaredesign/src/workspace"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("无法获取工作目录: %v\n", err)
		return
	}
	console := cli.NewConsole(os.Stdin, os.Stdout)
	bus := events.NewBus()
	logger := logging.NewManager()
	bus.Subscribe(logger)
	keeper := workspace.NewStateKeeper(wd)
	ws := workspace.NewWorkspace(wd, bus, keeper, logger, console)
	if err := ws.Restore(); err != nil {
		fmt.Printf("恢复工作区失败: %v\n", err)
	}
	dispatcher := cli.NewDispatcher(ws, console, logger)
	dispatcher.Run()
}
