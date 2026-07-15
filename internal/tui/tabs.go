package tui

import tea "charm.land/bubbletea/v2"

// TabType represents the different types of tabs
type TabType int

const (
	TabTypeMain TabType = iota
	TabTypeAgent
	TabTypePlanning
	TabTypeLog
)

// String returns the string representation of the TabType
func (t TabType) String() string {
	switch t {
	case TabTypeMain:
		return "Main"
	case TabTypeAgent:
		return "Agent"
	case TabTypePlanning:
		return "Planning"
	case TabTypeLog:
		return "Log"
	default:
		return "Unknown"
	}
}

// Tab interface defines the contract for all tab implementations
type Tab interface {
	ID() string
	Type() TabType
	Title() string
	IsClosable() bool
	View() string
	Update(tea.Msg) (Tab, tea.Cmd)
	Resize(width, height int)

	// Focus state management
	CaptureFocusState() FocusTarget
	RestoreFocusState(target FocusTarget) tea.Cmd
}
