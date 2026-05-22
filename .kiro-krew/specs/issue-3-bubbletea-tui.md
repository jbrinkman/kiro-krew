# Spec: Replace REPL with Bubbletea TUI

## Overview

Replace the `bufio.Scanner` REPL in `internal/repl/repl.go` with a bubbletea-based TUI that separates activity display from command input.

## Architecture

### New Package Structure

```
internal/
  tui/
    tui.go        — Main bubbletea model, Init/Update/View
    commands.go   — Command parsing and execution (extracted from repl.go)
    log.go        — File-based logger + log reader for activity pane
```

### Components

1. **Model** (`tui.go`) — Bubbletea model with:
   - `activityLines []string` — log lines displayed in the activity pane
   - `input textinput.Model` — command prompt (bubbles/textinput)
   - `confirmingExit bool` — exit confirmation state
   - `suspendedCmd *exec.Cmd` — for subprocess overlay
   - `watcher`, `manager` — existing dependencies

2. **Log System** (`log.go`):
   - Redirect Go's `log` package output to a file (`.kiro-krew/kiro-krew.log`)
   - TUI tails the log file via a bubbletea `Cmd` that watches for new lines
   - Activity pane displays the last N lines that fit the terminal height

3. **Command Handler** (`commands.go`):
   - Same commands: `watch start/stop`, `status`, `stop <issue>`, `exit`, `help`
   - Commands return strings (for activity pane display) instead of printing to stdout
   - `status` returns a formatted table string

4. **Subprocess Overlay**:
   - Use `tea.ExecProcess()` (bubbletea's built-in) to hand terminal to child process
   - TUI suspends, child runs, TUI resumes on exit

### Layout (View)

```
┌─────────────────────────────────────┐
│ Activity Pane (scrollable log)      │
│ [watcher] started — polling...      │
│ [agent] started agent-3-1234...     │
│                                     │
├─────────────────────────────────────┤
│ kiro-krew> _                        │
└─────────────────────────────────────┘
```

### Integration Points

- `cmd/kiro-krew/main.go`: Replace `repl.New(w, m).Run()` with `tui.Run(w, m)`
- `internal/agent/manager.go`: No changes (log.Printf goes to file)
- `internal/watcher/watcher.go`: No changes (log.Printf goes to file)
- `internal/repl/`: Removed (replaced by `internal/tui/`)

### Dependencies

- `github.com/charmbracelet/bubbletea`
- `github.com/charmbracelet/bubbles` (textinput)
- `github.com/charmbracelet/lipgloss` (styling)

## Key Decisions

- Use Go's `log.SetOutput()` to redirect all logging to file — zero changes to watcher/agent code
- Tail the log file with `os.Stat` polling (simple, no fsnotify dependency needed)
- Use bubbletea's `tea.ExecProcess` for subprocess overlay (handles terminal restore)
- Activity pane auto-scrolls to bottom; no manual scroll needed for MVP
