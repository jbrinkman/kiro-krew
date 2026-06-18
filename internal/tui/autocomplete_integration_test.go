package tui

import (
	"strings"
	"testing"

	"github.com/jbrinkman/kiro-krew/internal/config"
)

// TestAutocompleteIntegration performs comprehensive integration testing
// for all autocomplete UX fixes implemented in tasks 1-4
func TestAutocompleteIntegration(t *testing.T) {
	theme := &config.Theme{} // Use default theme
	styles := NewStyles(theme)
	registry := NewCommandRegistry()
	input := NewAutocompleteInput(registry, styles)

	t.Run("Task 1: Ghost Text Spacing Fix", func(t *testing.T) {
		// Reset state
		input.SetValue("")
		
		// Type 'w' and check ghost text
		input.SetValue("w")
		view := input.View()
		
		// Ghost text should appear without extra spaces
		// Should show "watch" completion for "w" without gaps
		if !strings.Contains(view, "watch") {
			t.Errorf("Expected ghost text to show 'watch' completion for 'w', got: %s", view)
		}
		
		// Verify no double spaces or formatting issues
		if strings.Contains(view, "w atch") || strings.Contains(view, "w  atch") {
			t.Errorf("Ghost text contains spacing issues: %s", view)
		}
	})

	t.Run("Task 2: Cursor Positioning State Management", func(t *testing.T) {
		// Reset state
		input.SetValue("")
		
		// Type 'w' to trigger autocomplete
		input.SetValue("w")
		
		// Verify autocomplete state is correct
		if !input.IsDropdownVisible() {
			t.Error("Expected dropdown to be visible after typing 'w'")
		}
		
		// Verify selected suggestion is available
		suggestion := input.GetSelectedSuggestion()
		if suggestion == "" {
			t.Error("Expected a selected suggestion when dropdown is visible")
		}
		
		// Test SetValue with completion updates state correctly
		input.SetValue("watch start")
		value := input.Value()
		if value != "watch start" {
			t.Errorf("Expected SetValue to set exact value 'watch start', got: %s", value)
		}
	})

	t.Run("Task 3: Dropdown Display Integration", func(t *testing.T) {
		// Reset state  
		input.SetValue("")
		
		// Type 'w' to trigger dropdown
		input.SetValue("w")
		
		// Verify dropdown is visible
		if !input.IsDropdownVisible() {
			t.Error("Expected dropdown to be visible after typing 'w'")
		}
		
		// Verify dropdown content
		dropdown := input.ViewDropdown()
		if dropdown == "" {
			t.Error("Expected dropdown content, got empty string")
		}
		
		// Should contain watch-related commands
		if !strings.Contains(dropdown, "watch") {
			t.Errorf("Expected dropdown to contain 'watch' commands, got: %s", dropdown)
		}
		
		// Verify dropdown can be hidden
		input.SetValue("")
		if input.IsDropdownVisible() {
			t.Error("Expected dropdown to be hidden for empty input")
		}
	})

	t.Run("Task 4: Compound Command Units", func(t *testing.T) {
		// Reset state
		input.SetValue("")
		
		// Type 'w' and verify flattened commands appear
		input.SetValue("w")
		
		// Check that compound commands are returned as units
		matches := registry.GetFlattenedMatches("w")
		
		foundWatchStart := false
		foundWatchStop := false
		for _, match := range matches {
			if match == "watch start" {
				foundWatchStart = true
			}
			if match == "watch stop" {
				foundWatchStop = true
			}
		}
		
		if !foundWatchStart {
			t.Error("Expected 'watch start' as a flattened command unit")
		}
		if !foundWatchStop {
			t.Error("Expected 'watch stop' as a flattened command unit")  
		}
		
		// Verify typing 'watch s' suggests compound commands
		input.SetValue("watch s")
		watchSMatches := registry.GetFlattenedMatches("watch s")
		
		if len(watchSMatches) == 0 {
			t.Error("Expected matches for 'watch s'")
		}
		
		foundStartMatch := false
		foundStopMatch := false
		for _, match := range watchSMatches {
			if match == "watch start" {
				foundStartMatch = true
			}
			if match == "watch stop" {
				foundStopMatch = true
			}
		}
		
		if !foundStartMatch || !foundStopMatch {
			t.Errorf("Expected both 'watch start' and 'watch stop' matches for 'watch s', got: %v", watchSMatches)
		}
	})

	t.Run("Edge Cases and Error Handling", func(t *testing.T) {
		// Test empty input
		input.SetValue("")
		if input.IsDropdownVisible() {
			t.Error("Dropdown should not be visible for empty input")
		}
		
		// Test invalid command
		input.SetValue("invalid")
		if input.IsValidCommand() {
			t.Error("'invalid' should not be a valid command")
		}
		
		// Test that non-matching input doesn't show dropdown
		input.SetValue("xyz")
		if input.IsDropdownVisible() {
			t.Error("Dropdown should not be visible for non-matching input 'xyz'")
		}
	})

	t.Run("Performance and State Consistency", func(t *testing.T) {
		// Test rapid input changes
		testInputs := []string{"", "w", "wa", "wat", "watch", "watch ", "watch s", "watch st", "watch start"}
		
		for _, testInput := range testInputs {
			input.SetValue(testInput)
			
			// Verify state consistency
			if input.IsDropdownVisible() {
				dropdown := input.ViewDropdown()
				if dropdown == "" {
					t.Errorf("Dropdown marked as visible but content is empty for input: '%s'", testInput)
				}
				
				suggestion := input.GetSelectedSuggestion()
				if suggestion == "" {
					t.Errorf("Dropdown visible but no suggestion selected for input: '%s'", testInput)
				}
			}
			
			// Verify view rendering doesn't panic
			view := input.View()
			if view == "" && testInput != "" {
				t.Errorf("View rendering returned empty string for input: '%s'", testInput)
			}
		}
	})

	t.Run("Theme Integration", func(t *testing.T) {
		// Test that different theme structures work
		defaultTheme := &config.Theme{}
		themeStyles := NewStyles(defaultTheme)
		themeInput := NewAutocompleteInput(registry, themeStyles)
		
		themeInput.SetValue("w")
		
		// Verify rendering works with theme
		view := themeInput.View()
		if view == "" {
			t.Error("View rendering failed with theme")
		}
		
		dropdown := themeInput.ViewDropdown()
		if !themeInput.IsDropdownVisible() {
			t.Error("Dropdown should be visible with theme")
		}
		
		if dropdown == "" {
			t.Error("Dropdown content empty with theme")
		}
	})
}

