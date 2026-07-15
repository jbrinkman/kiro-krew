package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// mockTab is a minimal implementation to verify the Tab interface compiles
type mockTab struct{}

func (m *mockTab) ID() string                                   { return "mock" }
func (m *mockTab) Type() TabType                                { return TabTypeMain }
func (m *mockTab) Title() string                                { return "Mock" }
func (m *mockTab) IsClosable() bool                             { return false }
func (m *mockTab) View() string                                 { return "" }
func (m *mockTab) Update(tea.Msg) (Tab, tea.Cmd)                { return m, nil }
func (m *mockTab) Resize(width, height int)                     {}
func (m *mockTab) CaptureFocusState() FocusTarget               { return FocusTargetFooter }
func (m *mockTab) RestoreFocusState(target FocusTarget) tea.Cmd { return nil }

// TestTabInterfaceDefinition verifies that the Tab interface compiles correctly
func TestTabInterfaceDefinition(t *testing.T) {
	var tab Tab = &mockTab{}

	// Verify basic methods
	if tab.ID() != "mock" {
		t.Errorf("ID() = %v, want mock", tab.ID())
	}

	// Verify focus state methods exist and work
	focus := tab.CaptureFocusState()
	if focus != FocusTargetFooter {
		t.Errorf("CaptureFocusState() = %v, want %v", focus, FocusTargetFooter)
	}

	cmd := tab.RestoreFocusState(FocusTargetFooter)
	if cmd != nil {
		t.Errorf("RestoreFocusState() = %v, want nil", cmd)
	}
}
