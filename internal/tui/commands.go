package tui

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/jbrinkman/kiro-krew/internal/config"
	"github.com/jbrinkman/kiro-krew/internal/github"
	"github.com/jbrinkman/kiro-krew/internal/session"
	"github.com/jbrinkman/kiro-krew/internal/version"
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
	content := []string{}

	// Add tab information section
	tabs := m.tabManager.GetTabs()
	activeTabIndex := m.tabManager.GetActiveTabIndex()
	
	content = append(content, m.styles.Prompt.Render("Active Tab"))
	if len(tabs) > 0 {
		activeTab := m.tabManager.GetActiveTab()
		content = append(content, fmt.Sprintf("  Current: %s", activeTab.Title()))
		content = append(content, fmt.Sprintf("  Type: %s", getTabTypeName(activeTab.Type())))
	} else {
		content = append(content, "  No tabs")
	}
	
	content = append(content, "")
	content = append(content, m.styles.Prompt.Render(fmt.Sprintf("Tabs (%d open)", len(tabs))))
	
	for i, tab := range tabs {
		indicator := "  "
		if i == activeTabIndex {
			indicator = "* "
		}
		closable := ""
		if tab.IsClosable() {
			closable = " (closable)"
		}
		content = append(content, fmt.Sprintf("%s%s%s", indicator, tab.Title(), closable))
	}
	
	if len(tabs) > 1 {
		content = append(content, "")
		content = append(content, m.styles.Prompt.Render("Navigation"))
		content = append(content, "  F2 - Toggle between main and first agent tab")
		content = append(content, "  Ctrl+Tab - Next tab")
		content = append(content, "  Ctrl+Shift+Tab - Previous tab")
	}

	if len(agents) == 0 {
		content = append(content, "", m.styles.Warning.Render("No agents running"))
	} else {
		content = append(content, "", m.styles.Prompt.Render("Agents"))
		
		contentWidth := m.getOverlayContentWidth()

		issueW := max(int(float64(contentWidth)*0.11), 5)
		titleW := max(int(float64(contentWidth)*0.43), 10)
		statusW := max(int(float64(contentWidth)*0.14), 7)

		// Ensure column widths fit within contentWidth (3 spaces between columns).
		minColW := 1
		for issueW+titleW+statusW+minColW+3 > contentWidth && titleW > minColW {
			titleW--
		}
		for issueW+titleW+statusW+minColW+3 > contentWidth && statusW > minColW {
			statusW--
		}
		for issueW+titleW+statusW+minColW+3 > contentWidth && issueW > minColW {
			issueW--
		}
		elapsedW := contentWidth - issueW - titleW - statusW - 3
		if elapsedW < minColW {
			elapsedW = minColW
		}

		header := fmt.Sprintf("%-*s %-*s %-*s %-*s", issueW, "Issue", titleW, "Title", statusW, "Status", elapsedW, "Elapsed")
		sep := strings.Repeat("-", contentWidth)
		content = append(content, m.styles.Separator.Render(sep), header, m.styles.Separator.Render(sep))

		for _, a := range agents {
			elapsed := time.Since(a.StartTime).Truncate(time.Second)
			line := fmt.Sprintf("%-*d %-*s %-*s %-*s",
				issueW, a.IssueNumber,
				titleW, truncate(a.IssueTitle, titleW),
				statusW, string(a.Status),
				elapsedW, elapsed)
			content = append(content, line)
		}
	}

	m = m.activateOverlay(overlayStatus, "System Status", content)
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
	content := []string{
		m.styles.Prompt.Render("Available commands:"),
		"  watch start    - Start watching for labeled issues",
		"  watch stop     - Stop watching",
		"  status         - List all agents with details",
		"  stop <issue>   - Stop agent for specific issue number",
		"  plan [desc]    - Start interactive planning session",
		"  theme          - Show current theme",
		"  theme <name>   - Switch to theme",
		"  about          - Show version information and check for updates",
		"  exit           - Exit (Ctrl+C also works)",
		"  help           - Show this help message",
		"",
		m.styles.Prompt.Render("Hotkeys:"),
		"  F2, o          - Toggle between console and agent output views",
		"  Ctrl+Alt+P     - Toggle between console and planning modes",
	}

	m = m.activateOverlay(overlayHelp, "Help", content)
	return m, nil
}

