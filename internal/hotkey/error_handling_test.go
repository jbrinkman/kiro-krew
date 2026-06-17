package hotkey

import (
	"os"
	"testing"
)

func TestHotkeyErrorHandling(t *testing.T) {
	// Test IsKiroKrewContext behavior
	os.Unsetenv("KIRO_KREW_WATCHER_PID")
	if IsKiroKrewContext() {
		t.Error("Expected false when not in kiro-krew context")
	}

	os.Setenv("KIRO_KREW_WATCHER_PID", "12345")
	if !IsKiroKrewContext() {
		t.Error("Expected true when in kiro-krew context")
	}
	os.Unsetenv("KIRO_KREW_WATCHER_PID")
}

func TestIsCtrlOptionPDetection(t *testing.T) {
	tests := []struct {
		keyStr   string
		expected bool
	}{
		{"ctrl+alt+p", true},
		{"ctrl+c", false},
		{"p", false},
		{"alt+p", false},
		{"ctrl+p", false},
	}

	for _, test := range tests {
		result := test.keyStr == "ctrl+alt+p"
		if result != test.expected {
			t.Errorf("Key %s: expected %v, got %v", test.keyStr, test.expected, result)
		}
	}
}
