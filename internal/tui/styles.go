package tui

import (
	"charm.land/lipgloss/v2"

	"github.com/jbrinkman/kiro-krew/internal/config"
)

type Styles struct {
	Prompt    lipgloss.Style
	Activity  lipgloss.Style
	Separator lipgloss.Style
	Success   lipgloss.Style
	Warning   lipgloss.Style
	Error     lipgloss.Style

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
}

func NewStyles(theme *config.Theme) *Styles {
	return &Styles{
		Prompt:    lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Prompt)),
		Activity:  lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Activity)),
		Separator: lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Separator)),
		Success:   lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Success)),
		Warning:   lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Warning)),
		Error:     lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Error)),

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
	}
}
