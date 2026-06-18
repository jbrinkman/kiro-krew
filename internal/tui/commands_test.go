package tui

import "testing"

func TestFormatCommitHash(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"full hash", "a76a9a4b8c3d2e1f", "a76a9a4"},
		{"exactly 7 chars", "a76a9a4", "a76a9a4"},
		{"short hash", "abc12", "abc12"},
		{"empty string", "", "unknown"},
		{"unknown value", "unknown", "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCommitHash(tt.input)
			if got != tt.want {
				t.Errorf("formatCommitHash(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
