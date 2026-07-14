package tui

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	clog "github.com/charmbracelet/log"

	"golang.org/x/mod/semver"

	"github.com/jbrinkman/kiro-krew/internal/agent"
	"github.com/jbrinkman/kiro-krew/internal/config"
	"github.com/jbrinkman/kiro-krew/internal/github"
	"github.com/jbrinkman/kiro-krew/internal/hotkey"
	"github.com/jbrinkman/kiro-krew/internal/logging"
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
	overlayLogs

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
	activeTabID   string
}

type model struct {
	watcher               *watcher.Watcher
	manager               *agent.Manager
	sessionManager        *session.SessionManager
	config                *config.Config
	styles                *Styles
	input                 *AutocompleteInput
	commandRegistry       *CommandRegistry
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
	tabManager *TabManager
	mainTab    *MainTab

	// Agent lifecycle tracking
	knownAgents         map[string]bool
	statusRunningAgents []*agent.Agent // Snapshot for deterministic number key selection

	// About dialog state
	aboutDialog *AboutDialog

	// Footer system
	footerManager *FooterManager

	// Logging system state
	loggingActive     bool
	activeLogTabID    string
	activeFileHandler *logging.FileHandler
}

func newModel(w *watcher.Watcher, m *agent.Manager, cfg *config.Config, logFile *os.File, logReader *os.File) model {
	theme := cfg.LoadedTheme
	styles := NewStyles(theme)

	// Create command registry and autocomplete input
	commandRegistry := NewCommandRegistry(m)
	autocompleteInput := NewAutocompleteInput(commandRegistry, styles)

	consoleViewport := viewport.New(viewport.WithWidth(80), viewport.WithHeight(24))
	// Disable built-in key bindings — we handle scrolling explicitly
	consoleViewport.KeyMap = viewport.KeyMap{}

	// Initialize tab system
	tabManager := NewTabManager()
	mainTab := NewMainTab()
	tabManager.AddTab(mainTab)

	// Initialize footer system
	footerManager := NewFooterManager(styles, cfg, autocompleteInput, tabManager)

	// Initialize logging subsystem (inactive state - no handlers attached)
	if err := logging.Initialize("info"); err != nil {
		log.Printf("Failed to initialize logging subsystem: %v", err)
	}

	return model{
		watcher:          w,
		manager:          m,
		sessionManager:   session.NewSessionManager(),
		config:           cfg,
		styles:           styles,
		input:            autocompleteInput,
		commandRegistry:  commandRegistry,
		consoleViewport:  consoleViewport,
		logFile:          logFile,
		logReader:        logReader,
		maxActivityLines: cfg.MaxActivityLines,
		currentMode:      session.Console,
		consoleState: &consoleState{
			inputValue:    "",
			activityLines: make([]string, 0),
		},
		tabManager:    tabManager,
		mainTab:       mainTab,
		knownAgents:   make(map[string]bool),
		aboutDialog:   NewAboutDialog(),
		footerManager: footerManager,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.input.Focus(), m.tickCmd())
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

		// Update footer manager dimensions
		m.footerManager.Resize(msg.Width, msg.Height)

		// Resize console viewport — account for tab header + footer system height
		footerHeight := m.footerManager.GetFooterHeight()
		activityHeight := m.height - footerHeight - tabHeaderHeight
		if activityHeight < 1 {
			activityHeight = 1
		}
		m.consoleViewport.SetWidth(msg.Width)
		m.consoleViewport.SetHeight(activityHeight)

		// Forward to tab manager with footer-aware resizing
		m.tabManager.ResizeForFooter(msg.Width, msg.Height, footerHeight)

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

			// Log the error for debugging
			log.Printf("Planning session error: %v", msg.err)
		} else {
			m = m.appendActivity(m.styles.Success.Render("Planning session completed successfully."))
		}

		// Clean up planning session tracking with comprehensive error handling
		if m.activePlanningSession != nil {
			// Attempt to properly save session state before cleanup
			if saveErr := m.activePlanningSession.SaveState(); saveErr != nil {
				m = m.appendActivity(m.styles.Warning.Render(fmt.Sprintf("Warning: Failed to save planning session state: %v", saveErr)))
				log.Printf("Planning session save error: %v", saveErr)
			}

			// Cleanup the session resources
			if cleanupErr := m.activePlanningSession.Cleanup(); cleanupErr != nil {
				m = m.appendActivity(m.styles.Warning.Render(fmt.Sprintf("Warning: Planning session cleanup had issues: %v", cleanupErr)))
				log.Printf("Planning session cleanup error: %v", cleanupErr)
			}

			m.activePlanningSession = nil
		}

		// Stop context tracking when exiting planning mode
		if m.footerManager != nil && m.footerManager.GetContextTracker() != nil {
			m.footerManager.GetContextTracker().StopPlanningSession()
		}

		m = m.restoreConsoleState()
		m.input.Focus()
		return m, tea.Batch(m.input.Focus(), tea.ClearScreen)

	case focusTransferMsg:
		// Handle focus coordination between planning tab and footer input
		activeTab := m.tabManager.GetActiveTab()
		if activeTab != nil && activeTab.Type() == TabTypePlanning {
			if msg.target == "footer" {
				// Focus footer input, blur any tab input
				m.input.SetFocus(true)
				return m, m.input.Focus()
			} else if msg.target == "message" {
				// Focus message input, blur footer input
				m.input.SetFocus(false)
				if pt, ok := activeTab.(*PlanningTab); ok {
					pt.SetFocusInput(true)
					return m, pt.RestoreFocus()
				}
			}
		}
		return m, nil

	case updateCheckMsg:
		updateLines := []string{}
		if msg.err != nil {
			// Check if error is ErrNoReleases - hide update status section entirely
			if errors.Is(msg.err, github.ErrNoReleases) {
				if m.activeOverlay == overlayAbout {
					// Hide update status section by passing empty slice
					m.aboutDialog.UpdateStatusLine([]string{})
					m.overlayContent.content = append(m.aboutDialog.GetFullContent(), "", "Press ESC to close")
				}
				// For console mode, don't add any activity lines
				return m, nil
			}

			// Other errors - show error message as before
			updateLines = append(updateLines,
				m.styles.Warning.Render("Update Status: Unable to check for updates"),
				m.styles.Error.Render(fmt.Sprintf("  Error: %v", msg.err)),
			)
		} else {
			// Check for empty/invalid TagName - treat as no releases
			if msg.release == nil || strings.TrimSpace(msg.release.TagName) == "" {
				if m.activeOverlay == overlayAbout {
					m.aboutDialog.UpdateStatusLine([]string{})
					m.overlayContent.content = append(m.aboutDialog.GetFullContent(), "", "Press ESC to close")
				}
				return m, nil
			}
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
			// Efficient partial update — only replace the status line
			m.aboutDialog.UpdateStatusLine(updateLines)
			m.overlayContent.content = append(m.aboutDialog.GetFullContent(), "", "Press ESC to close")
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

	case tea.MouseMotionMsg:
		// Handle mouse motion for tab hover effects when no overlay active
		if m.activeOverlay == overlayNone {
			mouse := msg.Mouse()
			// Only process hover when mouse is within tab header area
			if mouse.Y < tabHeaderHeight {
				m.tabManager.HandleTabHeaderHover(mouse.X)
			} else {
				m.tabManager.ClearHover()
			}
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

		// Priority handling for focus transfer in planning tabs
		if msg.String() == "esc" {
			activeTab := m.tabManager.GetActiveTab()
			if activeTab != nil && activeTab.Type() == TabTypePlanning {
				// When footer has focus on planning tab and Esc is pressed, transfer focus to message input
				if m.input.Focused() {
					m.input.SetFocus(false)
					return m, func() tea.Msg {
						return focusTransferMsg{target: "message"}
					}
				}
			}
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
				// Perform comprehensive cleanup
				m = m.performExitCleanup()
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
			tabCount := len(m.tabManager.GetTabs())
			if tabCount > 1 {
				prev := (m.tabManager.GetActiveTabIndex() - 1 + tabCount) % tabCount
				var cmd tea.Cmd
				m, cmd = m.switchActiveTab(prev)
				return m, cmd
			}
			return m, nil
		case "]":
			// Next tab
			tabCount := len(m.tabManager.GetTabs())
			if tabCount > 1 {
				next := (m.tabManager.GetActiveTabIndex() + 1) % tabCount
				var cmd tea.Cmd
				m, cmd = m.switchActiveTab(next)
				return m, cmd
			}
			return m, nil
		case "ctrl+w":
			// Close current tab (if closable)
			activeTab := m.tabManager.GetActiveTab()
			if activeTab != nil && activeTab.Type() == TabTypeLog {
				// Deactivate logging before closing the tab
				if err := m.deactivateLogging(); err != nil {
					m = m.appendActivity(m.styles.Warning.Render(fmt.Sprintf("Warning during logging deactivation: %v", err)))
				}
			}
			m.tabManager.CloseCurrentTab()
			return m, nil
		case "up", "down", "pgup", "pgdown", "home", "end":
			// Handle console scroll events when in main tab
			activeTab := m.tabManager.GetActiveTab()
			if activeTab != nil && activeTab.Type() == TabTypeMain {
				// Route up/down to textinput when suggestions are active
				if (msg.String() == "up" || msg.String() == "down") && m.input.HasMatchedSuggestions() {
					var cmd tea.Cmd
					m.input, cmd = m.input.Update(msg)
					return m, cmd
				}

				// Handle console scrolling
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
			// Forward to tab manager for non-main tabs
			// But if footer has focus in planning tab, route up/down to footer for dropdown
			if activeTab != nil && activeTab.Type() == TabTypePlanning && m.input.Focused() {
				if msg.String() == "up" || msg.String() == "down" {
					var cmd tea.Cmd
					m.input, cmd = m.input.Update(msg)
					return m, cmd
				}
			}
			// Blur footer input during viewport navigation to prevent cursor artifacts
			// when the planning tab enters scroll mode.
			if activeTab != nil && activeTab.Type() == TabTypePlanning {
				switch msg.String() {
				case "pgup", "pgdown", "home", "end":
					m.input.SetFocus(false)
				}
			}
			if cmd := m.tabManager.Update(msg); cmd != nil {
				return m, cmd
			}
			return m, nil
		case "tab":
			// Handle tab completion in footer input
			activeTab := m.tabManager.GetActiveTab()
			if activeTab != nil && activeTab.Type() == TabTypeMain {
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
			// Tab completion also works in footer when planning tab has footer focused
			if activeTab != nil && activeTab.Type() == TabTypePlanning && m.input.Focused() {
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
			// Forward to active tab for other handling
			if activeTab != nil {
				if cmd := m.tabManager.Update(msg); cmd != nil {
					return m, cmd
				}
			}
			return m, nil
		case "enter":
			// Handle enter in main tab (console view), forward to other tabs
			activeTab := m.tabManager.GetActiveTab()
			if activeTab == nil || activeTab.Type() != TabTypeMain {
				// If footer has focus in planning tab, route enter to footer for command execution
				if activeTab != nil && activeTab.Type() == TabTypePlanning && m.input.Focused() {
					var cmd tea.Cmd
					m.input, cmd = m.input.Update(msg)

					input := strings.TrimSpace(m.input.Value())
					m.input.SetValue("")
					if input == "" {
						return m, cmd
					}

					return m.executeCommand(input)
				}
				// Forward to active tab (e.g., planning tab message sending)
				if activeTab != nil {
					if cmd := m.tabManager.Update(msg); cmd != nil {
						return m, cmd
					}
				}
				return m, nil
			}

			// Let autocomplete handle enter first (which will pass through)
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)

			input := strings.TrimSpace(m.input.Value())
			m.input.SetValue("")
			if input == "" {
				return m, cmd
			}

			return m.executeCommand(input)
		default:
			// Forward key messages to active tab - removed TabTypeAgent restriction
			activeTab := m.tabManager.GetActiveTab()
			if activeTab != nil {
				if cmd := m.tabManager.Update(msg); cmd != nil {
					return m, cmd
				}
			}
		}
	}

	// Update input when no overlay is active - removed TabTypeMain restriction
	activeTab := m.tabManager.GetActiveTab()
	if m.activeOverlay == overlayNone && activeTab != nil {
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
	// Calculate activity height accounting for footer and tab header
	footerHeight := m.footerManager.GetFooterHeight()
	activityHeight := m.height - footerHeight - tabHeaderHeight
	if activityHeight < 1 {
		activityHeight = 1
	}

	// Update viewport height for the available space
	m.consoleViewport.SetHeight(activityHeight)
	activity := m.consoleViewport.View()

	// Use unified rendering system for consistent footer display
	return m.renderTabContentWithFooter(m.styles.Activity.Render(activity), TabTypeMain)
}

// renderTabContentWithFooter creates a unified rendering method that combines tab content with footer
func (m model) renderTabContentWithFooter(tabContent string, tabType TabType) string {
	// Render footer using the footer system
	footer := m.footerManager.RenderWithSeparator(tabType)

	// Normalize tab content by removing trailing newlines
	normalizedContent := strings.TrimRight(tabContent, "\n")

	// When content is empty, return footer directly without a separator to avoid
	// an unnecessary leading blank line.
	if normalizedContent == "" {
		return footer
	}

	// Compose the complete view with consistent newline separation
	return normalizedContent + "\n" + footer
}

func (m model) View() tea.View {
	if m.quitting {
		v := tea.NewView("Goodbye!\n")
		v.MouseMode = tea.MouseModeAllMotion
		return v
	}

	// Wait for window size before rendering full layout
	if m.height == 0 {
		v := tea.NewView(m.input.View())
		v.AltScreen = true
		v.MouseMode = tea.MouseModeAllMotion
		return v
	}

	// Render tab headers at the top
	tabHeaders := m.tabManager.RenderTabHeaders(m.width, m.styles)

	// Render active tab content using unified rendering system
	var content string
	activeTab := m.tabManager.GetActiveTab()
	if activeTab != nil && activeTab.Type() != TabTypeMain {
		// For non-main tabs, get their content and apply unified rendering with footer
		tabContent := activeTab.View()
		content = m.renderTabContentWithFooter(tabContent, activeTab.Type())
	} else {
		// For main tab, use the existing base view (which now uses unified rendering internally)
		content = m.renderBaseView()
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
	v.MouseMode = tea.MouseModeAllMotion
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

	// Count active planning tabs
	activePlanningTabs := 0
	for _, tab := range m.tabManager.GetTabs() {
		if planningTab, ok := tab.(*PlanningTab); ok && planningTab.IsActive() {
			activePlanningTabs++
		}
	}

	if running > 0 || activePlanningTabs > 0 {
		m.confirmingExit = true
		if running > 0 && activePlanningTabs > 0 {
			m = m.appendActivity(m.styles.Warning.Render(fmt.Sprintf("There are %d agents running and %d active planning sessions. Stop all and exit? (y/N)", running, activePlanningTabs)))
		} else if running > 0 {
			m = m.appendActivity(m.styles.Warning.Render(fmt.Sprintf("There are %d agents still running. Stop all and exit? (y/N)", running)))
		} else {
			m = m.appendActivity(m.styles.Warning.Render(fmt.Sprintf("There are %d active planning sessions. Close all and exit? (y/N)", activePlanningTabs)))
		}
		return m, nil
	}

	// Perform cleanup before exit
	m = m.performExitCleanup()
	m.quitting = true
	return m, tea.Quit
}

// performExitCleanup performs comprehensive cleanup when exiting
func (m model) performExitCleanup() model {
	cleanupErrors := []string{}

	// Stop all agents
	m.manager.StopAll()

	// Stop watcher
	m.watcher.Stop()

	// Cleanup all planning tabs
	for _, tab := range m.tabManager.GetTabs() {
		if planningTab, ok := tab.(*PlanningTab); ok {
			// Force save session state before cleanup
			planningTab.SaveSession()

			// Close tab (includes ACP client cleanup)
			planningTab.Close()
		}
	}

	// Cleanup planning sessions (remove orphaned/completed)
	if err := m.tabManager.CleanupSessionsOnExit(); err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Sprintf("Planning session cleanup warning: %v", err))
	}

	// Stop context tracking
	if m.footerManager != nil && m.footerManager.GetContextTracker() != nil {
		m.footerManager.GetContextTracker().StopPlanningSession()
	}

	// Session manager cleanup
	if err := m.sessionManager.CleanupOnExit(); err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Sprintf("Session cleanup warning: %v", err))
	}

	// Report cleanup issues (non-blocking)
	if len(cleanupErrors) > 0 {
		m = m.appendActivity(m.styles.Warning.Render("Cleanup warnings:"))
		for _, err := range cleanupErrors {
			m = m.appendActivity(m.styles.Warning.Render(fmt.Sprintf("  %s", err)))
		}
	}

	return m
}