// TestAutocompleteWorkflow tests complete user workflows without key simulation
func TestAutocompleteWorkflow(t *testing.T) {
	theme := &config.Theme{}
	styles := NewStyles(theme)
	registry := NewCommandRegistry()
	input := NewAutocompleteInput(registry, styles)

	t.Run("Complete Workflow: Type w -> Verify State", func(t *testing.T) {
		// Start with empty input
		input.SetValue("")
		
		// Type 'w'
		input.SetValue("w")
		
		// Verify autocomplete activates
		if !input.IsDropdownVisible() {
			t.Fatal("Dropdown should be visible after typing 'w'")
		}
		
		// Verify we have suggestions
		suggestion := input.GetSelectedSuggestion()
		if suggestion == "" {
			t.Error("Should have a selected suggestion")
		}
		
		// Test setting completed value
		input.SetValue("watch start")
		value := input.Value()
		if value != "watch start" {
			t.Errorf("Expected 'watch start', got: %s", value)
		}
		
		// Verify it's a valid command
		if !input.IsValidCommand() {
			t.Error("'watch start' should be a valid command")
		}
	})

	t.Run("Command Validation Workflow", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected bool
			desc     string
		}{
			{"watch start", true, "compound command should be valid"},
			{"watch stop", true, "compound command should be valid"},
			{"status", true, "simple command should be valid"},
			{"help", true, "simple command should be valid"},
			{"invalid", false, "invalid command should not be valid"},
			{"watch invalid", false, "invalid subcommand should not be valid"},
			{"", false, "empty input should not be valid"},
		}
		
		for _, tc := range testCases {
			input.SetValue(tc.input)
			result := input.IsValidCommand()
			if result != tc.expected {
				t.Errorf("%s: input '%s' expected valid=%t, got=%t", tc.desc, tc.input, tc.expected, result)
			}
		}
	})
}