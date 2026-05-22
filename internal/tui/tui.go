package tui

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jbrinkman/kiro-krew/internal/agent"
	"github.com/jbrinkman/kiro-krew/internal/watcher"
)

var (
	promptStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	activityStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

type logMsg string

type tickMsg struct{}

type model struct {
	watcher        *watcher.Watcher
	manager        *agent.Manager
	input          textinput.Model
	activityLines  []string
	width          int
	height         int
	confirmingExit bool
	logFile        *os.File
	logReader      *os.File
	lastLogPos     int64
	quitting       bool
}

func newModel(w *watcher.Watcher, m *agent.Manager, logFile *os.File, logReader *os.File) model {
	ti := textinput.New()
	ti.Prompt = "kiro-krew> "
	ti.Focus()

	return model{
		watcher:   w,
		manager:   m,
		input:     ti,
		logFile:   logFile,
		logReader: logReader,
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

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case execDoneMsg:
		if msg.err != nil {
			m.activityLines = append(m.activityLines, fmt.Sprintf("Process exited with error: %v", msg.err))
		}
		return m, nil

	case tickMsg:
		newLines := m.readNewLogLines()
		if len(newLines) > 0 {
			m.activityLines = append(m.activityLines, newLines...)
		}
		return m, m.tickCmd()

	case tea.KeyMsg:
		if m.confirmingExit {
			switch msg.String() {
			case "y", "Y":
				m.manager.StopAll()
				m.watcher.Stop()
				m.quitting = true
				return m, tea.Quit
			default:
				m.confirmingExit = false
				m.activityLines = append(m.activityLines, "Exit cancelled.")
				return m, nil
			}
		}

		switch msg.Type {
		case tea.KeyCtrlC:
			return m.tryExit()
		case tea.KeyEnter:
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

func (m model) View() string {
	if m.quitting {
		return "Goodbye!\n"
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

	// Pad activity pane
	activity := strings.Join(lines, "\n")
	lineCount := len(lines)
	if lineCount < activityHeight {
		activity += strings.Repeat("\n", activityHeight-lineCount)
	}

	separator := promptStyle.Render(strings.Repeat("─", m.width))
	prompt := m.input.View()

	return activityStyle.Render(activity) + "\n" + separator + "\n" + prompt
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
		m.activityLines = append(m.activityLines, fmt.Sprintf("There are %d agents still running. Stop all and exit? (y/N)", running))
		return m, nil
	}

	m.watcher.Stop()
	m.quitting = true
	return m, tea.Quit
}

func (m model) executeCommand(input string) (model, tea.Cmd) {
	parts := strings.Fields(input)
	cmd := parts[0]

	switch cmd {
	case "watch":
		if len(parts) < 2 {
			m.activityLines = append(m.activityLines, "Usage: watch start|stop")
			return m, nil
		}
		return m.handleWatch(parts[1])
	case "status":
		return m.handleStatus()
	case "stop":
		if len(parts) < 2 {
			m.activityLines = append(m.activityLines, "Usage: stop <issue-number>")
			return m, nil
		}
		return m.handleStop(parts[1])
	case "exit":
		return m.tryExit()
	case "help":
		return m.handleHelp()
	default:
		m.activityLines = append(m.activityLines, fmt.Sprintf("Unknown command: %s", cmd))
		return m, nil
	}
}

// Run starts the TUI, redirecting log output to a file.
func Run(w *watcher.Watcher, m *agent.Manager) error {
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

	mdl := newModel(w, m, logFile, logReader)
	mdl.lastLogPos = startPos

	p := tea.NewProgram(mdl, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
