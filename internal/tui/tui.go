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
	"charm.land/lipgloss/v2"

	"github.com/jbrinkman/kiro-krew/internal/agent"
	"github.com/jbrinkman/kiro-krew/internal/config"
	"github.com/jbrinkman/kiro-krew/internal/watcher"
)

var (
	promptStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	activityStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

type logMsg string

type tickMsg struct{}

type model struct {
	watcher          *watcher.Watcher
	manager          *agent.Manager
	config           *config.Config
	input            textinput.Model
	activityLines    []string
	maxActivityLines int
	width            int
	height           int
	confirmingExit   bool
	logFile          *os.File
	logReader        *os.File
	lastLogPos       int64
	quitting         bool
}

func newModel(w *watcher.Watcher, m *agent.Manager, cfg *config.Config, logFile *os.File, logReader *os.File) model {
	ti := textinput.New()
	ti.Prompt = "kiro-krew> "
	ti.Focus()

	return model{
		watcher:          w,
		manager:          m,
		config:           cfg,
		input:            ti,
		logFile:          logFile,
		logReader:        logReader,
		maxActivityLines: cfg.MaxActivityLines,
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
			m = m.appendActivity(fmt.Sprintf("Planning session exited with error: %v", msg.err))
		} else {
			m = m.appendActivity("Planning session completed.")
		}
		m.input.Focus()
		return m, tea.Batch(textinput.Blink, tea.ClearScreen)

	case tickMsg:
		newLines := m.readNewLogLines()
		if len(newLines) > 0 {
			m = m.appendActivity(newLines...)
		}
		return m, m.tickCmd()

	case tea.KeyPressMsg:
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
				m = m.appendActivity("Exit cancelled.")
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

	separator := promptStyle.Render(strings.Repeat("─", m.width))
	prompt := m.input.View()

	v := tea.NewView(activityStyle.Render(activity) + "\n" + separator + "\n" + prompt)
	v.AltScreen = true
	return v
}

func (m model) readNewLogLines() []string {
	info, err := m.logReader.Stat()
	if err != nil {
		return nil
	}

	size := info.Size()
	if size <= m.lastLogPos {
		return nil
	}

	buf := make([]byte, size-m.lastLogPos)
	n, err := m.logReader.ReadAt(buf, m.lastLogPos)
	if err != nil && err != io.EOF {
		return nil
	}
	m.lastLogPos += int64(n)

	text := strings.TrimRight(string(buf[:n]), "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
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
		m = m.appendActivity(fmt.Sprintf("There are %d agents still running. Stop all and exit? (y/N)", running))
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
			m = m.appendActivity("Usage: watch start|stop")
			return m, nil
		}
		return m.handleWatch(parts[1])
	case "status":
		return m.handleStatus()
	case "stop":
		if len(parts) < 2 {
			m = m.appendActivity("Usage: stop <issue-number>")
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
	default:
		m = m.appendActivity(fmt.Sprintf("Unknown command: %s", cmd))
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

	p := tea.NewProgram(mdl)
	_, err = p.Run()
	return err
}
