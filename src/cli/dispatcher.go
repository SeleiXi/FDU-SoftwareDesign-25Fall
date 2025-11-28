package cli

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"softwaredesign/src/logging"
	"softwaredesign/src/workspace"
)

// Dispatcher interprets user commands.
type Dispatcher struct {
	ws      *workspace.Workspace
	console *Console
	logger  *logging.Manager
}

// NewDispatcher constructs a dispatcher.
func NewDispatcher(ws *workspace.Workspace, console *Console, logger *logging.Manager) *Dispatcher {
	return &Dispatcher{
		ws:      ws,
		console: console,
		logger:  logger,
	}
}

// Run processes interactive commands until exit.
func (d *Dispatcher) Run() {
	for {
		d.console.Print("> ")
		line, err := d.console.ReadLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				_ = d.handleExit()
				return
			}
			d.console.Println(fmt.Sprintf("读取命令失败: %v", err))
			continue
		}
		exit, err := d.execute(line)
		if err != nil {
			d.console.Println(fmt.Sprintf("错误: %v", err))
			continue
		}
		if exit {
			return
		}
	}
}

// Execute runs a single command and returns whether to exit.
func (d *Dispatcher) Execute(raw string) error {
	_, err := d.execute(raw)
	return err
}

func (d *Dispatcher) execute(raw string) (bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, nil
	}
	tokens, err := tokenize(raw)
	if err != nil {
		return false, err
	}
	if len(tokens) == 0 {
		return false, nil
	}
	cmd := strings.ToLower(tokens[0])
	args := tokens[1:]
	var targetFile string
	var exit bool

	switch cmd {
	case "load":
		if len(args) != 1 {
			return false, errors.New("用法: load <file>")
		}
		ed, err := d.ws.Load(args[0])
		if err != nil {
			return false, err
		}
		targetFile = ed.Path()
		d.console.Println("已加载: " + ed.Path())
	case "save":
		if len(args) == 0 {
			if err := d.ws.Save(""); err != nil {
				return false, err
			}
			ed, _ := d.ws.ActiveEditor()
			if ed != nil {
				targetFile = ed.Path()
			}
			d.console.Println("已保存当前文件")
		} else if len(args) == 1 && strings.ToLower(args[0]) == "all" {
			if err := d.ws.SaveAll(); err != nil {
				return false, err
			}
			targetFile = ""
			d.console.Println("已保存全部文件")
		} else if len(args) == 1 {
			if err := d.ws.Save(args[0]); err != nil {
				return false, err
			}
			abs, _ := filepath.Abs(args[0])
			targetFile = abs
			d.console.Println("已保存: " + abs)
		} else {
			return false, errors.New("用法: save [file|all]")
		}
	case "init":
		if len(args) == 0 {
			return false, errors.New("用法: init <file> [with-log]")
		}
		withLog := len(args) > 1 && args[1] == "with-log"
		ed, err := d.ws.Init(args[0], withLog)
		if err != nil {
			return false, err
		}
		targetFile = ed.Path()
		d.console.Println("已创建缓冲区: " + ed.Path())
	case "close":
		var requesting string
		if len(args) > 0 {
			requesting = args[0]
		}
		var abs string
		if requesting != "" {
			abs, _ = filepath.Abs(requesting)
			targetFile = abs
		} else if ed, _ := d.ws.ActiveEditor(); ed != nil {
			targetFile = ed.Path()
		}
		if err := d.ws.Close(requesting); err != nil {
			return false, err
		}
		d.console.Println("已关闭")
	case "edit":
		if len(args) != 1 {
			return false, errors.New("用法: edit <file>")
		}
		if err := d.ws.Edit(args[0]); err != nil {
			return false, err
		}
		ed, _ := d.ws.ActiveEditor()
		if ed != nil {
			targetFile = ed.Path()
		}
		d.console.Println("已切换活动文件")
	case "editor-list":
		d.printEditors()
	case "dir-tree":
		var dir string
		if len(args) > 0 {
			dir = args[0]
		}
		result, err := d.ws.DirTree(dir)
		if err != nil {
			return false, err
		}
		d.console.Println(result)
	case "undo":
		if err := d.ws.Undo(); err != nil {
			return false, err
		}
		if ed, err := d.ws.ActiveEditor(); err == nil {
			targetFile = ed.Path()
		}
		d.console.Println("已撤销")
	case "redo":
		if err := d.ws.Redo(); err != nil {
			return false, err
		}
		if ed, err := d.ws.ActiveEditor(); err == nil {
			targetFile = ed.Path()
		}
		d.console.Println("已重做")
	case "append":
		if len(args) != 1 {
			return false, errors.New("用法: append \"text\"")
		}
		ed, err := d.ws.ActiveEditor()
		if err != nil {
			return false, err
		}
		if err := ed.Append(args[0]); err != nil {
			return false, err
		}
		targetFile = ed.Path()
		d.console.Println("已追加")
	case "insert":
		if len(args) != 2 {
			return false, errors.New("用法: insert <line:col> \"text\"")
		}
		line, col, err := parseLineCol(args[0])
		if err != nil {
			return false, err
		}
		ed, err := d.ws.ActiveEditor()
		if err != nil {
			return false, err
		}
		if err := ed.Insert(line, col, args[1]); err != nil {
			return false, err
		}
		targetFile = ed.Path()
		d.console.Println("已插入")
	case "delete":
		if len(args) != 2 {
			return false, errors.New("用法: delete <line:col> <len>")
		}
		line, col, err := parseLineCol(args[0])
		if err != nil {
			return false, err
		}
		length, err := strconv.Atoi(args[1])
		if err != nil {
			return false, fmt.Errorf("长度无效: %s", args[1])
		}
		ed, err := d.ws.ActiveEditor()
		if err != nil {
			return false, err
		}
		if err := ed.Delete(line, col, length); err != nil {
			return false, err
		}
		targetFile = ed.Path()
		d.console.Println("已删除")
	case "replace":
		if len(args) != 3 {
			return false, errors.New("用法: replace <line:col> <len> \"text\"")
		}
		line, col, err := parseLineCol(args[0])
		if err != nil {
			return false, err
		}
		length, err := strconv.Atoi(args[1])
		if err != nil {
			return false, fmt.Errorf("长度无效: %s", args[1])
		}
		ed, err := d.ws.ActiveEditor()
		if err != nil {
			return false, err
		}
		if err := ed.Replace(line, col, length, args[2]); err != nil {
			return false, err
		}
		targetFile = ed.Path()
		d.console.Println("已替换")
	case "show":
		ed, err := d.ws.ActiveEditor()
		if err != nil {
			return false, err
		}
		targetFile = ed.Path()
		displayStart := 1
		start := 1
		end := 0
		if len(args) == 0 {
			start = 1
			end = 0
		} else if len(args) == 1 {
			var parseErr error
			start, end, parseErr = parseRange(args[0])
			if parseErr != nil {
				return false, parseErr
			}
			if start == 0 {
				start = 1
			}
			if end == 0 {
				end = 0
			}
			displayStart = start
		} else {
			return false, errors.New("用法: show [start:end]")
		}
		lines, err := ed.Show(start, end)
		if err != nil {
			return false, err
		}
		for i, line := range lines {
			d.console.Println(fmt.Sprintf("%d: %s", displayStart+i, line))
		}
	case "log-on":
		fileArg, err := d.resolveFileArg(args)
		if err != nil {
			return false, err
		}
		if err := d.logger.Enable(fileArg); err != nil {
			return false, err
		}
		targetFile = fileArg
		d.console.Println("已开启日志")
	case "log-off":
		fileArg, err := d.resolveFileArg(args)
		if err != nil {
			return false, err
		}
		if err := d.logger.Disable(fileArg); err != nil {
			return false, err
		}
		targetFile = fileArg
		d.console.Println("已关闭日志")
	case "log-show":
		fileArg, err := d.resolveFileArg(args)
		if err != nil {
			return false, err
		}
		targetFile = fileArg
		content, err := d.logger.Show(fileArg)
		if err != nil {
			return false, err
		}
		d.console.Println(content)
	case "exit":
		if err := d.handleExit(); err != nil {
			return false, err
		}
		exit = true
	default:
		return false, fmt.Errorf("未知命令: %s", cmd)
	}

	if cmd != "exit" {
		d.ws.PublishCommand(cmd, raw, targetFile)
	}
	return exit, nil
}

