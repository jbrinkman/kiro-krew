package tui

import (
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/creack/pty"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// modalOutputMsg delivers new output from the subprocess PTY.
type modalOutputMsg struct {
	data string
}

// modalDoneMsg signals the subprocess has exited.
type modalDoneMsg struct {
	err error
}

// modal represents a modal overlay running a subprocess in a PTY.
type modal struct {
	lines  []string
	ptmx   *os.File
	cmd    *exec.Cmd
	mu     sync.Mutex
	width  int
	height int
	scroll int // lines scrolled from bottom
}

var (
	modalBorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("6")).
		Padding(0, 1)
	modalTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Bold(true)
)

// newModal starts a subprocess in a PTY and returns the modal state.
func newModal(name string, args []string, width, height int) (*modal, tea.Cmd) {
	m := &modal{
		width:  width,
		height: height,
	}

	m.cmd = exec.Command(name, args...)

	ptmx, err := pty.Start(m.cmd)
	if err != nil {
		return m, func() tea.Msg { return modalDoneMsg{err: err} }
	}
	m.ptmx = ptmx

	// Set PTY size to match modal content area
	cw, ch := m.contentSize()
	_ = pty.Setsize(ptmx, &pty.Winsize{Rows: uint16(ch), Cols: uint16(cw)})

	return m, m.readOutput()
}

// contentSize returns the usable content area inside the modal border.
func (m *modal) contentSize() (int, int) {
	// Modal takes 80% of terminal, minus border (2) and padding (2)
	w := m.width*4/5 - 4
	h := m.height*4/5 - 3 // -3 for border top/bottom + title
	if w < 20 {
		w = 20
	}
	if h < 5 {
		h = 5
	}
	return w, h
}

// readOutput returns a Cmd that reads from the PTY and sends modalOutputMsg.
func (m *modal) readOutput() tea.Cmd {
	return func() tea.Msg {
		buf := make([]byte, 4096)
		n, err := m.ptmx.Read(buf)
		if n > 0 {
			return modalOutputMsg{data: string(buf[:n])}
		}
		if err != nil {
			// Wait for process to finish
			waitErr := m.cmd.Wait()
			return modalDoneMsg{err: waitErr}
		}
		return modalDoneMsg{}
	}
}

// writeInput sends raw bytes to the subprocess PTY.
func (m *modal) writeInput(data string) {
	if m.ptmx != nil {
		_, _ = io.WriteString(m.ptmx, data)
	}
}

func (m *modal) close() {
	// Best-effort shutdown: try SIGINT first, then force-kill.
	if m.cmd != nil && m.cmd.Process != nil {
		_ = m.cmd.Process.Signal(os.Interrupt)
		_ = m.cmd.Process.Kill()
	}
	if m.ptmx != nil {
		_ = m.ptmx.Close()
	}
}

// resize updates the modal dimensions and PTY size.
func (m *modal) resize(width, height int) {
	m.width = width
	m.height = height
	if m.ptmx != nil {
		cw, ch := m.contentSize()
		_ = pty.Setsize(m.ptmx, &pty.Winsize{Rows: uint16(ch), Cols: uint16(cw)})
	}
}

// appendOutput processes raw PTY output into lines.
func (m *modal) appendOutput(data string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Normalize newlines.
	data = strings.ReplaceAll(data, "\r\n", "\n")

	// Ensure we always have a "current" line to append to.
	if len(m.lines) == 0 {
		m.lines = append(m.lines, "")
	}

	for _, chunk := range strings.SplitAfter(data, "\n") {
		hasNL := strings.HasSuffix(chunk, "\n")
		text := strings.TrimSuffix(chunk, "\n")

		// Carriage return moves to start-of-line; approximate by replacing current line.
		if cr := strings.LastIndex(text, "\r"); cr >= 0 {
			text = text[cr+1:]
			m.lines[len(m.lines)-1] = text
		} else {
			m.lines[len(m.lines)-1] += text
		}

		if hasNL {
			m.lines = append(m.lines, "")
		}
	}

	const maxModalLines = 5000
	if len(m.lines) > maxModalLines {
		m.lines = m.lines[len(m.lines)-maxModalLines:]
	}
}

// view renders the modal overlay.
func (m *modal) view(termWidth, termHeight int) string {
	cw, ch := m.contentSize()

	// Get visible lines (bottom of output)
	m.mu.Lock()
	lines := m.lines
	m.mu.Unlock()

	start := len(lines) - ch - m.scroll
	if start < 0 {
		start = 0
	}
	end := start + ch
	if end > len(lines) {
		end = len(lines)
	}
	visible := lines[start:end]

	// Truncate lines to content width
	var content strings.Builder
	for i, line := range visible {
		// Truncate to width (rune-safe; still not ANSI-width aware).
		r := []rune(line)
		if len(r) > cw {
			line = string(r[:cw])
		}
		content.WriteString(line)
		if i < len(visible)-1 {
			content.WriteByte('\n')
		}
	}

	// Pad to fill content height
	for i := len(visible); i < ch; i++ {
		content.WriteByte('\n')
	}

	title := modalTitleStyle.Render(" Planning Session ")
	body := content.String()

	// Build modal box
	modalContent := title + "\n" + body
	box := modalBorderStyle.Width(cw).Render(modalContent)

	// Center the modal on screen
	return lipgloss.Place(termWidth, termHeight, lipgloss.Center, lipgloss.Center, box)
}
