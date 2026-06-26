package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
			value := selected
			if isTemplateCommand(selected) {
				value = extractTemplatePrefix(selected)
			}
			a.textinput.SetValue(value)
			a.textinput.CursorEnd()
			a.updateAutocomplete()
			return a, nil
		}

	case "esc":
		a.state.showDropdown = false
		a.state.ghostText = ""
		return a, nil

	case "enter":
		// Apply selected suggestion if dropdown is visible
		if a.state.showDropdown && len(a.state.suggestions) > 0 {
			selected := a.state.suggestions[a.state.selectedIndex]

			// Template commands position cursor without executing
			if isTemplateCommand(selected) {
				a.textinput.SetValue(extractTemplatePrefix(selected))
				a.textinput.CursorEnd()
				a.state.showDropdown = false
				a.state.ghostText = ""
				a.updateAutocomplete()
				return a, nil
			}

			// Regular command - set value and fall through to execute
			a.textinput.SetValue(selected)
			a.textinput.CursorEnd()
			a.state.showDropdown = false
			a.state.ghostText = ""
		} else if a.state.ghostText != "" {
			a.textinput.SetValue(a.state.ghostText)
			a.textinput.CursorEnd()
			a.state.ghostText = ""
		}

		// Hide dropdown and pass through to parent for command execution
		a.state.showDropdown = false
		a.state.ghostText = ""

		// Pass through to underlying textinput
		var cmd tea.Cmd
		a.textinput, cmd = a.textinput.Update(msg)
		return a, cmd

	default:
		// Handle overwrite mode for printable character input when ghost text is present
		key := msg.Key()
		if key.Text != "" && a.state.ghostText != "" {
			currentValue := a.textinput.Value()
			if len(a.state.ghostText) > len(currentValue) {
				// Overwrite mode: replace next ghost character with typed character
				newValue := currentValue + key.Text
				a.textinput.SetValue(newValue)
				a.textinput.CursorEnd()
				a.updateAutocomplete()
				return a, nil
			}
		}

		// Handle regular input (including backspace, navigation, etc.)
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

// View renders the autocomplete input with ghost text overlaid at cursor position
func (a *AutocompleteInput) View() string {
	// If no ghost text, return base view
	if a.state.ghostText == "" || len(a.state.ghostText) <= len(a.textinput.Value()) {
		return a.textinput.View()
	}

	// Store original values
	originalValue := a.textinput.Value()
	originalCursorPos := a.textinput.Position()

	// Temporarily set the display value to include ghost text as plain text
	// so the textinput can properly calculate cursor position and rendering
	ghost := a.state.ghostText[len(originalValue):]
	a.textinput.SetValue(originalValue + ghost)
	a.textinput.SetCursor(originalCursorPos)

	// Render with ghost text overlaid
	view := a.textinput.View()

	// Restore original state
	a.textinput.SetValue(originalValue)
	a.textinput.SetCursor(originalCursorPos)

	return view
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