// switchActiveTab switches to a tab by index and updates context tracking
func (m model) switchActiveTab(index int) (model, tea.Cmd) {
	m.tabManager.SetActiveTab(index)
	// Update context tracker and focus based on active tab type
	if activeTab := m.tabManager.GetActiveTab(); activeTab != nil {
		if activeTab.Type() == TabTypePlanning {
			if !m.footerManager.GetContextTracker().IsActive() {
				m.footerManager.GetContextTracker().StartPlanningSession("claude-sonnet-4")
			}
			// Blur footer input so only the planning tab message input shows a cursor
			m.input.SetFocus(false)
			// Restore the planning tab's preserved focus state
			if pt, ok := activeTab.(*PlanningTab); ok {
				return m, pt.RestoreFocus()
			}
		} else {
			if m.footerManager.GetContextTracker().IsActive() {
				m.footerManager.GetContextTracker().StopPlanningSession()
			}
			// Restore footer input focus for non-planning tabs
			m.input.SetFocus(true)
		}
	}
	return m, nil
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
		// Check if this is "plan classic" command
		if len(parts) >= 2 && strings.ToLower(parts[1]) == "classic" {
			// Extract description after "classic"
			description := ""
			if len(parts) > 2 {
				description = strings.Join(parts[2:], " ")
			}
			return m.handlePlanClassic(description)
		}

		// Regular plan command - uses ACP-based planning tabs
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
	case "logs":
		return m.handleLogs()
	case "log":
		args := []string{}
		if len(parts) > 1 {
			args = parts[1:]
		}
		return m.handleLog(args)
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

// activateLogging activates the logging subsystem when a log viewer tab opens
func (m *model) activateLogging(logTab *LogTab) error {
	if m.loggingActive {
		return fmt.Errorf("logging already active")
	}

	// Get the ring buffer from the log tab
	ringBuffer := logTab.GetRingBuffer()
	if ringBuffer == nil {
		return fmt.Errorf("log tab has no ring buffer")
	}

	// Create file handler with configuration
	fileConfig := logging.FileOutputConfig{
		LogDir:        m.config.Logging.LogDir,
		MaxFileSizeMB: m.config.Logging.MaxFileSizeMB,
	}
	fileHandler, err := logging.NewFileHandler(fileConfig)
	if err != nil {
		return fmt.Errorf("failed to create file handler: %w", err)
	}

	// Create a custom writer that writes to both ring buffer and file handler
	multiWriter := &loggingMultiWriter{
		ringBuffer:  ringBuffer,
		fileHandler: fileHandler,
	}

	// Activate logging with the multi-writer
	logging.Activate(multiWriter)

	// Set the configured log level
	if err := logging.SetLevel(logTab.level); err != nil {
		return fmt.Errorf("failed to set log level: %w", err)
	}

	// Store state
	m.loggingActive = true
	m.activeLogTabID = logTab.ID()
	m.activeFileHandler = fileHandler

	log.Printf("Logging activated: level=%s, buffer_size=%d", logTab.level, logTab.bufferSize)
	return nil
}

// deactivateLogging deactivates the logging subsystem when the log viewer tab closes
func (m *model) deactivateLogging() error {
	if !m.loggingActive {
		return nil // Already inactive
	}

	// Deactivate the logger (removes handlers)
	logging.Deactivate()

	// Close the file handler
	if m.activeFileHandler != nil {
		if err := m.activeFileHandler.Close(); err != nil {
			log.Printf("Error closing file handler: %v", err)
		}
		m.activeFileHandler = nil
	}

	// Clear state
	m.loggingActive = false
	m.activeLogTabID = ""

	log.Printf("Logging deactivated")
	return nil
}

// loggingMultiWriter writes to both ring buffer and file handler
type loggingMultiWriter struct {
	ringBuffer  *logging.RingBuffer
	fileHandler *logging.FileHandler
}

func (lmw *loggingMultiWriter) Write(p []byte) (n int, err error) {
	// Parse the log entry from the formatted output
	// The charmbracelet/log library writes formatted strings
	// We need to extract level and message and write to ring buffer

	// Write to file handler first
	n, err = lmw.fileHandler.Write(p)
	if err != nil {
		return n, err
	}

	// For ring buffer, we need to parse the log entry
	// This is a simplified approach - just add the raw message
	// In a production system, you'd parse the structured log format
	message := string(p)
	if len(message) > 0 && message[len(message)-1] == '\n' {
		message = message[:len(message)-1]
	}

	// Add to ring buffer with INFO level (we can't easily extract level from formatted output)
	// This is a limitation of using charmbracelet/log's formatted output
	// A better approach would be to implement a custom Handler that writes structured data
	lmw.ringBuffer.Add(clog.InfoLevel, message)

	return n, nil
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
