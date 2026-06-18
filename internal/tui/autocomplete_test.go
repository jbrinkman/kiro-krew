package tui

import (
	"testing"

	"github.com/jbrinkman/kiro-krew/internal/config"
)

func TestAutocompleteInput(t *testing.T) {
	registry := NewCommandRegistry()
	theme := &config.Theme{} // Use default theme
	styles := NewStyles(theme)

	autocomplete := NewAutocompleteInput(registry, styles)

	// Test initial state
	if autocomplete.Value() != "" {
		t.Error("Expected empty initial value")
	}

	// Test setting value and autocomplete update
	autocomplete.SetValue("w")
	if !autocomplete.IsDropdownVisible() {
		t.Error("Expected dropdown to be visible after typing 'w'")
	}

	// Test valid command detection
	autocomplete.SetValue("watch start")
	if !autocomplete.IsValidCommand() {
		t.Error("Expected 'watch start' to be valid command")
	}

	autocomplete.SetValue("invalid command")
	if autocomplete.IsValidCommand() {
		t.Error("Expected 'invalid command' to be invalid")
	}
}
