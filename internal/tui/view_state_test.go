package tui

import (
	"testing"
)

func TestViewManager(t *testing.T) {
	vm := NewViewManager()

	// Test initial state
	if vm.CurrentView() != ViewConsole {
		t.Errorf("Expected initial view to be ViewConsole, got %v", vm.CurrentView())
	}

	// Test toggle
	vm.ToggleView()
	if vm.CurrentView() != ViewAgentOutput {
		t.Errorf("Expected view to be ViewAgentOutput after toggle, got %v", vm.CurrentView())
	}

	// Test toggle back
	vm.ToggleView()
	if vm.CurrentView() != ViewConsole {
		t.Errorf("Expected view to be ViewConsole after second toggle, got %v", vm.CurrentView())
	}

	// Test explicit set
	vm.SetView(ViewAgentOutput)
	if vm.CurrentView() != ViewAgentOutput {
		t.Errorf("Expected view to be ViewAgentOutput after SetView, got %v", vm.CurrentView())
	}
}

func TestViewStatePreservation(t *testing.T) {
	vm := NewViewManager()

	// Test console scroll preservation
	vm.PreserveConsoleScroll(42)
	if vm.GetConsoleScroll() != 42 {
		t.Errorf("Expected console scroll position 42, got %d", vm.GetConsoleScroll())
	}
}