func (m model) handlePlan(description string) (model, tea.Cmd) {
	// Suspend agent output capture before starting planning mode
	m.manager.SuspendOutputCapture()

	// Check for existing planning sessions
	sessions, err := m.sessionManager.List()
	if err != nil {
		m = m.appendActivity(m.styles.Error.Render(fmt.Sprintf("Failed to check sessions: %v", err)))
		return m, nil
	}

	// Look for existing planning sessions
	var planningSessionID string
	for _, sessionID := range sessions {
		state, err := m.sessionManager.Load(sessionID)
		if err != nil {
			// Log corrupted session but continue
			m = m.appendActivity(m.styles.Warning.Render(fmt.Sprintf("Skipping corrupted session %s", sessionID[:8])))
			continue
		}
		if state.Type == session.Planning {
			planningSessionID = sessionID
			break
		}
	}

	if planningSessionID != "" {
		// Resume existing session
		m = m.appendActivity(m.styles.Success.Render(fmt.Sprintf("Resuming planning session %s...", planningSessionID[:8])))
	} else {
		// Create new session
		sessionID, err := m.sessionManager.Create(session.Planning)
		if err != nil {
			m = m.appendActivity(m.styles.Error.Render(fmt.Sprintf("Failed to create session: %v", err)))
			return m, nil
		}
		m = m.appendActivity(m.styles.Success.Render(fmt.Sprintf("Created new planning session %s", sessionID[:8])))
	}

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
	if max <= 0 {
		return ""
	}
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m model) getOverlayContentWidth() int {
	overlayWidth := int(float64(m.width) * 0.6)
	if overlayWidth < 40 {
		overlayWidth = 40
	}

	// Ensure overlay doesn't exceed screen bounds (mirrors renderOverlay()).
	if overlayWidth >= m.width {
		overlayWidth = m.width - 2
	}

	contentWidth := overlayWidth - 6 // Account for border and padding
	if contentWidth < 1 {
		contentWidth = 1
	}
	return contentWidth
}

func (m model) handleAbout() (model, tea.Cmd) {
	info := version.Info()

	content := []string{
		fmt.Sprintf("  Version:    %s", info["version"]),
		fmt.Sprintf("  Build Date: %s", info["build_date"]),
		fmt.Sprintf("  Go Version: %s", info["go_version"]),
		fmt.Sprintf("  Arch:       %s", info["arch"]),
		"",
		"Checking for updates...",
	}

	m = m.activateOverlay(overlayAbout, "Kiro-Krew Version Information", content)
	return m, checkForUpdateCmd()
}

func (m model) handleTheme(args []string) (model, tea.Cmd) {
	if len(args) == 0 {
		// No longer show overlay for current theme - persistent display handles this
		return m, nil
	}

	if len(args) > 1 {
		m = m.appendActivity(m.styles.Error.Render("Usage: theme [name]"))
		return m, nil
	}

	themeName := args[0]

	// Try to load the theme (this handles validation)
	theme, err := config.LoadTheme(themeName)
	if err != nil {
		available := config.GetAvailableThemes()
		m = m.appendActivity(
			m.styles.Error.Render(fmt.Sprintf("Failed to load theme '%s': %v", themeName, err)),
			m.styles.Warning.Render(fmt.Sprintf("Available themes: %s", strings.Join(available, ", "))),
		)
		return m, nil
	}

	previousTheme := m.config.Theme
	previousLoadedTheme := m.config.LoadedTheme

	// Update config
	m.config.Theme = themeName
	m.config.LoadedTheme = theme

	// Save config
	if err := m.config.Save(); err != nil {
		m.config.Theme = previousTheme
		m.config.LoadedTheme = previousLoadedTheme
		m = m.appendActivity(m.styles.Error.Render(fmt.Sprintf("Failed to save config: %v", err)))
		return m, nil
	}

	// Update styles with new theme
	m.styles = NewStyles(theme)
	m.viewManager.UpdateOutputViewStyles(m.styles)

	m = m.appendActivity(m.styles.Success.Render(fmt.Sprintf("Theme changed to: %s", themeName)))
	return m, tea.ClearScreen
}

