package tui

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

type runningWatcher interface {
	Running() bool
}

func watcherIsRunning(w any) bool {
	rw, ok := w.(runningWatcher)
	return ok && rw.Running()
}

func (m model) handleWatch(action string) (model, tea.Cmd) {
	switch action {
	case "start":
		if watcherIsRunning(any(m.watcher)) {
			m = m.appendActivity("Watcher already running")
			return m, nil
		}
		m.watcher.Start()
	case "stop":
		if !watcherIsRunning(any(m.watcher)) {
			m = m.appendActivity("Watcher not running")
			return m, nil
		}
		m.watcher.Stop()
	default:
		m = m.appendActivity("Usage: watch start|stop")
	}
	return m, nil
}

func (m model) handleStatus() (model, tea.Cmd) {
	agents := m.manager.List()
	if len(agents) == 0 {
		m = m.appendActivity("No agents running")
		return m, nil
	}

	header := fmt.Sprintf("%-8s %-30s %-10s %s", "Issue", "Title", "Status", "Elapsed")
	sep := strings.Repeat("─", 70)
	m = m.appendActivity(header, sep)

	for _, a := range agents {
		elapsed := time.Since(a.StartTime).Truncate(time.Second)
		line := fmt.Sprintf("%-8d %-30s %-10s %s",
			a.IssueNumber,
			truncate(a.IssueTitle, 30),
			string(a.Status),
			elapsed)
		m = m.appendActivity(line)
	}
	return m, nil
}

func (m model) handleStop(issueStr string) (model, tea.Cmd) {
	issueNum, err := strconv.Atoi(issueStr)
	if err != nil {
		m = m.appendActivity(fmt.Sprintf("Invalid issue number: %s", issueStr))
		return m, nil
	}

	agents := m.manager.List()
	for _, a := range agents {
		if a.IssueNumber == issueNum {
			if err := m.manager.Stop(a.ID); err != nil {
				m = m.appendActivity(fmt.Sprintf("Error stopping agent: %v", err))
			} else {
				m = m.appendActivity(fmt.Sprintf("Stopped agent for issue %d", issueNum))
			}
			return m, nil
		}
	}
	m = m.appendActivity(fmt.Sprintf("No agent running for issue %d", issueNum))
	return m, nil
}

func (m model) handleHelp() (model, tea.Cmd) {
	help := []string{
		"Available commands:",
		"  watch start    - Start watching for labeled issues",
		"  watch stop     - Stop watching",
		"  status         - List all agents with details",
		"  stop <issue>   - Stop agent for specific issue number",
		"  plan [desc]    - Start interactive planning session",
		"  exit           - Exit (Ctrl+C also works)",
		"  help           - Show this help message",
	}
	m = m.appendActivity(help...)
	return m, nil
}

func (m model) handleExec(name string, args ...string) (model, tea.Cmd) {
	c := tea.ExecProcess(execCommand(name, args...), func(err error) tea.Msg {
		return execDoneMsg{err: err}
	})
	return m, c
}

func (m model) handlePlan(description string) (model, tea.Cmd) {
	args := []string{"chat", "--agent", "planner"}
	if description != "" {
		args = append(args, description)
	}
	c := tea.ExecProcess(execCommand("kiro-cli", args...), func(err error) tea.Msg {
		return execDoneMsg{err: err, planCmd: true}
	})
	return m, c
}

type execDoneMsg struct {
	err     error
	planCmd bool
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func (m model) labelLastIssue(issueNums ...int) {
	if len(issueNums) == 0 {
		return
	}

	issueNum := issueNums[0]
	exec.Command("gh", "issue", "edit", fmt.Sprintf("%d", issueNum),
		"--repo", m.config.Repo, "--add-label", m.config.Label).Run()
}
