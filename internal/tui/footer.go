package tui

import (
	"fmt"
	"strings"

	"github.com/jbrinkman/kiro-krew/internal/config"
	"github.com/jbrinkman/kiro-krew/internal/session"
)

// FooterManager manages the two-row footer display system
type FooterManager struct {
	styles            *Styles
	config            *config.Config
	contextTracker    *ContextTracker
	autocompleteInput *AutocompleteInput
	tabManager        *TabManager
	width             int
	height            int
}

// FooterContent represents the structured content for the footer
type FooterContent struct {
	InputRow  string // Row 1: Command entry area
	StatusRow string // Row 2: Contextual information
}

// NewFooterManager creates a new footer manager
func NewFooterManager(styles *Styles, config *config.Config, autocompleteInput *AutocompleteInput, tabManager *TabManager) *FooterManager {
	return &FooterManager{
		styles:            styles,
		config:            config,
		contextTracker:    NewContextTracker(),
		autocompleteInput: autocompleteInput,
		tabManager:        tabManager,
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

// renderInputRow creates the command entry row (Row 1: input only, no theme label)
func (fm *FooterManager) renderInputRow() string {
	if fm.autocompleteInput == nil {
		return ""
	}

	// Use full width for the prompt input
	promptWidth := fm.width
	if promptWidth < 1 {
		promptWidth = 1
	}

	promptInput := fm.autocompleteInput.View()
	return fm.styles.Prompt.Width(promptWidth).Render(promptInput)
}

// renderStatusRow creates the contextual information row based on tab type
func (fm *FooterManager) renderStatusRow(activeTabType TabType) string {
	// Base information shown on all tabs
	baseInfo := fm.renderBaseInfo()

	// Additional information for planning tabs
	if activeTabType == TabTypePlanning {
		planningInfo := fm.renderPlanningInfo()
		planningStatusInfo := fm.renderPlanningStatusInfo()

		// Combine all planning information
		var planningInfoParts []string
		if planningInfo != "" {
			planningInfoParts = append(planningInfoParts, planningInfo)
		}
		if planningStatusInfo != "" {
			planningInfoParts = append(planningInfoParts, planningStatusInfo)
		}

		if len(planningInfoParts) > 0 {
			combinedPlanningInfo := strings.Join(planningInfoParts, " | ")
			return fm.joinStatusInfo(baseInfo, combinedPlanningInfo)
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
	// Only show context info if context tracker is active
	if !fm.contextTracker.IsActive() {
		return ""
	}

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

// renderPlanningStatusInfo renders the status of the active planning tab
func (fm *FooterManager) renderPlanningStatusInfo() string {
	if fm.tabManager == nil {
		return ""
	}

	activeTab := fm.tabManager.GetActiveTab()
	if activeTab == nil || activeTab.Type() != TabTypePlanning {
		return ""
	}

	planningTab, ok := activeTab.(*PlanningTab)
	if !ok {
		return ""
	}

	state := planningTab.GetState()

	// Format status with appropriate visual indicator
	var statusText string
	switch state {
	case session.PlanningStateIdle:
		statusText = "status: ready"
	case session.PlanningStateActive:
		statusText = "status: ● processing"
	case session.PlanningStateCompleted:
		statusText = "status: ✓ completed"
	case session.PlanningStateFailed:
		statusText = "status: ✗ failed"
	case session.PlanningStateReadOnly:
		statusText = "status: 🔒 read-only"
	default:
		statusText = "status: ready"
	}

	// Add message count if there are messages
	messageCount := planningTab.GetMessageCount()
	if messageCount > 0 {
		statusText += fmt.Sprintf(" (%d msg%s)", messageCount, func() string {
			if messageCount != 1 {
				return "s"
			}
			return ""
		}())
	}

	return statusText
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
