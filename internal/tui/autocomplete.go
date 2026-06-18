package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// AutocompleteState manages the current autocomplete UI state
type AutocompleteState struct {
	suggestions   []string
	selectedIndex int
	showDropdown  bool
	ghostText     string
}

// AutocompleteInput wraps textinput.Model with autocomplete functionality
type AutocompleteInput struct {
	textinput textinput.Model
	registry  *CommandRegistry
	state     AutocompleteState
	styles    *Styles
}

// NewAutocompleteInput creates a new autocomplete input component
func NewAutocompleteInput(registry *CommandRegistry, styles *Styles) *AutocompleteInput {
	ti := textinput.New()
	ti.Prompt = "kiro-krew> "
	ti.Focus()

	return &AutocompleteInput{
		textinput: ti,
		registry:  registry,
		styles:    styles,
		state: AutocompleteState{
			suggestions:   []string{},
			selectedIndex: 0,
			showDropdown:  false,
			ghostText:     "",
		},
	}
}

// Focus gives focus to the underlying textinput
func (a *AutocompleteInput) Focus() tea.Cmd {
	return a.textinput.Focus()
}

// Blur removes focus from the underlying textinput
func (a *AutocompleteInput) Blur() {
	a.textinput.Blur()
	a.state.showDropdown = false
}

// Value returns the current input value
func (a *AutocompleteInput) Value() string {
	return a.textinput.Value()
}

// SetValue sets the input value and updates autocomplete state
func (a *AutocompleteInput) SetValue(value string) {
	a.textinput.SetValue(value)
	a.updateAutocomplete()
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

// handleKeyMsg processes key events for autocomplete functionality
func (a *AutocompleteInput) handleKeyMsg(msg tea.KeyMsg) (*AutocompleteInput, tea.Cmd) {
	switch msg.String() {
	case "up":
		if a.state.showDropdown && len(a.state.suggestions) > 0 {
			a.state.selectedIndex--
			if a.state.selectedIndex < 0 {
				a.state.selectedIndex = len(a.state.suggestions) - 1
			}
			a.updateGhostText()
			return a, nil
		}

	case "down":
		if a.state.showDropdown && len(a.state.suggestions) > 0 {
			a.state.selectedIndex++
			if a.state.selectedIndex >= len(a.state.suggestions) {
				a.state.selectedIndex = 0
			}
			a.updateGhostText()
			return a, nil
		}

	case "tab":
		if a.state.ghostText != "" {
			a.textinput.SetValue(a.state.ghostText)
			a.textinput.CursorEnd()
			a.updateAutocomplete()
			return a, nil
		}
		if a.state.showDropdown && len(a.state.suggestions) > 0 {
			selected := a.state.suggestions[a.state.selectedIndex]
			a.textinput.SetValue(selected)
			a.textinput.CursorEnd()
			a.updateAutocomplete()
			return a, nil
		}

	case "esc":
		a.state.showDropdown = false
		a.state.ghostText = ""
		return a, nil

	case "enter":
		// Don't handle enter here - let parent handle command execution
		// But hide dropdown
		a.state.showDropdown = false
		a.state.ghostText = ""

		// Pass through to underlying textinput
		var cmd tea.Cmd
		a.textinput, cmd = a.textinput.Update(msg)
		return a, cmd

	default:
		// Handle regular input
		var cmd tea.Cmd
		a.textinput, cmd = a.textinput.Update(msg)
		a.updateAutocomplete()
		return a, cmd
	}

	return a, nil
}

// updateAutocomplete updates the autocomplete state based on current input
func (a *AutocompleteInput) updateAutocomplete() {
	input := a.textinput.Value()

	if input == "" {
		a.state.showDropdown = false
		a.state.suggestions = []string{}
		a.state.ghostText = ""
		a.state.selectedIndex = 0
		return
	}

	// Use flattened commands for atomic compound command matching
	suggestions := a.registry.GetFlattenedMatches(input)
	
	a.state.suggestions = suggestions
	a.state.showDropdown = len(suggestions) > 0
	a.state.selectedIndex = 0

	a.updateGhostText()
}

// updateGhostText updates the ghost text based on current selection
func (a *AutocompleteInput) updateGhostText() {
	input := a.textinput.Value()

	if !a.state.showDropdown || len(a.state.suggestions) == 0 {
		a.state.ghostText = ""
		return
	}

	if a.state.selectedIndex < len(a.state.suggestions) {
		suggestion := a.state.suggestions[a.state.selectedIndex]
		if strings.HasPrefix(strings.ToLower(suggestion), strings.ToLower(input)) && len(suggestion) > len(input) {
			a.state.ghostText = suggestion
		} else {
			a.state.ghostText = ""
		}
	}
}

// View renders the autocomplete input with ghost text
func (a *AutocompleteInput) View() string {
	// Get the base input view
	baseView := a.textinput.View()

	// Add ghost text if available
	if a.state.ghostText != "" && len(a.state.ghostText) > len(a.textinput.Value()) {
		ghost := a.state.ghostText[len(a.textinput.Value()):]
		ghostText := a.styles.AutocompleteGhost.Render(ghost)
		
		// Insert ghost text right after the cursor, without extra spaces
		// The textinput view includes prompt + input + cursor, we need to place ghost text after input but before cursor
		prompt := a.textinput.Prompt
		inputValue := a.textinput.Value()
		
		// Render as: prompt + inputValue + ghostText + cursor
		return prompt + inputValue + ghostText
	}

	return baseView
}

// ViewDropdown renders the dropdown menu
func (a *AutocompleteInput) ViewDropdown() string {
	if !a.state.showDropdown || len(a.state.suggestions) == 0 {
		return ""
	}

	var lines []string

	for i, suggestion := range a.state.suggestions {
		if i >= 10 { // Limit dropdown size
			break
		}

		var style lipgloss.Style
		if i == a.state.selectedIndex {
			style = a.styles.AutocompleteSelected.Padding(0, 1)
		} else {
			style = lipgloss.NewStyle().Padding(0, 1)
		}

		lines = append(lines, style.Render(suggestion))
	}

	// Create dropdown box using theme styles
	return a.styles.AutocompleteDropdown.Render(strings.Join(lines, "\n"))
}

// IsDropdownVisible returns whether the dropdown is currently visible
func (a *AutocompleteInput) IsDropdownVisible() bool {
	return a.state.showDropdown
}

// GetSelectedSuggestion returns the currently selected suggestion
func (a *AutocompleteInput) GetSelectedSuggestion() string {
	if !a.state.showDropdown || len(a.state.suggestions) == 0 || a.state.selectedIndex >= len(a.state.suggestions) {
		return ""
	}
	return a.state.suggestions[a.state.selectedIndex]
}

// IsValidCommand checks if current input is a valid command
func (a *AutocompleteInput) IsValidCommand() bool {
	return a.registry.IsValidCommand(a.textinput.Value())
}
