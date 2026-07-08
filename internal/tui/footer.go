package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/jbrinkman/kiro-krew/internal/config"
)

// FooterManager manages the two-row footer display system
type FooterManager struct {
	styles            *Styles
	config            *config.Config
	contextTracker    *ContextTracker
	autocompleteInput *AutocompleteInput
	width             int
	height            int
}

// FooterContent represents the structured content for the footer
type FooterContent struct {
	InputRow  string // Row 1: Command entry area
	StatusRow string // Row 2: Contextual information
}

// NewFooterManager creates a new footer manager
func NewFooterManager(styles *Styles, config *config.Config, autocompleteInput *AutocompleteInput) *FooterManager {
	return &FooterManager{
		styles:            styles,
		config:            config,
		contextTracker:    NewContextTracker(),
		autocompleteInput: autocompleteInput,
	}
}

// Resize updates the footer dimensions
func (fm *FooterManager) Resize(width, height int) {
	fm.width = width
	fm.height = height
}

// GetContextTracker returns the context tracker for external access
func (fm *FooterManager) GetContextTracker() *ContextTracker {
	return fm.contextTracker
}

// RenderFooter renders the complete two-row footer
func (fm *FooterManager) RenderFooter(activeTabType TabType) FooterContent {
	// Row 1: Command entry area (same as existing prompt functionality)
	inputRow := fm.renderInputRow()

	// Row 2: Contextual information based on tab type
	statusRow := fm.renderStatusRow(activeTabType)

	return FooterContent{
		InputRow:  inputRow,
		StatusRow: statusRow,
	}
}

// renderInputRow creates the command entry row with responsive sizing
func (fm *FooterManager) renderInputRow() string {
	if fm.autocompleteInput == nil {
		return ""
	}

	// Get theme label for width calculations
	themeLabel := fm.styles.ThemeLabel.Render(fmt.Sprintf("theme: %s", fm.config.Theme))
	themeLabelWidth := lipgloss.Width(themeLabel)

	// Calculate responsive sizing
	// If terminal is too narrow for both prompt + theme label, hide theme label
	var promptWidth int
	var showThemeLabel bool

	if fm.width > 0 && themeLabelWidth+20 > fm.width {
		// Terminal too narrow - hide theme label and use full width for prompt
		promptWidth = fm.width
		showThemeLabel = false
	} else {
		// Normal sizing - reserve space for theme label
		promptWidth = fm.width - themeLabelWidth
		showThemeLabel = true

		// Ensure minimum prompt width
		if fm.width >= 20 && promptWidth < 20 {
			promptWidth = 20
		}
		if promptWidth < 1 {
			promptWidth = 1
		}
	}

	// Create prompt with responsive width
	promptInput := fm.autocompleteInput.View()
	prompt := fm.styles.Prompt.Width(promptWidth).Render(promptInput)

	// Join prompt with theme label if there's space
	if showThemeLabel {
		return lipgloss.JoinHorizontal(lipgloss.Top, prompt, themeLabel)
	}

	return prompt
}

// renderStatusRow creates the contextual information row based on tab type
func (fm *FooterManager) renderStatusRow(activeTabType TabType) string {
	// Base information shown on all tabs
	baseInfo := fm.renderBaseInfo()

	// Additional information for planning tabs
	if activeTabType == TabTypeMain && fm.contextTracker.IsActive() {
		planningInfo := fm.renderPlanningInfo()
		if planningInfo != "" {
			return fm.joinStatusInfo(baseInfo, planningInfo)
		}
	}

	return baseInfo
}

// renderBaseInfo renders the base information shown on all tabs
func (fm *FooterManager) renderBaseInfo() string {
	return fmt.Sprintf("theme: %s", fm.config.Theme)
}

// renderPlanningInfo renders additional information for planning tabs
func (fm *FooterManager) renderPlanningInfo() string {
	planningContext := fm.contextTracker.GetPlanningContext()
	if planningContext == nil {
		return ""
	}

	var parts []string

	// Context usage
	if contextUsage := fm.contextTracker.FormatContextUsage(); contextUsage != "" {
		parts = append(parts, contextUsage)
	}

	// Model information
	if planningContext.Model != "" {
		parts = append(parts, fmt.Sprintf("model: %s", planningContext.Model))
	}

	// Directory information with folder icon
	if planningContext.Directory != "" {
		parts = append(parts, fmt.Sprintf("📁 %s", planningContext.Directory))
	}

	return strings.Join(parts, " | ")
}

// joinStatusInfo combines base and additional status information
func (fm *FooterManager) joinStatusInfo(baseInfo, additionalInfo string) string {
	if baseInfo == "" {
		return additionalInfo
	}
	if additionalInfo == "" {
		return baseInfo
	}

	return fmt.Sprintf("%s | %s", baseInfo, additionalInfo)
}

// RenderWithSeparator renders the footer with separator line above it
func (fm *FooterManager) RenderWithSeparator(activeTabType TabType) string {
	footer := fm.RenderFooter(activeTabType)

	// Create separator line
	separator := fm.styles.Separator.Render(strings.Repeat("─", fm.width))

	// Combine separator with footer rows
	var result []string
	result = append(result, separator)
	result = append(result, footer.InputRow)

	// Only add status row if it has content
	if strings.TrimSpace(footer.StatusRow) != "" {
		result = append(result, fm.styles.ThemeLabel.Render(footer.StatusRow))
	}

	return strings.Join(result, "\n")
}

// RenderDropdownWithFooter handles autocomplete dropdown rendering with footer integration
func (fm *FooterManager) RenderDropdownWithFooter(activeTabType TabType) (string, int) {
	if fm.autocompleteInput == nil {
		return fm.RenderWithSeparator(activeTabType), 0
	}

	// Get dropdown content and calculate height impact
	var dropdownContent string
	var dropdownHeight int

	if fm.autocompleteInput.IsDropdownVisible() {
		dropdownContent = fm.autocompleteInput.ViewDropdown()
		if dropdownContent != "" {
			dropdownHeight = len(strings.Split(dropdownContent, "\n"))
		}
	}

	// Render footer components
	footer := fm.RenderFooter(activeTabType)
	separator := fm.styles.Separator.Render(strings.Repeat("─", fm.width))

	// Construct the final layout
	var result []string
	result = append(result, separator)

	// Add dropdown above input if present
	if dropdownContent != "" {
		result = append(result, dropdownContent)
	}

	result = append(result, footer.InputRow)

	// Add status row if it has content
	if strings.TrimSpace(footer.StatusRow) != "" {
		result = append(result, fm.styles.ThemeLabel.Render(footer.StatusRow))
	}

	return strings.Join(result, "\n"), dropdownHeight
}

// GetFooterHeight returns the total height occupied by the footer
func (fm *FooterManager) GetFooterHeight() int {
	// Base height: separator (1) + input row (1) + status row (1) = 3
	return 3
}

// GetFooterHeightWithDropdown returns the footer height including dropdown
func (fm *FooterManager) GetFooterHeightWithDropdown() int {
	baseHeight := fm.GetFooterHeight()

	if fm.autocompleteInput != nil && fm.autocompleteInput.IsDropdownVisible() {
		dropdownContent := fm.autocompleteInput.ViewDropdown()
		if dropdownContent != "" {
			dropdownHeight := len(strings.Split(dropdownContent, "\n"))
			return baseHeight + dropdownHeight
		}
	}

	return baseHeight
}
