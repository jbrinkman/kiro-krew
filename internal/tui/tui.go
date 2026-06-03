package tui

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

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

type consoleState struct {
	inputValue    string
	activityLines []string
}

type model struct {
	watcher            *watcher.Watcher
	manager            *agent.Manager
	sessionManager     *session.SessionManager
	config             *config.Config
	styles             *Styles
	input              textinput.Model
	activityLines      []string
	maxActivityLines   int
	width              int
	height             int
	confirmingExit     bool
	logFile            *os.File
	logReader          *os.File
	lastLogPos         int64
	quitting           bool
	currentMode        session.SessionType
	consoleState       *consoleState
	activePlanningSession *session.PlanningSession
}

func newModel(w *watcher.Watcher, m *agent.Manager, cfg *config.Config, logFile *os.File, logReader *os.File) model {
	ti := textinput.New()
	ti.Prompt = "kiro-krew> "
	ti.Focus()

	theme := cfg.LoadedTheme

	return model{
		watcher:          w,
		manager:          m,
		sessionManager:   session.NewSessionManager(),
		config:           cfg,
		styles:           NewStyles(theme),
		input:            ti,
		logFile:          logFile,
		logReader:        logReader,
		maxActivityLines: cfg.MaxActivityLines,
		currentMode:      session.Console,
		consoleState: &consoleState{
			inputValue:    "",
			activityLines: make([]string, 0),
		},
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
	return m
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case execDoneMsg:
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
		if msg.err != nil {
			m = m.appendActivity(m.styles.Warning.Render("Update Status: Unable to check for updates"))
			m = m.appendActivity(m.styles.Error.Render(fmt.Sprintf("  Error: %v", msg.err)))
		} else {
			current := version.Version
			latest := msg.release.TagName
			if current == "dev" {
				m = m.appendActivity(m.styles.Warning.Render("Update Status: Development build"))
			} else {
				// Ensure "v" prefix for semver comparison
				if !strings.HasPrefix(current, "v") {
					current = "v" + current
				}
				if !strings.HasPrefix(latest, "v") {
					latest = "v" + latest
				}
				if !semver.IsValid(current) || !semver.IsValid(latest) {
					m = m.appendActivity(m.styles.Warning.Render("Update Status: Unable to compare versions (non-semver format)"))
					m = m.appendActivity(fmt.Sprintf("  Current: %s, Latest: %s", version.Version, msg.release.TagName))
				} else if semver.Compare(current, latest) < 0 {
					m = m.appendActivity(m.styles.Warning.Render("Update Status: Update available"))
					m = m.appendActivity(fmt.Sprintf("  Latest: %s (%s)", msg.release.TagName, msg.release.Name))
				} else {
					m = m.appendActivity(m.styles.Success.Render("Update Status: Up to date"))
				}
			}
		}
		return m, nil

	case tickMsg:
		newLines, newPos := m.readNewLogLines()
		if len(newLines) > 0 {
			m.lastLogPos = newPos
			m = m.appendActivity(newLines...)
		}
		return m, m.tickCmd()

	case planningHotkeyMsg:
		if m.currentMode == session.Planning {
			return m.switchToConsoleMode()
		}
		return m, nil

	case hotkey.HotkeyTriggeredMsg:
		if m.currentMode == session.Console {
			return m.switchToPlanningMode()
		} else if m.currentMode == session.Planning && m.activePlanningSession != nil {
			return m.switchToConsoleMode()
		}
		return m, nil

	case hotkey.HotkeyErrorMsg:
		m = m.appendActivity(m.styles.Error.Render(fmt.Sprintf("Hotkey error: %v", msg.Err)))
		return m, nil

	case tea.KeyPressMsg:
		// Handle hotkey detection first
		if hotkeyCmd := hotkey.HandleKeyMsg(msg); hotkeyCmd != nil {
			return m, hotkeyCmd
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
		case "enter":
			input := strings.TrimSpace(m.input.Value())
			m.input.SetValue("")
			if input == "" {
				return m, nil
			}
			return m.executeCommand(input)
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
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

	// Reserve 2 lines for prompt area (separator + input)
	activityHeight := m.height - 2
	if activityHeight < 1 {
		activityHeight = 1
	}

	// Get visible activity lines
	lines := m.activityLines
	if len(lines) > activityHeight {
		lines = lines[len(lines)-activityHeight:]
	}

	// Build activity pane padded to exactly activityHeight lines
	var activityBuilder strings.Builder
	for i := 0; i < activityHeight; i++ {
		if i < len(lines) {
			activityBuilder.WriteString(lines[i])
		}
		if i < activityHeight-1 {
			activityBuilder.WriteByte('\n')
		}
	}
	activity := activityBuilder.String()

	separator := m.styles.Separator.Render(strings.Repeat("─", m.width))
	prompt := m.styles.Prompt.Render(m.input.View())

	v := tea.NewView(m.styles.Activity.Render(activity) + "\n" + separator + "\n" + prompt)
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
