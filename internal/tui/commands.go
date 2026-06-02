package tui

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/jbrinkman/kiro-krew/internal/github"
	"github.com/jbrinkman/kiro-krew/internal/version"
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
	switch strings.ToLower(action) {
	case "start":
		if watcherIsRunning(any(m.watcher)) {
			m = m.appendActivity(m.styles.Warning.Render("Watcher already running"))
			return m, nil
		}
		// Update log position to current end before starting watcher
		if info, err := m.logReader.Stat(); err == nil {
			m.lastLogPos = info.Size()
		}
		m.watcher.Start()
		m = m.appendActivity(m.styles.Success.Render("Watcher started"))
	case "stop":
		if !watcherIsRunning(any(m.watcher)) {
			m = m.appendActivity(m.styles.Warning.Render("Watcher not running"))
			return m, nil
		}
		m.watcher.Stop()
		m = m.appendActivity(m.styles.Success.Render("Watcher stopped"))
	default:
		m = m.appendActivity(m.styles.Error.Render("Usage: watch start|stop"))
	}
	return m, nil
}

func (m model) handleStatus() (model, tea.Cmd) {
	agents := m.manager.List()
	if len(agents) == 0 {
		m = m.appendActivity(m.styles.Warning.Render("No agents running"))
		return m, nil
	}

	header := fmt.Sprintf("%-8s %-30s %-10s %s", "Issue", "Title", "Status", "Elapsed")
	sep := strings.Repeat("─", 70)
	m = m.appendActivity(m.styles.Prompt.Render(header), m.styles.Separator.Render(sep))

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
		m = m.appendActivity(m.styles.Error.Render(fmt.Sprintf("Invalid issue number: %s", issueStr)))
		return m, nil
	}

	agents := m.manager.List()
	for _, a := range agents {
		if a.IssueNumber == issueNum {
			if err := m.manager.Stop(a.ID); err != nil {
				m = m.appendActivity(m.styles.Error.Render(fmt.Sprintf("Error stopping agent: %v", err)))
			} else {
				m = m.appendActivity(m.styles.Success.Render(fmt.Sprintf("Stopped agent for issue %d", issueNum)))
			}
			return m, nil
		}
	}
	m = m.appendActivity(m.styles.Warning.Render(fmt.Sprintf("No agent running for issue %d", issueNum)))
	return m, nil
}

func (m model) handleHelp() (model, tea.Cmd) {
	help := []string{
		m.styles.Prompt.Render("Available commands:"),
		"  watch start    - Start watching for labeled issues",
		"  watch stop     - Stop watching",
		"  status         - List all agents with details",
		"  stop <issue>   - Stop agent for specific issue number",
		"  plan [desc]    - Start interactive planning session",
		"  about          - Show version information and check for updates",
		"  exit           - Exit (Ctrl+C also works)",
		"  help           - Show this help message",
	}
	m = m.appendActivity(help...)
	return m, nil
}

func (m model) handlePlan(description string) (model, tea.Cmd) {
	args := []string{"chat", "--classic", "--agent", "planner"}
	if description != "" {
		args = append(args, description)
	}
	m.input.Blur()
	// Wrap in shell with clear and centered ASCII art banner
	banner := `cols=$(tput cols 2>/dev/null || echo 80)
art1="  _  ___              _  __                   "
art2=" | |/ (_)_ __ ___    | |/ /_ __ _____      __"
art3=" | ' /| | '__/ _ \   | ' /| '__/ _ \ \ /\ / /"
art4=" | . \| | | | (_) |  | . \| | |  __/\ V  V / "
art5=" |_|\_\_|_|  \___/   |_|\_\_|  \___| \_/\_/  "
pad() { w=${#1}; p=$(( (cols - w) / 2 )); [ "$p" -lt 0 ] && p=0; printf "%*s%s\n" "$p" "" "$1"; }
echo ""
pad "$art1"
pad "$art2"
pad "$art3"
pad "$art4"
pad "$art5"
echo ""`
	script := "clear && " + banner + " && exec kiro-cli \"$@\""
	cmd := exec.Command("sh", append([]string{"-c", script, "sh"}, args...)...)
	c := tea.ExecProcess(cmd, func(err error) tea.Msg {
		return execDoneMsg{err: err}
	})
	return m, c
}

type execDoneMsg struct {
	err error
}

type updateCheckMsg struct {
	release *github.Release
	err     error
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func (m model) handleAbout() (model, tea.Cmd) {
	info := version.Info()

	m = m.appendActivity(
		m.styles.Prompt.Render("Kiro-Krew Version Information:"),
		fmt.Sprintf("  Version:    %s", info["version"]),
		fmt.Sprintf("  Build Date: %s", info["build_date"]),
		fmt.Sprintf("  Go Version: %s", info["go_version"]),
		fmt.Sprintf("  Arch:       %s", info["arch"]),
		"",
		"Checking for updates...",
	)

	return m, checkForUpdateCmd()
}

func checkForUpdateCmd() tea.Cmd {
	return func() tea.Msg {
		release, err := github.GetLatestRelease("jbrinkman/kiro-krew")
		return updateCheckMsg{release: release, err: err}
	}
}
