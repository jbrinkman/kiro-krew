package tui

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"golang.org/x/mod/semver"

	"github.com/jbrinkman/kiro-krew/internal/agent"
	"github.com/jbrinkman/kiro-krew/internal/config"
	"github.com/jbrinkman/kiro-krew/internal/hotkey"
	"github.com/jbrinkman/kiro-krew/internal/session"
	"github.com/jbrinkman/kiro-krew/internal/version"
	"github.com/jbrinkman/kiro-krew/internal/watcher"
)

type logMsg string

type tickMsg struct{}

type planningHotkeyMsg struct{}

type overlayType int

const (
	overlayNone overlayType = iota
	overlayStatus
	overlayHelp
	overlayAbout

	maxOverlayLines = 1000 // Prevent memory growth from very large overlay content

	// tabHeaderHeight is the number of lines the tab header occupies in the view
	tabHeaderHeight = 1
)

type overlayContent struct {
	title   string
	content []string
}

type consoleState struct {
	inputValue    string
	activityLines []string
}

type model struct {
	watcher               *watcher.Watcher
	manager               *agent.Manager
	sessionManager        *session.SessionManager
	config                *config.Config
	styles                *Styles
	input                 textinput.Model
	consoleViewport       viewport.Model
	activityLines         []string
	maxActivityLines      int
	width                 int
	height                int
	confirmingExit        bool
	logFile               *os.File
	logReader             *os.File
	lastLogPos            int64
	quitting              bool
	currentMode           session.SessionType
	consoleState          *consoleState
	activePlanningSession *session.PlanningSession

	// Overlay system
	activeOverlay  overlayType
	overlayContent overlayContent
	overlayWidth   int
	overlayHeight  int

	// View state management
	tabManager  *TabManager
	mainTab     *MainTab
	
	// Agent lifecycle tracking
	knownAgents        map[string]bool
	statusRunningAgents []*agent.Agent // Snapshot for deterministic number key selection
}

func newModel(w *watcher.Watcher, m *agent.Manager, cfg *config.Config, logFile *os.File, logReader *os.File) model {
	ti := textinput.New()
	ti.Prompt = "kiro-krew> "
	ti.Focus()

	theme := cfg.LoadedTheme
	styles := NewStyles(theme)

	consoleViewport := viewport.New(viewport.WithWidth(80), viewport.WithHeight(24))
	// Disable built-in key bindings — we handle scrolling explicitly
	consoleViewport.KeyMap = viewport.KeyMap{}

	// Initialize tab system
	tabManager := NewTabManager()
	mainTab := NewMainTab()
	tabManager.AddTab(mainTab)

	return model{
		watcher:          w,
		manager:          m,
		sessionManager:   session.NewSessionManager(),
		config:           cfg,
		styles:           styles,
		input:            ti,
		consoleViewport:  consoleViewport,
		logFile:          logFile,
		logReader:        logReader,
		maxActivityLines: cfg.MaxActivityLines,
		currentMode:      session.Console,
		consoleState: &consoleState{
			inputValue:    "",
			activityLines: make([]string, 0),
		},
		tabManager:  tabManager,
		mainTab:     mainTab,
		knownAgents: make(map[string]bool),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.tickCmd())
}

func (m model) tickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

