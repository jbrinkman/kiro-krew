package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/jbrinkman/kiro-krew/internal/agent"
)

// extractIssueNumberFromAgentID parses issue number from agent ID format "agent-{issueNumber}-{timestamp}"
func extractIssueNumberFromAgentID(agentID string) (int, error) {
	parts := strings.Split(agentID, "-")
	if len(parts) >= 3 && parts[0] == "agent" {
		return strconv.Atoi(parts[1])
	}
	return 0, fmt.Errorf("invalid agent ID format")
}

// AgentTab implements Tab interface for individual agent views
type AgentTab struct {
	agentID    string
	outputView *OutputView
}

// NewAgentTab creates a new agent tab
func NewAgentTab(agentID string, manager *agent.Manager, styles *Styles) *AgentTab {
	return &AgentTab{
		agentID:    agentID,
		outputView: NewOutputViewForAgent(agentID, manager, styles),
	}
}

// ID returns the tab identifier
func (at *AgentTab) ID() string {
	return "agent-" + at.agentID
}

// Type returns the tab type
func (at *AgentTab) Type() TabType {
	return TabTypeAgent
}

// Title returns the tab title
func (at *AgentTab) Title() string {
	// Primary: Use direct agent lookup
	if agent := at.outputView.manager.GetAgent(at.agentID); agent != nil {
		return fmt.Sprintf("Issue %d", agent.IssueNumber)
	}
	
	// Fallback: Parse issue number from agent ID format "agent-{issueNumber}-{timestamp}"
	if issueNum, err := extractIssueNumberFromAgentID(at.agentID); err == nil {
		return fmt.Sprintf("Issue %d", issueNum)
	}
	
	// Last resort: Use old format
	return "Agent " + at.agentID
}

// IsClosable returns whether this tab can be closed
func (at *AgentTab) IsClosable() bool {
	return true
}

// View returns the tab's rendered content
func (at *AgentTab) View() string {
	return at.outputView.View()
}

// Update handles messages for the agent tab
func (at *AgentTab) Update(msg tea.Msg) (Tab, tea.Cmd) {
	var cmd tea.Cmd
	at.outputView, cmd = at.outputView.Update(msg)
	return at, cmd
}

// Resize updates the tab dimensions
func (at *AgentTab) Resize(width, height int) {
	at.outputView.Resize(width, height)
}
