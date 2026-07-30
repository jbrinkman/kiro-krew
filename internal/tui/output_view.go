package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/jbrinkman/kiro-krew/internal/agent"
)

// OutputView displays agent output in a scrollable view.
//
// Footer System Integration (Issue #211):
// OutputView is used by agent tabs and must coordinate with the unified footer
// rendering system. When agent tabs are rendered, the flow is:
//  1. AgentTab.View() calls OutputView.View() to get viewport content
//  2. The parent model (tui.go) wraps this content with renderTabContentWithFooter()
//  3. The footer system (FooterManager) appends a 3-line footer:
//     - Line 1: Separator (─────)
//     - Line 2: Input row (kiro-krew> prompt)
//     - Line 3: Status row (theme: <name>)
//
// To prevent the footer from being pushed off-screen, OutputView must reserve
// space by subtracting the footer height (3 lines) from the viewport height.
// This ensures the total rendered content (viewport + footer) fits within the
// allocated screen space without overflow or layout issues.
type OutputView struct {
	viewport     viewport.Model
	manager      *agent.Manager
	styles       *Styles
	width        int
	height       int
	cachedOutput []string
	agentID      string // Filter output by this agent ID, empty string shows all
	lastGen      uint64 // last observed OutputCapture generation
}

// SetStyles updates the styles used by this view.
func (ov *OutputView) SetStyles(styles *Styles) {
	ov.styles = styles
}

// NewOutputView creates a new output view for all agents
func NewOutputView(manager *agent.Manager, styles *Styles) *OutputView {
	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(24))
	return &OutputView{
		viewport: vp,
		manager:  manager,
		styles:   styles,
		agentID:  "", // Empty means show all agents
	}
}

// NewOutputViewForAgent creates a new output view filtered to a specific agent
func NewOutputViewForAgent(agentID string, manager *agent.Manager, styles *Styles) *OutputView {
	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(24))
	return &OutputView{
		viewport: vp,
		manager:  manager,
		styles:   styles,
		agentID:  agentID,
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

		// Reserve space for the footer system (separator + input row + status row = 3 lines)
		footerHeight := 3
		viewportHeight := msg.Height - footerHeight
		if viewportHeight < 1 {
			viewportHeight = 1 // Minimum viewport height
		}

		ov.viewport = viewport.New(viewport.WithWidth(msg.Width), viewport.WithHeight(viewportHeight))
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

	gen := ov.manager.GetOutputGeneration()
	if gen != ov.lastGen {
		ov.lastGen = gen
		ov.refreshContent()
	}
	return ov.viewport.View()
}

// Resize updates the output view dimensions
// Note: height should be the total available height including footer space.
// The viewport height will be adjusted to leave room for the footer (3 lines).
func (ov *OutputView) Resize(width, height int) {
	ov.width = width
	ov.height = height

	// Reserve space for the footer system (separator + input row + status row = 3 lines)
	// This ensures the footer rendered by renderTabContentWithFooter() doesn't overflow
	footerHeight := 3
	viewportHeight := height - footerHeight
	if viewportHeight < 1 {
		viewportHeight = 1 // Minimum viewport height
	}

	ov.viewport = viewport.New(viewport.WithWidth(width), viewport.WithHeight(viewportHeight))
	ov.lastGen = 0 // Force refresh on next View()
	ov.refreshContent()
}

// refreshContent updates the viewport content with latest agent output
func (ov *OutputView) refreshContent() {
	agents := ov.manager.List()
	capturedLines := ov.manager.GetOutputLines()

	// If filtering by agent ID, only show that agent
	if ov.agentID != "" {
		var filteredAgents []*agent.Agent
		for _, agentItem := range agents {
			if agentItem.ID == ov.agentID {
				filteredAgents = []*agent.Agent{agentItem}
				break
			}
		}
		agents = filteredAgents
	}

	if len(agents) == 0 {
		if len(capturedLines) == 0 {
			content := ov.styles.Warning.Render("No agents running. Use 'watch start' to begin monitoring issues.")
			ov.viewport.SetContent(content)
			return
		}

		// If filtering by agent ID but agent not found, show appropriate message
		if ov.agentID != "" {
			content := ov.styles.Warning.Render(fmt.Sprintf("Agent %s not found or no longer running.", ov.agentID))
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

		// Add separator between agents (only when showing multiple agents)
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
