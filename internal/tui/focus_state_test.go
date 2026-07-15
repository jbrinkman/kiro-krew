package tui

import "testing"

func TestFocusTargetString(t *testing.T) {
	tests := []struct {
		target   FocusTarget
		expected string
	}{
		{FocusTargetFooter, "footer"},
		{FocusTargetMessage, "message"},
		{FocusTargetNone, "none"},
	}

	for _, tt := range tests {
		if got := tt.target.String(); got != tt.expected {
			t.Errorf("FocusTarget.String() = %v, want %v", got, tt.expected)
		}
	}
}

func TestFocusTargetIsValid(t *testing.T) {
	tests := []struct {
		target   FocusTarget
		expected bool
	}{
		{FocusTargetFooter, true},
		{FocusTargetMessage, true},
		{FocusTargetNone, true},
		{FocusTarget("invalid"), false},
		{FocusTarget(""), false},
	}

	for _, tt := range tests {
		if got := tt.target.IsValid(); got != tt.expected {
			t.Errorf("FocusTarget(%q).IsValid() = %v, want %v",
				tt.target, got, tt.expected)
		}
	}
}
