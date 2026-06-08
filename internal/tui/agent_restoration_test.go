package tui

import (
	"testing"

	"github.com/jbrinkman/kiro-krew/internal/agent"
	"github.com/jbrinkman/kiro-krew/internal/config"
)

func TestRestoreOrFocusAgentTab(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Theme: "test",
		LoadedTheme: &config.Theme{
			Name: "test",
		},
	}
	manager := agent.NewManager(cfg)
	styles := NewStyles(cfg.LoadedTheme)
	tabManager := NewTabManager()
	
	// Add main tab
	mainTab := NewMainTab()
	tabManager.AddTab(mainTab)
	
	agentID := "test-agent-123"
	
	// Test 1: Create new agent tab when none exists
	t.Run("creates new agent tab", func(t *testing.T) {
		result := tabManager.RestoreOrFocusAgentTab(agentID, manager, styles)
		if !result {
			t.Error("Expected RestoreOrFocusAgentTab to return true")
		}
		
		if len(tabManager.GetTabs()) != 2 {
			t.Errorf("Expected 2 tabs, got %d", len(tabManager.GetTabs()))
		}
		
		activeIndex := tabManager.GetActiveTabIndex()
		if activeIndex != 1 {
			t.Errorf("Expected active tab to be 1, got %d", activeIndex)
		}
		
		activeTab := tabManager.GetActiveTab()
		if activeTab.Type() != TabTypeAgent {
			t.Error("Expected active tab to be an agent tab")
		}
	})
	
	// Test 2: Focus existing agent tab
	t.Run("focuses existing agent tab", func(t *testing.T) {
		// Switch to main tab first
		tabManager.SetActiveTab(0)
		
		result := tabManager.RestoreOrFocusAgentTab(agentID, manager, styles)
		if !result {
			t.Error("Expected RestoreOrFocusAgentTab to return true")
		}
		
		// Should still have 2 tabs (no new tab created)
		if len(tabManager.GetTabs()) != 2 {
			t.Errorf("Expected 2 tabs, got %d", len(tabManager.GetTabs()))
		}
		
		// Should focus the existing agent tab
		activeIndex := tabManager.GetActiveTabIndex()
		if activeIndex != 1 {
			t.Errorf("Expected active tab to be 1, got %d", activeIndex)
		}
	})
}

func TestFindTabByAgentID(t *testing.T) {
	tabManager := NewTabManager()
	cfg := &config.Config{
		Theme: "test",
		LoadedTheme: &config.Theme{
			Name: "test",
		},
	}
	manager := agent.NewManager(cfg)
	styles := NewStyles(cfg.LoadedTheme)
	
	// Add main tab
	mainTab := NewMainTab()
	tabManager.AddTab(mainTab)
	
	// Test agent ID
	agentID := "test-agent-456"
	
	// Should not find non-existent agent tab
	index := tabManager.FindTabByAgentID(agentID)
	if index != -1 {
		t.Errorf("Expected -1 for non-existent agent, got %d", index)
	}
	
	// Add agent tab
	agentTab := NewAgentTab(agentID, manager, styles)
	tabManager.AddTab(agentTab)
	
	// Should find the agent tab
	index = tabManager.FindTabByAgentID(agentID)
	if index != 1 {
		t.Errorf("Expected 1 for agent tab index, got %d", index)
	}
}