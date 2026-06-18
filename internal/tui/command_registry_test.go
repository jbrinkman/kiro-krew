package tui

import "testing"

func TestCommandRegistry(t *testing.T) {
	registry := NewCommandRegistry()

	// Test command filtering
	matches := registry.FilterCommands("w")
	if len(matches) != 1 || matches[0].Name != "watch" {
		t.Errorf("Expected 1 match for 'w', got %d", len(matches))
	}

	// Test subcommand filtering
	subcommands := registry.GetSubcommands("watch")
	if len(subcommands) != 2 {
		t.Errorf("Expected 2 subcommands for 'watch', got %d", len(subcommands))
	}

	// Test best match
	match := registry.GetBestMatch("wat")
	if match != "watch" {
		t.Errorf("Expected 'watch' for 'wat', got '%s'", match)
	}

	// Test valid command
	if !registry.IsValidCommand("watch start") {
		t.Error("Expected 'watch start' to be valid")
	}

	if registry.IsValidCommand("invalid command") {
		t.Error("Expected 'invalid command' to be invalid")
	}
}