// appendActivity appends lines to activityLines and trims to maxActivityLines if set.
func (m model) appendActivity(lines ...string) model {
	m.activityLines = append(m.activityLines, lines...)
	if m.maxActivityLines > 0 && len(m.activityLines) > m.maxActivityLines {
		m.activityLines = m.activityLines[len(m.activityLines)-m.maxActivityLines:]
	}
	// Sync viewport content and auto-scroll if user is near the bottom
	content := strings.Join(m.activityLines, "\n")
	m.consoleViewport.SetContent(content)
	if m.consoleViewport.ScrollPercent() >= 0.95 {
		m.consoleViewport.GotoBottom()
	}
	return m
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Resize console viewport — account for tab header + separator + input
		activityHeight := m.height - 2 - tabHeaderHeight
		if activityHeight < 1 {
			activityHeight = 1
		}
		m.consoleViewport.SetWidth(msg.Width)
		m.consoleViewport.SetHeight(activityHeight)
		// Forward to tab manager
		m.tabManager.Resize(msg.Width, msg.Height)
		// Recalculate overlay dimensions on resize
		if m.activeOverlay != overlayNone {
			m.overlayWidth = int(float64(m.width) * 0.6)
			m.overlayHeight = int(float64(m.height) * 0.6)
			if m.overlayWidth < 40 {
				m.overlayWidth = 40
			}
			if m.overlayHeight < 10 {
				m.overlayHeight = 10
			}
		}
		return m, nil

	case execDoneMsg:
		// Resume agent output capture when planning mode exits
		m.manager.ResumeOutputCapture()

		if msg.err != nil {
			m = m.appendActivity(m.styles.Error.Render(fmt.Sprintf("Planning session exited with error: %v", msg.err)))
		} else {
			m = m.appendActivity(m.styles.Success.Render("Planning session completed."))
		}

		// Clean up planning session tracking
		if m.activePlanningSession != nil {
			m.activePlanningSession = nil
		}

		m = m.restoreConsoleState()
		m.input.Focus()
		return m, tea.Batch(textinput.Blink, tea.ClearScreen)

	case updateCheckMsg:
		updateLines := []string{}
		if msg.err != nil {
			updateLines = append(updateLines,
				m.styles.Warning.Render("Update Status: Unable to check for updates"),
				m.styles.Error.Render(fmt.Sprintf("  Error: %v", msg.err)),
			)
		} else {
			current := version.Version
			latest := msg.release.TagName
			if current == "dev" {
				updateLines = append(updateLines, m.styles.Warning.Render("Update Status: Development build"))
			} else {
				// Ensure "v" prefix for semver comparison
				if !strings.HasPrefix(current, "v") {
					current = "v" + current
				}
				if !strings.HasPrefix(latest, "v") {
					latest = "v" + latest
				}
				if !semver.IsValid(current) || !semver.IsValid(latest) {
					updateLines = append(updateLines,
						m.styles.Warning.Render("Update Status: Unable to compare versions (non-semver format)"),
						fmt.Sprintf("  Current: %s, Latest: %s", version.Version, msg.release.TagName),
					)
				} else if semver.Compare(current, latest) < 0 {
					updateLines = append(updateLines,
						m.styles.Warning.Render("Update Status: Update available"),
						fmt.Sprintf("  Latest: %s (%s)", msg.release.TagName, msg.release.Name),
					)
				} else {
					updateLines = append(updateLines, m.styles.Success.Render("Update Status: Up to date"))
				}
			}
		}

		if m.activeOverlay == overlayAbout {
			// Update about overlay content
			info := version.Info()
			content := []string{
				fmt.Sprintf("  Version:    %s", info["version"]),
				fmt.Sprintf("  Build Date: %s", info["build_date"]),
				fmt.Sprintf("  Go Version: %s", info["go_version"]),
				fmt.Sprintf("  Arch:       %s", info["arch"]),
				"",
			}
			content = append(content, updateLines...)
			m.overlayContent.content = append(content, "", "Press ESC to close")
		} else {
			// Add to activity as before
			m = m.appendActivity(updateLines...)
		}
		return m, nil

	case tickMsg:
		newLines, newPos := m.readNewLogLines()
		if len(newLines) > 0 {
			m.lastLogPos = newPos
			m = m.appendActivity(newLines...)
		}
		
		// Check for agent lifecycle changes
		m = m.updateAgentTabs()
		
		return m, m.tickCmd()

	case planningHotkeyMsg:
		if m.currentMode == session.Planning {
			return m.switchToConsoleMode()
		}
		return m, nil

	case hotkey.HotkeyTriggeredMsg:
		if m.currentMode == session.Console {
			return m.switchToPlanningMode()
		} else if m.currentMode == session.Planning {
			return m.switchToConsoleMode()
		}
		return m, nil

	case hotkey.HotkeyErrorMsg:
		m = m.appendActivity(m.styles.Error.Render(fmt.Sprintf("Hotkey error: %v", msg.Err)))
		return m, nil

	case tea.MouseWheelMsg:
		// Handle mouse wheel scrolling in main tab when no overlay active
		activeTab := m.tabManager.GetActiveTab()
		if m.activeOverlay == overlayNone && activeTab != nil && activeTab.Type() == TabTypeMain {
			mouse := msg.Mouse()
			if mouse.Button == tea.MouseWheelUp {
				m.consoleViewport.ScrollUp(3)
			} else if mouse.Button == tea.MouseWheelDown {
				m.consoleViewport.ScrollDown(3)
			}
			return m, nil
		}
		// Forward to tab manager for agent tabs
		if cmd := m.tabManager.Update(msg); cmd != nil {
			return m, cmd
		}
		return m, nil

	case tea.MouseClickMsg:
		// Handle mouse clicks on tab headers when no overlay active
		if m.activeOverlay == overlayNone {
			mouse := msg.Mouse()
			// Check if click is in the tab header area (first line)
			if mouse.Y < tabHeaderHeight {
				m.tabManager.HandleTabHeaderClick(mouse.X)
				return m, nil
			}
		}
		// Forward to active tab
		if cmd := m.tabManager.Update(msg); cmd != nil {
			return m, cmd
		}
		return m, nil

	case tea.KeyPressMsg:
		// Handle hotkey detection first
		if hotkeyCmd := hotkey.HandleKeyMsg(msg); hotkeyCmd != nil {
			return m, hotkeyCmd
		}

		// Priority handling for overlay dismissal
		if m.activeOverlay != overlayNone && msg.String() == "esc" {
			m = m.clearOverlay()
			return m, nil
		}

		// Handle number key selection in status overlay for agent restoration
		if m.activeOverlay == overlayStatus {
			switch msg.String() {
			case "1", "2", "3", "4", "5", "6", "7", "8", "9":
				agentIndex, _ := strconv.Atoi(msg.String())
				agentIndex-- // Convert to 0-based index
				
				// Use snapshot stored when overlay was created
				if agentIndex >= 0 && agentIndex < len(m.statusRunningAgents) && agentIndex < 9 {
					selectedAgent := m.statusRunningAgents[agentIndex]
					m = m.clearOverlay()
					
					// Validate agent is still running before creating/focusing tab
					currentAgents := m.manager.List()
					agentStillRunning := false
					for _, currentAgent := range currentAgents {
						if currentAgent.ID == selectedAgent.ID && currentAgent.Status == agent.StatusRunning {
							agentStillRunning = true
							break
						}
					}
					
					if agentStillRunning {
						m.tabManager.RestoreOrFocusAgentTab(selectedAgent.ID, m.manager, m.styles)
						m.knownAgents[selectedAgent.ID] = true
					} else {
						m = m.appendActivity(m.styles.Warning.Render("Agent no longer running"))
					}
					return m, nil
				}
				// Invalid selection - just ignore
				return m, nil
			}
		}

		// Block other input when overlay is active
		if m.activeOverlay != overlayNone {
			return m, nil
		}

		if m.confirmingExit {
			input := strings.ToLower(strings.TrimSpace(msg.String()))
			switch input {
			case "y", "yes":
				m.manager.StopAll()
				m.watcher.Stop()
				m.quitting = true
				return m, tea.Quit
			default:
				m.confirmingExit = false
				m = m.appendActivity(m.styles.Warning.Render("Exit cancelled."))
				return m, nil
			}
		}

		switch msg.String() {
		case "ctrl+c":
			return m.tryExit()
		case "f2":
			// Toggle between main and agent tabs
			m.tabManager.ToggleView()
			return m, nil
		case "[":
			// Previous tab
			m.tabManager.PreviousTab()
			return m, nil
		case "]":
			// Next tab
			m.tabManager.NextTab()
			return m, nil
		case "ctrl+w":
			// Close current tab (if closable)
			m.tabManager.CloseCurrentTab()
			return m, nil
		case "up", "down", "pgup", "pgdown", "home", "end":
			// Handle console scroll events when in main tab
			activeTab := m.tabManager.GetActiveTab()
			if activeTab != nil && activeTab.Type() == TabTypeMain {
				switch msg.String() {
				case "up":
					m.consoleViewport.ScrollUp(1)
				case "down":
					m.consoleViewport.ScrollDown(1)
				case "pgup":
					m.consoleViewport.HalfPageUp()
				case "pgdown":
					m.consoleViewport.HalfPageDown()
				case "home":
					m.consoleViewport.GotoTop()
				case "end":
					m.consoleViewport.GotoBottom()
				}
				return m, nil
			}
			// Forward to tab manager for agent tabs
			if cmd := m.tabManager.Update(msg); cmd != nil {
				return m, cmd
			}
			return m, nil
		case "enter":
			// Only handle enter in main tab (console view)
			activeTab := m.tabManager.GetActiveTab()
			if activeTab == nil || activeTab.Type() != TabTypeMain {
				return m, nil
			}
			input := strings.TrimSpace(m.input.Value())
			m.input.SetValue("")
			if input == "" {
				return m, nil
			}
			return m.executeCommand(input)
		default:
			// Forward key messages to active tab
			activeTab := m.tabManager.GetActiveTab()
			if activeTab != nil && activeTab.Type() == TabTypeAgent {
				if cmd := m.tabManager.Update(msg); cmd != nil {
					return m, cmd
				}
			}
		}
	}

	// Only update input when no overlay is active and in main tab
	activeTab := m.tabManager.GetActiveTab()
	if m.activeOverlay == overlayNone && activeTab != nil && activeTab.Type() == TabTypeMain {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) activateOverlay(overlay overlayType, title string, content []string) model {
	// Limit content size to prevent memory issues
	if len(content) > maxOverlayLines {
		content = content[len(content)-maxOverlayLines:]
	}

	m.activeOverlay = overlay
	m.overlayContent = overlayContent{
		title:   title,
		content: append(content, "", "Press ESC to close"),
	}
	return m
}

