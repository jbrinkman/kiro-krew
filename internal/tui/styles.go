package tui

import (
	"charm.land/lipgloss/v2"

	"github.com/jbrinkman/kiro-krew/internal/config"
	"github.com/jbrinkman/kiro-krew/internal/session"
)

// getColorOrFallback returns the primary color if not empty, otherwise returns fallback
func getColorOrFallback(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}

type Styles struct {
	Prompt    lipgloss.Style
	Activity  lipgloss.Style
	Separator lipgloss.Style
	Success   lipgloss.Style
	Warning   lipgloss.Style
	Error     lipgloss.Style

	// Agent status styles
	AgentSuccess lipgloss.Style
	AgentFail    lipgloss.Style

	// Tab header styles
	TabActive        lipgloss.Style
	TabInactive      lipgloss.Style
	TabInactiveHover lipgloss.Style
	TabClose         lipgloss.Style

	// Overlay styles
	OverlayBorder     lipgloss.Style
	OverlayTitle      lipgloss.Style
	OverlayContent    lipgloss.Style
	OverlayBackground lipgloss.Style
	ThemeLabel        lipgloss.Style

	// Autocomplete styles
	AutocompleteGhost    lipgloss.Style
	AutocompleteDropdown lipgloss.Style
	AutocompleteSelected lipgloss.Style
	AutocompleteError    lipgloss.Style

	// Planning tab styles
	PlanningBorder             lipgloss.Style
	PlanningUser               lipgloss.Style
	PlanningAssistant          lipgloss.Style
	PlanningInputActive        lipgloss.Style
	PlanningInputInactive      lipgloss.Style
	PlanningScrollbar          lipgloss.Style
	PlanningTimestamp          lipgloss.Style
	PlanningError              lipgloss.Style
	PlanningStreamingIndicator lipgloss.Style
	PlanningPrompt             lipgloss.Style

	// Planning tab header states (used in tab manager)
	PlanningTabActive     lipgloss.Style
	PlanningTabInactive   lipgloss.Style
	PlanningTabHover      lipgloss.Style
	PlanningTabProcessing lipgloss.Style
	PlanningTabCompleted  lipgloss.Style
	PlanningTabFailed     lipgloss.Style
	PlanningTabReadOnly   lipgloss.Style
}

