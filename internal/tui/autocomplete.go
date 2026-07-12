package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// isTemplateCommand checks if a command is a template (contains placeholders like <...>)
func isTemplateCommand(command string) bool {
	return strings.Contains(command, "<") && strings.Contains(command, ">")
}

// extractTemplatePrefix extracts the prefix before the placeholder for cursor positioning
func extractTemplatePrefix(command string) string {
	idx := strings.Index(command, "<")
	if idx == -1 {
		return command
	}
	return strings.TrimSpace(command[:idx])
}

// AutocompleteInput wraps textinput.Model with built-in autocomplete functionality
type AutocompleteInput struct {
	textinput textinput.Model
	registry  *CommandRegistry
}

// NewAutocompleteInput creates a new autocomplete input component using built-in suggestions
func NewAutocompleteInput(registry *CommandRegistry, styles *Styles) *AutocompleteInput {
	ti := textinput.New()
	ti.Prompt = "kiro-krew> "

	// Configure solid cursor (non-blinking)
	currentStyles := ti.Styles()
	currentStyles.Cursor.Blink = false
	ti.SetStyles(currentStyles)

	// Enable built-in suggestions
	ti.ShowSuggestions = true

	ti.Focus()

	input := &AutocompleteInput{
		textinput: ti,
		registry:  registry,
	}

	// Initialize suggestions
	input.updateSuggestions()

	return input
}

// Focused returns whether the input currently has focus
func (a *AutocompleteInput) Focused() bool {
	return a.textinput.Focused()
}

// SetFocus sets the focus state without triggering focus commands
func (a *AutocompleteInput) SetFocus(focused bool) {
	if focused {
		a.textinput.Focus()
	} else {
		a.textinput.Blur()
	}
}

// Focus gives focus to the underlying textinput
func (a *AutocompleteInput) Focus() tea.Cmd {
	return a.textinput.Focus()
}

// Blur removes focus from the underlying textinput
func (a *AutocompleteInput) Blur() {
	a.textinput.Blur()
}

// Value returns the current input value
func (a *AutocompleteInput) Value() string {
	return a.textinput.Value()
}

// SetValue sets the input value and updates autocomplete state
func (a *AutocompleteInput) SetValue(value string) {
	a.textinput.SetValue(value)
	a.updateSuggestions()
}

// Update handles key events for the autocomplete input
func (a *AutocompleteInput) Update(msg tea.Msg) (*AutocompleteInput, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return a.handleKeyMsg(msg)
	}

	// Update underlying textinput for other messages
	var cmd tea.Cmd
	a.textinput, cmd = a.textinput.Update(msg)
	return a, cmd
}

// handleKeyMsg processes key events with template command support
func (a *AutocompleteInput) handleKeyMsg(msg tea.KeyMsg) (*AutocompleteInput, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Handle template commands before normal enter processing
		current := a.textinput.CurrentSuggestion()
		if current != "" && isTemplateCommand(current) {
			// Template commands position cursor without executing
			a.textinput.SetValue(extractTemplatePrefix(current))
			a.textinput.CursorEnd()
			a.updateSuggestions()
			return a, nil
		}
		// Fall through to normal textinput handling for execution
	case "esc":
		// Clear suggestions on escape
		a.textinput.SetSuggestions([]string{})
		return a, nil
	}

	// Handle all other input normally and update suggestions
	var cmd tea.Cmd
	a.textinput, cmd = a.textinput.Update(msg)

	// Update suggestions after any text change
	a.updateSuggestions()

	return a, cmd
}

// updateSuggestions updates the textinput's built-in suggestions based on current input
func (a *AutocompleteInput) updateSuggestions() {
	input := a.textinput.Value()

	if input == "" {
		// Show all commands when input is empty
		suggestions := a.registry.GetFlattenedMatches("")
		a.textinput.SetSuggestions(suggestions)
		return
	}

	// Use flattened commands for atomic compound command matching
	suggestions := a.registry.GetFlattenedMatches(input)
	a.textinput.SetSuggestions(suggestions)
}

// View renders the autocomplete input using built-in textinput rendering
func (a *AutocompleteInput) View() string {
	return a.textinput.View()
}

// IsValidCommand checks if current input is a valid command
func (a *AutocompleteInput) IsValidCommand() bool {
	return a.registry.IsValidCommand(a.textinput.Value())
}
