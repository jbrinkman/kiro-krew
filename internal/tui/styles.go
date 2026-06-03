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
	
	// Overlay styles
	OverlayBorder     lipgloss.Style
	OverlayTitle      lipgloss.Style
	OverlayContent    lipgloss.Style
	OverlayBackground lipgloss.Style
	ThemeLabel        lipgloss.Style
}

func NewStyles(theme *config.Theme) *Styles {
	return &Styles{
		Prompt:    lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Prompt)),
		Activity:  lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Activity)),
		Separator: lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Separator)),
		Success:   lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Success)),
		Warning:   lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Warning)),
		Error:     lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Error)),
		
		OverlayBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Colors.Primary)).
			Background(lipgloss.Color(theme.Colors.Surface)).
			Padding(1, 2),
		OverlayTitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.Primary)).
			Bold(true),
		OverlayContent: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.TextPrimary)),
		OverlayBackground: lipgloss.NewStyle().
			Background(lipgloss.Color(theme.Colors.Surface)),
		ThemeLabel: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Colors.TextMuted)).
			Italic(true),
	}
}
