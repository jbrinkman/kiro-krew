package tui

import (
	tea "charm.land/bubbletea/v2"
)

// MainTab wraps the existing console functionality in a tab interface
type MainTab struct {
	id       string
	width    int
	height   int
	baseView string // Stores the rendered base view from TUI
}

// NewMainTab creates a new main tab
func NewMainTab() *MainTab {
	return &MainTab{
		id: "main",
	}
}

// ID returns the tab identifier
func (mt *MainTab) ID() string {
	return mt.id
}

// Type returns the tab type
func (mt *MainTab) Type() TabType {
	return TabTypeMain
}

// Title returns the tab title
func (mt *MainTab) Title() string {
	return "Main TUI"
}

// IsClosable returns whether this tab can be closed
func (mt *MainTab) IsClosable() bool {
	return false // Main tab is never closable
}

// View returns the tab's rendered content
func (mt *MainTab) View() string {
	return mt.baseView
}

// Update handles messages for the main tab
func (mt *MainTab) Update(msg tea.Msg) (Tab, tea.Cmd) {
	// Main tab doesn't handle updates directly - the main TUI model handles them
	return mt, nil
}

// Resize updates the tab dimensions
func (mt *MainTab) Resize(width, height int) {
	mt.width = width
	mt.height = height
}

// SetBaseView updates the base view content from the main TUI
func (mt *MainTab) SetBaseView(view string) {
	mt.baseView = view
}