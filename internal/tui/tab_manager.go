package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// TabManager manages the lifecycle and state of all tabs
type TabManager struct {
	tabs      []Tab
	activeTab int
	width     int
	height    int
}

// NewTabManager creates a new tab manager
func NewTabManager() *TabManager {
	return &TabManager{
		tabs:      make([]Tab, 0),
		activeTab: 0,
	}
}

// AddTab adds a new tab to the manager
func (tm *TabManager) AddTab(tab Tab) {
	tm.tabs = append(tm.tabs, tab)
	if tm.width > 0 && tm.height > 0 {
		tab.Resize(tm.width, tm.height)
	}
	// If this is the first tab, set it as active
	if len(tm.tabs) == 1 {
		tm.activeTab = 0
	}
}

// RemoveTab removes a tab by ID if it's closable
func (tm *TabManager) RemoveTab(id string) bool {
	for i, tab := range tm.tabs {
		if tab.ID() == id && tab.IsClosable() {
			tm.tabs = append(tm.tabs[:i], tm.tabs[i+1:]...)
			if tm.activeTab >= len(tm.tabs) && len(tm.tabs) > 0 {
				tm.activeTab = len(tm.tabs) - 1
			}
			return true
		}
	}
	return false
}

// SetActiveTab sets the active tab by index
func (tm *TabManager) SetActiveTab(index int) {
	if index >= 0 && index < len(tm.tabs) {
		tm.activeTab = index
	}
}

// GetActiveTab returns the currently active tab
func (tm *TabManager) GetActiveTab() Tab {
	if len(tm.tabs) == 0 || tm.activeTab >= len(tm.tabs) {
		return nil
	}
	return tm.tabs[tm.activeTab]
}

// GetTabs returns all tabs
func (tm *TabManager) GetTabs() []Tab {
	return tm.tabs
}

// Update forwards update messages to the active tab
func (tm *TabManager) Update(msg tea.Msg) tea.Cmd {
	if activeTab := tm.GetActiveTab(); activeTab != nil {
		updated, cmd := activeTab.Update(msg)
		if updated != nil {
			tm.tabs[tm.activeTab] = updated
		}
		return cmd
	}
	return nil
}

// Resize resizes all tabs
func (tm *TabManager) Resize(width, height int) {
	tm.width = width
	tm.height = height
	for _, tab := range tm.tabs {
		tab.Resize(width, height)
	}
}

// RenderCurrentView renders the active tab's view
func (tm *TabManager) RenderCurrentView() string {
	if activeTab := tm.GetActiveTab(); activeTab != nil {
		return activeTab.View()
	}
	return ""
}

// NextTab switches to the next tab
func (tm *TabManager) NextTab() {
	if len(tm.tabs) > 1 {
		tm.activeTab = (tm.activeTab + 1) % len(tm.tabs)
	}
}

// PreviousTab switches to the previous tab
func (tm *TabManager) PreviousTab() {
	if len(tm.tabs) > 1 {
		tm.activeTab = (tm.activeTab - 1 + len(tm.tabs)) % len(tm.tabs)
	}
}

// FindTabByAgentID finds tab index by agent ID
func (tm *TabManager) FindTabByAgentID(agentID string) int {
	for i, tab := range tm.tabs {
		if tab.Type() == TabTypeAgent && tab.ID() == "agent-"+agentID {
			return i
		}
	}
	return -1
}

// CloseTab closes tab at index
func (tm *TabManager) CloseTab(index int) bool {
	if index < 0 || index >= len(tm.tabs) || !tm.tabs[index].IsClosable() {
		return false
	}
	
	tm.tabs = append(tm.tabs[:index], tm.tabs[index+1:]...)
	
	// Maintain active tab index
	if tm.activeTab >= len(tm.tabs) && len(tm.tabs) > 0 {
		tm.activeTab = len(tm.tabs) - 1
	} else if tm.activeTab > index {
		tm.activeTab--
	}
	
	return true
}

// CloseTabByID closes tab by ID
func (tm *TabManager) CloseTabByID(tabID string) bool {
	for i, tab := range tm.tabs {
		if tab.ID() == tabID {
			return tm.CloseTab(i)
		}
	}
	return false
}

// GetActiveTabIndex returns the index of the currently active tab
func (tm *TabManager) GetActiveTabIndex() int {
	return tm.activeTab
}

// CloseCurrentTab closes the currently active tab if it's closable
func (tm *TabManager) CloseCurrentTab() bool {
	return tm.CloseTab(tm.activeTab)
}

// ToggleView switches between main tab and first agent tab (for F2 compatibility)
func (tm *TabManager) ToggleView() {
	if len(tm.tabs) < 2 {
		return
	}
	
	// If on main tab, switch to first agent tab
	if tm.activeTab == 0 {
		for i, tab := range tm.tabs {
			if tab.Type() == TabTypeAgent {
				tm.activeTab = i
				return
			}
		}
	} else {
		// Switch back to main tab
		tm.activeTab = 0
	}
}

// RenderTabHeaders renders visual tab headers showing all tabs with active highlighting and close buttons
func (tm *TabManager) RenderTabHeaders(styles *Styles) string {
	if len(tm.tabs) == 0 {
		return ""
	}

	var tabHeaders []string
	
	for i, tab := range tm.tabs {
		title := tab.Title()
		
		// Truncate long titles
		if len(title) > 15 {
			title = title[:12] + "..."
		}
		
		var tabContent string
		if tab.IsClosable() {
			// Add close button for closable tabs
			closeBtn := styles.TabClose.Render(" ×")
			tabContent = title + closeBtn
		} else {
			tabContent = title
		}
		
		// Style based on active state
		if i == tm.activeTab {
			tabHeaders = append(tabHeaders, styles.TabActive.Render(tabContent))
		} else {
			tabHeaders = append(tabHeaders, styles.TabInactive.Render(tabContent))
		}
	}
	
	return strings.Join(tabHeaders, styles.Separator.Render("│"))
}

// HandleTabHeaderClick handles mouse clicks on tab headers
func (tm *TabManager) HandleTabHeaderClick(x int) bool {
	if len(tm.tabs) == 0 {
		return false
	}

	position := 0
	for i, tab := range tm.tabs {
		title := tab.Title()
		if len(title) > 15 {
			title = title[:12] + "..."
		}
		
		tabWidth := len(title)
		closeButtonStart := position + tabWidth
		
		if tab.IsClosable() {
			tabWidth += 2 // " ×"
		}
		
		// Check if click is within this tab
		if x >= position && x < position+tabWidth {
			if tab.IsClosable() && x >= closeButtonStart {
				// Clicked on close button area
				tm.CloseTab(i)
			} else {
				// Clicked on tab content, switch to this tab
				tm.SetActiveTab(i)
			}
			return true
		}
		
		position += tabWidth + 1 // +1 for separator
	}
	
	return false
}