func NewStyles(theme *config.Theme) *Styles {
	return &Styles{
		Prompt:    lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Prompt)),
		Activity:  lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Activity)),
		Separator: lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Separator)),
		Success:   lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Success)),
		Warning:   lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Warning)),
		Error:     lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Error)),

		// Agent status styles with fallbacks for backwards compatibility
		AgentSuccess: lipgloss.NewStyle().Foreground(lipgloss.Color(getColorOrFallback(theme.Colors.AgentSuccess, theme.Colors.Success))),
		AgentFail:    lipgloss.NewStyle().Foreground(lipgloss.Color(getColorOrFallback(theme.Colors.AgentFail, theme.Colors.Error))),

		// Tab header styles
		TabActive: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.Primary)).
			Background(lipgloss.Color(theme.Colors.Surface)).
			Bold(true).
			Padding(0, 1),
		TabInactive: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.TextMuted)).
			Padding(0, 1),
		TabInactiveHover: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.TextMuted)).
			Padding(0, 1).
			Underline(true),
		TabClose: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.Warning)).
			Bold(true),

		OverlayBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Colors.Primary)).
			Background(lipgloss.Color(theme.Colors.Surface)).
			Padding(1, 2),
		OverlayTitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.Primary)).
			Bold(true),
		OverlayContent: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.TextPrimary)).
			Background(lipgloss.Color(theme.Colors.Surface)),
		OverlayBackground: lipgloss.NewStyle().
			Background(lipgloss.Color(theme.Colors.Surface)),
		ThemeLabel: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.TextMuted)).
			Italic(true),

		// Autocomplete styles
		AutocompleteGhost: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.TextMuted)),
		AutocompleteDropdown: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Colors.Primary)).
			Background(lipgloss.Color(theme.Colors.Surface)).
			Padding(0, 1),
		AutocompleteSelected: lipgloss.NewStyle().
			Background(lipgloss.Color(theme.Colors.Primary)).
			Foreground(lipgloss.Color(theme.Colors.Surface)),
		AutocompleteError: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.Error)).
			Bold(true),

		// Planning tab styles - comprehensive styling for all states and elements
		PlanningBorder: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(getColorOrFallback(theme.Colors.Primary, theme.Colors.TextMuted))).
			Background(lipgloss.Color(theme.Colors.Background)),
		PlanningUser: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.Primary)).
			Bold(true).
			MarginBottom(1),
		PlanningAssistant: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.TextPrimary)).
			MarginBottom(1),
		PlanningInputActive: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Colors.Primary)).
			Background(lipgloss.Color(getColorOrFallback(theme.Colors.Surface, theme.Colors.Background))).
			Padding(1).
			MarginTop(1),
		PlanningInputInactive: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Colors.TextMuted)).
			Background(lipgloss.Color(getColorOrFallback(theme.Colors.Surface, theme.Colors.Background))).
			Padding(1).
			MarginTop(1),
		PlanningScrollbar: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.TextMuted)).
			Background(lipgloss.Color(getColorOrFallback(theme.Colors.Surface, theme.Colors.Background))),
		PlanningTimestamp: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.TextMuted)).
			Italic(true).
			MarginLeft(1),
		PlanningError: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.Error)).
			Background(lipgloss.Color(getColorOrFallback(theme.Colors.Surface, theme.Colors.Background))).
			Bold(true).
			Padding(0, 1),
		PlanningStreamingIndicator: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.Warning)).
			Bold(true).
			Blink(true),
		PlanningPrompt: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.Primary)).
			Bold(true),

		// Planning tab header states for comprehensive theme compatibility
		PlanningTabActive: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.Primary)).
			Background(lipgloss.Color(getColorOrFallback(theme.Colors.Surface, theme.Colors.Background))).
			Bold(true).
			Padding(0, 1).
			Border(lipgloss.Border{
				Bottom: "▔",
			}).
			BorderForeground(lipgloss.Color(theme.Colors.Primary)),
		PlanningTabInactive: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.TextMuted)).
			Padding(0, 1),
		PlanningTabHover: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.TextPrimary)).
			Background(lipgloss.Color(getColorOrFallback(theme.Colors.Surface, theme.Colors.Background))).
			Padding(0, 1).
			Underline(true),
		PlanningTabProcessing: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.Warning)).
			Padding(0, 1).
			Bold(true),
		PlanningTabCompleted: lipgloss.NewStyle().
			Foreground(lipgloss.Color(getColorOrFallback(theme.Colors.AgentSuccess, theme.Colors.Success))).
			Padding(0, 1).
			Bold(true),
		PlanningTabFailed: lipgloss.NewStyle().
			Foreground(lipgloss.Color(getColorOrFallback(theme.Colors.AgentFail, theme.Colors.Error))).
			Padding(0, 1).
			Bold(true),
		PlanningTabReadOnly: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.TextSecondary)).
			Padding(0, 1).
			Italic(true),
	}
}

// Planning Tab Responsive Styling Methods

// GetPlanningInputStyle returns appropriate input style based on focus and terminal width
func (s *Styles) GetPlanningInputStyle(focused bool, width int) lipgloss.Style {
	var baseStyle lipgloss.Style
	if focused {
		baseStyle = s.PlanningInputActive
	} else {
		baseStyle = s.PlanningInputInactive
	}

	// Responsive adjustments for narrow terminals
	if width < 60 {
		return baseStyle.
			Padding(0, 1). // Reduce padding on narrow screens
			MaxWidth(width - 4)
	}

	return baseStyle
}

// GetPlanningBorderStyle returns appropriate border style based on terminal width
func (s *Styles) GetPlanningBorderStyle(width int) lipgloss.Style {
	if width < 60 {
		// Use simpler border for narrow terminals
		return s.PlanningBorder.
			Border(lipgloss.NormalBorder()).
			Padding(0)
	}

	return s.PlanningBorder
}

// GetPlanningTabStyle returns appropriate tab header style based on state and hover
func (s *Styles) GetPlanningTabStyle(state session.PlanningTabState, isActive, isHovered bool) lipgloss.Style {
	switch {
	case isActive:
		return s.PlanningTabActive
	case isHovered:
		return s.PlanningTabHover
	case state == session.PlanningStateActive:
		return s.PlanningTabProcessing
	case state == session.PlanningStateCompleted:
		return s.PlanningTabCompleted
	case state == session.PlanningStateFailed:
		return s.PlanningTabFailed
	case state == session.PlanningStateReadOnly:
		return s.PlanningTabReadOnly
	default:
		return s.PlanningTabInactive
	}
}

// GetPlanningMessageStyle returns appropriate message style with responsive adjustments
func (s *Styles) GetPlanningMessageStyle(role string, width int) lipgloss.Style {
	var baseStyle lipgloss.Style

	if role == "user" {
		baseStyle = s.PlanningUser
	} else {
		baseStyle = s.PlanningAssistant
	}

	// Responsive adjustments
	if width < 60 {
		return baseStyle.
			MarginBottom(0). // Reduce spacing on narrow screens
			MaxWidth(width - 4)
	}

	return baseStyle
}
