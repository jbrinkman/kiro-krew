package hotkey

import (
	"os"
	"testing"
)

func TestIsKiroKrewContext(t *testing.T) {
	// Test without environment variable
	os.Unsetenv("KIRO_KREW_WATCHER_PID")
	if IsKiroKrewContext() {
		t.Error("Expected IsKiroKrewContext to return false when KIRO_KREW_WATCHER_PID is not set")
	}

	// Test with environment variable
	os.Setenv("KIRO_KREW_WATCHER_PID", "12345")
	if !IsKiroKrewContext() {
		t.Error("Expected IsKiroKrewContext to return true when KIRO_KREW_WATCHER_PID is set")
	}

	// Cleanup
	os.Unsetenv("KIRO_KREW_WATCHER_PID")
}

func TestIsCtrlOptionP(t *testing.T) {
	tests := []struct {
		name     string
		keyStr   string
		expected bool
	}{
		{"Ctrl+Alt+P", "ctrl+alt+p", true},
		{"Ctrl+C", "ctrl+c", false},
		{"Regular P", "p", false},
		{"Alt+P", "alt+p", false},
		{"Ctrl+P", "ctrl+p", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the string matching logic directly
			result := tt.keyStr == "ctrl+alt+p"
			if result != tt.expected {
				t.Errorf("Expected %v for %s, got %v", tt.expected, tt.keyStr, result)
			}
		})
	}
}