func (m model) clearOverlay() model {
	m.activeOverlay = overlayNone
	m.overlayContent = overlayContent{} // Clear content to prevent memory accumulation
	return m
}

func (m model) renderBaseView() string {
	// Reserve 2 lines for prompt area (separator + input); tab header accounted for in viewport height
	activityHeight := m.height - 2 - tabHeaderHeight
	if activityHeight < 1 {
		activityHeight = 1
	}

	// Use console viewport for scrollable content
	activity := m.consoleViewport.View()

	separator := m.styles.Separator.Render(strings.Repeat("─", m.width))

	// Create theme label
	themeLabel := m.styles.ThemeLabel.Render(fmt.Sprintf("theme: %s", m.config.Theme))
	themeLabelWidth := lipgloss.Width(themeLabel)

	// If the terminal is too narrow to fit both prompt + theme label, hide the theme label.
	if m.width > 0 && themeLabelWidth+20 > m.width {
		themeLabel = ""
		themeLabelWidth = 0
	}

	// Calculate available width for prompt (minimum 20 columns when possible)
	promptWidth := m.width - themeLabelWidth
	if m.width >= 20 && promptWidth < 20 {
		promptWidth = 20
	}
	if promptWidth < 1 {
		promptWidth = 1
	}
	// Create prompt with adjusted width
	promptInput := m.input.View()
	prompt := m.styles.Prompt.Width(promptWidth).Render(promptInput)

	// Join prompt and theme label horizontally
	promptLine := lipgloss.JoinHorizontal(lipgloss.Top, prompt, themeLabel)

	return m.styles.Activity.Render(activity) + "\n" + separator + "\n" + promptLine
}