func checkForUpdateCmd() tea.Cmd {
	return func() tea.Msg {
		release, err := github.GetLatestRelease("jbrinkman/kiro-krew")
		return updateCheckMsg{release: release, err: err}
	}
}

// getTabTypeName converts TabType enum to readable string
func getTabTypeName(tabType TabType) string {
	switch tabType {
	case TabTypeMain:
		return "Main Console"
	case TabTypeAgent:
		return "Agent Output"
	default:
		return "Unknown"
	}
}

// switchToPlanningMode switches from console to planning mode while preserving console state
func (m model) switchToPlanningMode() (model, tea.Cmd) {
	// Suspend agent output capture before entering planning mode
	m.manager.SuspendOutputCapture()

	// Preserve current console state
	m.consoleState.inputValue = m.input.Value()
	m.consoleState.activityLines = make([]string, len(m.activityLines))
	copy(m.consoleState.activityLines, m.activityLines)

	// Check for existing planning sessions with error recovery
	sessions, err := m.sessionManager.List()
	if err != nil {
		m = m.appendActivity(m.styles.Error.Render(fmt.Sprintf("Failed to check sessions: %v", err)))
		return m, nil
	}

	// Look for existing planning session with corruption handling
	var planningSessionID string
	for _, sessionID := range sessions {
		state, err := m.sessionManager.Load(sessionID)
		if err != nil {
			if strings.Contains(err.Error(), "corruption") {
				m = m.appendActivity(m.styles.Warning.Render(fmt.Sprintf("Removing corrupted session %s", sessionID[:8])))
				_ = m.sessionManager.Delete(sessionID)
			}
			continue
		}
		if state.Type == session.Planning {
			planningSessionID = sessionID
			break
		}
	}

	var sessionMsg string
	if planningSessionID == "" {
		m = m.appendActivity(m.styles.Warning.Render("No active planning session"))
		return m, nil
	}
	sessionMsg = fmt.Sprintf("Resuming planning session %s...", planningSessionID[:8])

	m = m.appendActivity(m.styles.Success.Render(sessionMsg))
	m.currentMode = session.Planning
	m.input.Blur()

	return m, tea.ClearScreen
}

// switchToConsoleMode switches from planning to console mode while preserving planning state
func (m model) switchToConsoleMode() (model, tea.Cmd) {
	if m.activePlanningSession != nil {
		// Suspend and preserve planning session state with error handling
		if err := m.activePlanningSession.SuspendAndDetach(); err != nil {
			m = m.appendActivity(m.styles.Error.Render(fmt.Sprintf("Failed to suspend planning session: %v", err)))
			// Continue anyway, don't block mode switch
		} else {
			m = m.appendActivity(m.styles.Success.Render("Planning session suspended, switching to console mode"))
		}
		m.activePlanningSession = nil
	}

	// Resume agent output capture when returning to console mode
	m.manager.ResumeOutputCapture()

	// Switch to console mode and restore console state
	m.currentMode = session.Console
	m = m.restoreConsoleState()
	m.input.Focus()

	return m, tea.Batch(textinput.Blink, tea.ClearScreen)
}

// restoreConsoleState restores the console state after returning from planning mode
func (m model) restoreConsoleState() model {
	if m.consoleState != nil {
		m.input.SetValue(m.consoleState.inputValue)
		m.activityLines = make([]string, len(m.consoleState.activityLines))
		copy(m.activityLines, m.consoleState.activityLines)
	}
	m.currentMode = session.Console
	return m
}
