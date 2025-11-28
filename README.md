# Lab2：基于命令行的多文件编辑器

基于 Go 1.22.5，在 Lab1 的文本工作区基础上扩展 XML 编辑、会话时长统计与拼写检查三大模块，同时保留 Lab1 的全部 18 条命令并补充 9 条新/改命令。

## 系统架构

```Mermaid
graph TD
    CLI[cli.Dispatcher] -->|commands & queries| WS[workspace.Workspace]

    subgraph Workspace Core
        WS --> TextEd[editor.TextEditor]
        WS --> XMLEd[editor.XMLEditor]
        WS --> Stats[statistics.Tracker]
        WS --> Keeper[workspace.StateKeeper]
    end

    subgraph XML Extensions
        XMLEd --> Spell[spellcheck.Service]
    end

    subgraph Event System
        Bus[events.Bus] -->|Observer| Log[logging.Manager]
        WS -.-> Bus
    end

    style CLI fill:#f9f,stroke:#333,stroke-width:2px
    style WS fill:#bbf,stroke:#333,stroke-width:2px
```

````

| 模块                     | 责任                                     | 依赖关系                                                       |
| ------------------------ | ---------------------------------------- | -------------------------------------------------------------- |
| `src/cli`                | 解析命令、提供提示界面，分发到工作区     | 依赖 `workspace`、`logging`                                    |
| `src/workspace`          | 协调编辑器、统计、拼写和日志；持久化状态 | 依赖 `editor`、`statistics`、`spellcheck`、`logging`、`events` |
| `src/editor/text_editor` | 文本缓冲区及撤销/重做                    | 纯内存实现                                                     |
| `src/editor/xml_editor`  | DOM 树编辑、XML 序列化、撤销/重做        | 使用组合模式维护树，依赖 `encoding/xml`                        |
| `src/statistics`         | 会话级计时与结果格式化                   | 独立模块，被 `workspace` 和 `cli` 使用                         |
| `src/spellcheck`         | 拼写检查服务及默认词库                   | 通过接口隔离第三方依赖，可替换实现                             |
| `src/logging`            | 监听命令事件写入 `.name.log`             | 订阅 `events.Bus`                                              |
| `src/events`             | 简单观察者总线                           | 被工作区与日志共享                                             |
| `src/fs`                 | `dir-tree` 目录渲染                      | 独立工具                                                       |

## 核心设计与模式

- **编辑器多态**：`editor.Editor` 抽象统一了文本与 XML 编辑器；CLI 在运行时按需求断言 `TextDocument` / `XMLTreeEditor` 能力。
- **XML 编辑器**：
  - 组合模式维护 DOM 树与 `id → node` 索引。
  - 命令模式 + 快照实现撤销/重做。
  - 解析时阻止混合内容，保存时生成 UTF-8 XML，根属性 `log="true"` 自动触发日志。
- **统计模块**：
  - 观察者式 `workspace.setActive` 触发 `statistics.Tracker.Switch`，会话结束或关闭文件时刷新。
  - `statistics.FormatDuration` 按实验规范输出“秒/分钟/小时/天”。
- **拼写检查模块**：
  - 通过 `spellcheck.Checker` 接口隔离第三方依赖。本实现内嵌轻量词典 + Levenshtein 距离（适配器位置，可替换为语言工具 API）。
  - `Service` 针对文本行与 XML 文本节点分别输出问题列表。
- **横切功能整合**：`workspace.Workspace` 注入 `statistics`、`spellcheck`、`logging`，并在 `Persist` 时统一刷新状态。

## 命令清单

- **保持不变**：Lab1 的全部 18 条文本命令与日志控制命令。
- **修改**：
  - `init <text|xml> <file> [with-log]`：支持选择文本/默认 XML 根结构。
  - `editor-list`：输出 `* name [modified] (2小时15分钟)`，会话时长来自统计模块。
- **新增**：
  - XML 编辑：`insert-before`、`append-child`、`edit-id`、`edit-text`、`delete-element`、`xml-tree [file]`
  - 拼写检查：`spell-check [file]` （文本 & XML 文本节点）

## 运行说明

- **环境**：Go 1.22.5，UTF-8 文件编码。
- **依赖安装**：`go mod tidy`
- **运行程序**：`go run .`
- **执行全部测试**：`go test ./...`
- **二进制**：仓库提供 `editor.exe`（Windows）供直接体验。

## 测试说明

| 测试文件                            | 关注点                            |
| ----------------------------------- | --------------------------------- |
| `tests/editor/xml_editor_test.go`   | XML 插入/删除/撤销、树形打印      |
| `tests/statistics/tracker_test.go`  | 计时切换、格式化边界              |
| `tests/spellcheck/service_test.go`  | 文本与 XML 拼写报告               |
| `tests/cli/dispatcher_test.go`      | 命令分发回归（load、editor-list） |
| `tests/workspace/workspace_test.go` | 多文件切换、持久化、关闭逻辑      |

所有测试均在 Windows 下通过：

```bash
go test ./...
````

输出 `ok softwaredesign/tests/...`，覆盖新增模块与命令主要路径。

## 相比 Lab1 的增强

- 引入 `editor.Editor` 多态层，实现文本/DOM 编辑器共存。
- `workspace` 集成统计与拼写服务，并通过依赖注入保持可替换性。
- CLI 增加 XML 专属命令与拼写检查输出，列表展示会话时长。
- 新增文档化测试覆盖 XML、统计、拼写三大增量模块。