func (m model) View() tea.View {
	if m.quitting {
		return tea.NewView("Goodbye!\n")
	}

	// Wait for window size before rendering full layout
	if m.height == 0 {
		v := tea.NewView(m.input.View())
		v.AltScreen = true
		return v
	}

	// Render tab headers at the top
	tabHeaders := m.tabManager.RenderTabHeaders(m.width, m.styles)
	
	base := m.renderBaseView()

	// Render active tab content (use base view directly for main tab)
	var content string
	activeTab := m.tabManager.GetActiveTab()
	if activeTab != nil && activeTab.Type() != TabTypeMain {
		content = activeTab.View()
	} else {
		content = base
	}

	// Combine tab headers with content
	content = tabHeaders + "\n" + content

	// Compose overlay if active (overlays work on any view)
	if m.activeOverlay != overlayNone {
		overlay := m.renderOverlay()
		content = m.layerOverlay(content, overlay)
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m model) readNewLogLines() ([]string, int64) {
	info, err := m.logReader.Stat()
	if err != nil {
		return nil, m.lastLogPos
	}

	size := info.Size()
	if size <= m.lastLogPos {
		return nil, m.lastLogPos
	}

	buf := make([]byte, size-m.lastLogPos)
	n, err := m.logReader.ReadAt(buf, m.lastLogPos)
	if err != nil && err != io.EOF {
		return nil, m.lastLogPos
	}
	newPos := m.lastLogPos + int64(n)

	text := strings.TrimRight(string(buf[:n]), "\n")
	if text == "" {
		return nil, newPos
	}
	return strings.Split(text, "\n"), newPos
}

func (m model) renderOverlay() string {
	// Calculate overlay dimensions (60% of screen, centered)
	m.overlayWidth = int(float64(m.width) * 0.6)
	m.overlayHeight = int(float64(m.height) * 0.6)

	// Ensure minimum dimensions
	if m.overlayWidth < 40 {
		m.overlayWidth = 40
	}
	if m.overlayHeight < 10 {
		m.overlayHeight = 10
	}

	// Ensure overlay doesn't exceed screen bounds
	if m.overlayWidth >= m.width {
		m.overlayWidth = m.width - 2
	}
	if m.overlayHeight >= m.height {
		m.overlayHeight = m.height - 2
	}

	// Create overlay content
	title := m.styles.OverlayTitle.Render(m.overlayContent.title)

	contentHeight := m.overlayHeight - 4 // Account for border + title + padding
	if contentHeight < 1 {
		contentHeight = 1
	}

	content := m.overlayContent.content
	if len(content) > contentHeight {
		content = content[len(content)-contentHeight:]
	}

	// Pad content to fill overlay
	for len(content) < contentHeight {
		content = append(content, "")
	}

	// Trim content lines to fit within overlay width
	maxContentWidth := m.overlayWidth - 6 // Account for border + padding
	if maxContentWidth < 1 {
		maxContentWidth = 1
	}
	truncateStyle := lipgloss.NewStyle().Width(maxContentWidth)
	for i, line := range content {
		if lipgloss.Width(line) > maxContentWidth {
			content[i] = truncateStyle.Render(line)
		}
	}

	contentStr := strings.Join(content, "\n")

	// Apply overlay styling with proper content rendering
	overlayContent := lipgloss.JoinVertical(lipgloss.Left, title, "", m.styles.OverlayContent.Render(contentStr))

	return m.styles.OverlayBorder.
		Width(m.overlayWidth - 4). // Account for border
		Height(m.overlayHeight - 2).
		Render(overlayContent)
}

func (m model) layerOverlay(base, overlay string) string {
	// Center overlay on base view
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")

	overlayW := 0
	for _, l := range overlayLines {
		if w := lipgloss.Width(l); w > overlayW {
			overlayW = w
		}
	}

	startRow := (m.height - len(overlayLines)) / 2
	startCol := (m.width - overlayW) / 2
	// Ensure overlay stays within bounds
	if startRow < 0 {
		startRow = 0
	}
	if startCol < 0 {
		startCol = 0
	}
	if startRow+len(overlayLines) > len(baseLines) {
		startRow = len(baseLines) - len(overlayLines)
		if startRow < 0 {
			startRow = 0
		}
	}

	// Create result with same length as base
	result := make([]string, len(baseLines))
	copy(result, baseLines)

	// Overlay the content
	for i, overlayLine := range overlayLines {
		targetRow := startRow + i
		if targetRow >= 0 && targetRow < len(result) {
			baseLine := result[targetRow]
			overlayWidth := lipgloss.Width(overlayLine)

			// Pad base line if needed
			if len(baseLine) < startCol {
				baseLine += strings.Repeat(" ", startCol-len(baseLine))
			}

			// Calculate portions of base line
			beforeOverlay := ""
			if startCol > 0 && len(baseLine) > 0 {
				end := startCol
				if end > len(baseLine) {
					end = len(baseLine)
				}
				beforeOverlay = baseLine[:end]
			}

			afterOverlay := ""
			afterStart := startCol + overlayWidth
			if afterStart < len(baseLine) {
				afterOverlay = baseLine[afterStart:]
			}

			result[targetRow] = beforeOverlay + overlayLine + afterOverlay
		}
	}

	return strings.Join(result, "\n")
}

func (m model) tryExit() (model, tea.Cmd) {
	agents := m.manager.List()
	running := 0
	for _, a := range agents {
		if a.Status == agent.StatusRunning {
			running++
		}
	}

	if running > 0 {
		m.confirmingExit = true
		m = m.appendActivity(m.styles.Warning.Render(fmt.Sprintf("There are %d agents still running. Stop all and exit? (y/N)", running)))
		return m, nil
	}

	m.watcher.Stop()
	m.quitting = true
	return m, tea.Quit
}

func (m model) executeCommand(input string) (model, tea.Cmd) {
	parts := strings.Fields(input)
	cmd := parts[0]

	switch strings.ToLower(cmd) {
	case "watch":
		if len(parts) < 2 {
			m = m.appendActivity(m.styles.Error.Render("Usage: watch start|stop"))
			return m, nil
		}
		return m.handleWatch(parts[1])
	case "status":
		return m.handleStatus()
	case "stop":
		if len(parts) < 2 {
			m = m.appendActivity(m.styles.Error.Render("Usage: stop <issue-number>"))
			return m, nil
		}
		return m.handleStop(parts[1])
	case "plan":
		description := ""
		if len(parts) > 1 {
			description = strings.Join(parts[1:], " ")
		}
		return m.handlePlan(description)
	case "exit":
		return m.tryExit()
	case "about":
		return m.handleAbout()
	case "help":
		return m.handleHelp()
	case "theme":
		args := []string{}
		if len(parts) > 1 {
			args = parts[1:]
		}
		return m.handleTheme(args)
	default:
		m = m.appendActivity(m.styles.Error.Render(fmt.Sprintf("Unknown command: %s", cmd)))
		return m, nil
	}
}

// Run starts the TUI, redirecting log output to a file.
func (m model) updateAgentTabs() model {
	agents := m.manager.List()

	// Track current agent IDs
	currentAgents := make(map[string]bool)
	for _, ag := range agents {
		currentAgents[ag.ID] = true

		// Create tab for new agents when they start (no auto-switch)
		if !m.knownAgents[ag.ID] {
			agentTab := NewAgentTab(ag.ID, m.manager, m.styles)
			m.tabManager.AddTab(agentTab)
			m.knownAgents[ag.ID] = true
		}
	}

	// Prune knownAgents entries for agents no longer in the manager
	for id := range m.knownAgents {
		if !currentAgents[id] {
			delete(m.knownAgents, id)
		}
	}

	return m
}

func Run(w *watcher.Watcher, m *agent.Manager, cfg *config.Config) error {
	logPath := ".kiro-krew/kiro-krew.log"
	if err := os.MkdirAll(".kiro-krew", 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFile.Close()

	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags)

	logReader, err := os.Open(logPath)
	if err != nil {
		return fmt.Errorf("failed to open log reader: %w", err)
	}
	defer logReader.Close()

	// Seek to end so we only show new entries
	info, err := logReader.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat log file: %w", err)
	}
	startPos := info.Size()

	mdl := newModel(w, m, cfg, logFile, logReader)
	mdl.lastLogPos = startPos

	// Setup cleanup on exit
	defer func() {
		if err := mdl.sessionManager.CleanupOnExit(); err != nil {
			log.Printf("Session cleanup error: %v", err)
		}
	}()

	p := tea.NewProgram(mdl)
	_, err = p.Run()
	return err
}
