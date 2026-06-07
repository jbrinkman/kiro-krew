package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/jbrinkman/kiro-krew/internal/agent"
)

// OutputView displays agent output in a scrollable view
type OutputView struct {
	viewport     viewport.Model
	manager      *agent.Manager
	styles       *Styles
	width        int
	height       int
	lastUpdate   time.Time
	cachedOutput []string
}

// SetStyles updates the styles used by this view.
func (ov *OutputView) SetStyles(styles *Styles) {
	ov.styles = styles
}

// NewOutputView creates a new output view
func NewOutputView(manager *agent.Manager, styles *Styles) *OutputView {
	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(24))
	return &OutputView{
		viewport:   vp,
		manager:    manager,
		styles:     styles,
		lastUpdate: time.Now(),
	}
}

// Init initializes the output view
func (ov *OutputView) Init() tea.Cmd {
	return nil
}

// Update handles messages for the output view
func (ov *OutputView) Update(msg tea.Msg) (*OutputView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		ov.width = msg.Width
		ov.height = msg.Height
		ov.viewport = viewport.New(viewport.WithWidth(msg.Width), viewport.WithHeight(msg.Height))
		ov.refreshContent()
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up":
			ov.viewport.ScrollUp(1)
		case "down":
			ov.viewport.ScrollDown(1)
		case "pgup":
			ov.viewport.HalfPageUp()
		case "pgdown":
			ov.viewport.HalfPageDown()
		case "home":
			ov.viewport.GotoTop()
		case "end":
			ov.viewport.GotoBottom()
		}
	}

	var cmd tea.Cmd
	ov.viewport, cmd = ov.viewport.Update(msg)
	return ov, cmd
}

// View renders the output view
func (ov *OutputView) View() string {
	if ov.width == 0 || ov.height == 0 {
		return ""
	}

	ov.refreshContent()
	return ov.viewport.View()
}

// Resize updates the output view dimensions
func (ov *OutputView) Resize(width, height int) {
	ov.width = width
	ov.height = height
	ov.viewport = viewport.New(viewport.WithWidth(width), viewport.WithHeight(height))
	ov.refreshContent()
}

// refreshContent updates the viewport content with latest agent output
func (ov *OutputView) refreshContent() {
	agents := ov.manager.List()
	capturedLines := ov.manager.GetOutputLines()

	if len(agents) == 0 {
		if len(capturedLines) == 0 {
			content := ov.styles.Warning.Render("No agents running. Use 'watch start' to begin monitoring issues.")
			ov.viewport.SetContent(content)
			return
		}

		content := strings.Join(capturedLines, "\n")
		ov.viewport.SetContent(content)
		return
	}

	var output []string

	for _, agentItem := range agents {
		// Agent header with status indicator
		statusIndicator := "●"
		statusStyle := ov.styles.Success
		switch agentItem.Status {
		case agent.StatusRunning:
			statusStyle = ov.styles.Success
		case agent.StatusCompleted:
			statusStyle = ov.styles.Activity
		case agent.StatusFailed:
			statusStyle = ov.styles.Error
		}

		header := fmt.Sprintf("%s Agent %s - Issue #%d: %s",
			statusStyle.Render(statusIndicator),
			agentItem.ID,
			agentItem.IssueNumber,
			agentItem.IssueTitle)

		output = append(output, header)

		agentPrefix := fmt.Sprintf("[agent issue-%d] ", agentItem.IssueNumber)
		agentOutput := make([]string, 0)
		for _, line := range capturedLines {
			if strings.HasPrefix(line, agentPrefix) {
				agentOutput = append(agentOutput, strings.TrimPrefix(line, agentPrefix))
			}
		}
		if len(agentOutput) == 0 {
			agentOutput = []string{
				"No captured output yet.",
			}
		}

		// Wrap and indent agent output
		for _, line := range agentOutput {
			wrapped := ov.wrapText(line, ov.width-4)
			for _, wrappedLine := range wrapped {
				output = append(output, "  "+wrappedLine)
			}
		}

		// Add separator between agents
		if len(agents) > 1 {
			output = append(output, "")
			output = append(output, ov.styles.Separator.Render(strings.Repeat("─", ov.width)))
			output = append(output, "")
		}
	}

	content := strings.Join(output, "\n")
	ov.viewport.SetContent(content)

	// Intentionally do not force-scroll here; preserve the user's scroll position.
}

// wrapText wraps text to fit within the specified width
func (ov *OutputView) wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	if len(text) <= width {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{text}
	}

	currentLine := words[0]

	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}