func (d *Dispatcher) resolveFileArg(args []string) (string, error) {
	if len(args) > 1 {
		return "", errors.New("命令参数过多")
	}
	if len(args) == 1 {
		return filepath.Abs(args[0])
	}
	ed, err := d.ws.ActiveEditor()
	if err != nil {
		return "", err
	}
	return ed.Path(), nil
}

func (d *Dispatcher) printEditors() {
	infos := d.ws.List()
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Path < infos[j].Path
	})
	for _, info := range infos {
		activeMark := " "
		if info.Active {
			activeMark = "*"
		}
		line := fmt.Sprintf("%s %s", activeMark, info.Name)
		if info.Modified {
			line += " [modified]"
		}
		d.console.Println(line)
	}
}

func (d *Dispatcher) handleExit() error {
	infos := d.ws.List()
	for _, info := range infos {
		if !info.Modified {
			continue
		}
		save, err := d.console.ConfirmSave(info.Path)
		if err != nil {
			return err
		}
		if save {
			if err := d.ws.Save(info.Path); err != nil {
				return err
			}
		}
	}
	if err := d.ws.Persist(); err != nil {
		return err
	}
	d.console.Println("已退出并保存工作区状态")
	return nil
}

func tokenize(line string) ([]string, error) {
	var tokens []string
	var builder strings.Builder
	inQuotes := false
	tokenReady := false
	for i := 0; i < len(line); i++ {
		ch := line[i]
		switch ch {
		case '"':
			if inQuotes {
				inQuotes = false
				if builder.Len() == 0 {
					tokenReady = true
				}
			} else {
				inQuotes = true
			}
		case ' ', '\t':
			if inQuotes {
				builder.WriteByte(ch)
			} else if builder.Len() > 0 || tokenReady {
				tokens = append(tokens, builder.String())
				builder.Reset()
				tokenReady = false
			}
		default:
			builder.WriteByte(ch)
			tokenReady = false
		}
	}
	if inQuotes {
		return nil, errors.New("缺少匹配的引号")
	}
	if builder.Len() > 0 || tokenReady {
		tokens = append(tokens, builder.String())
	}
	return tokens, nil
}

func parseLineCol(token string) (int, int, error) {
	parts := strings.Split(token, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("位置参数无效: %s", token)
	}
	line, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("行号无效: %s", parts[0])
	}
	col, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("列号无效: %s", parts[1])
	}
	return line, col, nil
}

func parseRange(token string) (int, int, error) {
	if !strings.Contains(token, ":") {
		start, err := strconv.Atoi(token)
		if err != nil {
			return 0, 0, fmt.Errorf("范围无效: %s", token)
		}
		return start, start, nil
	}
	parts := strings.Split(token, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("范围无效: %s", token)
	}
	var start, end int
	var err error
	if parts[0] != "" {
		start, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, fmt.Errorf("起始行无效: %s", parts[0])
		}
	}
	if parts[1] != "" {
		end, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, fmt.Errorf("结束行无效: %s", parts[1])
		}
	}
	return start, end, nil
}
