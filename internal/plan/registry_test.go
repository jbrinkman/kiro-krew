package plan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewAgentRegistry(t *testing.T) {
	registry := NewAgentRegistry()
	if registry == nil {
		t.Fatal("expected non-nil registry")
	}
	if registry.Count() != 0 {
		t.Errorf("expected empty registry, got count: %d", registry.Count())
	}
}

func TestDiscoverAgents_ValidDirectory(t *testing.T) {
	// Create a temporary directory with test agent configs
	tmpDir := t.TempDir()

	// Create test agent files
	agents := []struct {
		filename string
		content  string
	}{
		{
			filename: "builder.json",
			content:  `{"name": "builder", "description": "Builder agent"}`,
		},
		{
			filename: "validator.json",
			content:  `{"name": "validator", "description": "Validator agent"}`,
		},
		{
			filename: "architect.json",
			content:  `{"name": "architect", "description": "Architect agent"}`,
		},
	}

	for _, agent := range agents {
		path := filepath.Join(tmpDir, agent.filename)
		if err := os.WriteFile(path, []byte(agent.content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
	}

	// Discover agents
	registry, err := DiscoverAgents(tmpDir)
	if err != nil {
		t.Fatalf("expected successful discovery, got error: %v", err)
	}

	// Verify count
	if registry.Count() != 3 {
		t.Errorf("expected 3 agents, got %d", registry.Count())
	}

	// Verify each agent is registered
	for _, agent := range agents {
		config := struct{ Name string }{}
		// Extract expected name from content (simplified)
		if agent.filename == "builder.json" {
			config.Name = "builder"
		} else if agent.filename == "validator.json" {
			config.Name = "validator"
		} else if agent.filename == "architect.json" {
			config.Name = "architect"
		}

		if !registry.Contains(config.Name) {
			t.Errorf("expected registry to contain agent '%s'", config.Name)
		}

		path, exists := registry.GetConfigPath(config.Name)
		if !exists {
			t.Errorf("expected to find config path for agent '%s'", config.Name)
		}
		if path == "" {
			t.Errorf("expected non-empty config path for agent '%s'", config.Name)
		}
	}
}

func TestDiscoverAgents_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Attempt to discover agents from empty directory
	_, err := DiscoverAgents(tmpDir)
	if err == nil {
		t.Error("expected error for empty directory")
	}
}

func TestDiscoverAgents_NonexistentDirectory(t *testing.T) {
	_, err := DiscoverAgents("/nonexistent/directory/path")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestDiscoverAgents_DuplicateAgentNames(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two files with the same agent name
	files := []struct {
		filename string
		content  string
	}{
		{
			filename: "builder1.json",
			content:  `{"name": "builder", "description": "First builder"}`,
		},
		{
			filename: "builder2.json",
			content:  `{"name": "builder", "description": "Second builder"}`,
		},
	}

	for _, file := range files {
		path := filepath.Join(tmpDir, file.filename)
		if err := os.WriteFile(path, []byte(file.content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
	}

	// Attempt to discover agents - should fail due to duplicate
	_, err := DiscoverAgents(tmpDir)
	if err == nil {
		t.Error("expected error for duplicate agent names")
	}
}

func TestDiscoverAgents_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file with invalid JSON
	path := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(path, []byte(`{invalid json`), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := DiscoverAgents(tmpDir)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestDiscoverAgents_MissingNameField(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid JSON file without a name field
	path := filepath.Join(tmpDir, "noname.json")
	if err := os.WriteFile(path, []byte(`{"description": "Agent without name"}`), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := DiscoverAgents(tmpDir)
	if err == nil {
		t.Error("expected error for missing name field")
	}
}

func TestAgentRegistry_Contains(t *testing.T) {
	registry := NewAgentRegistry()
	registry.agents["builder"] = "/path/to/builder.json"
	registry.agents["validator"] = "/path/to/validator.json"

	tests := []struct {
		name     string
		expected bool
	}{
		{"builder", true},
		{"validator", true},
		{"architect", false},
		{"", false},
	}

	for _, tt := range tests {
		result := registry.Contains(tt.name)
		if result != tt.expected {
			t.Errorf("Contains(%q) = %v, expected %v", tt.name, result, tt.expected)
		}
	}
}

func TestAgentRegistry_GetConfigPath(t *testing.T) {
	registry := NewAgentRegistry()
	expectedPath := "/path/to/builder.json"
	registry.agents["builder"] = expectedPath

	// Test existing agent
	path, exists := registry.GetConfigPath("builder")
	if !exists {
		t.Error("expected to find builder")
	}
	if path != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, path)
	}

	// Test non-existent agent
	path, exists = registry.GetConfigPath("nonexistent")
	if exists {
		t.Error("expected not to find nonexistent agent")
	}
	if path != "" {
		t.Errorf("expected empty path for non-existent agent, got %s", path)
	}
}

func TestAgentRegistry_GetAgentNames(t *testing.T) {
	registry := NewAgentRegistry()
	registry.agents["builder"] = "/path/to/builder.json"
	registry.agents["validator"] = "/path/to/validator.json"
	registry.agents["architect"] = "/path/to/architect.json"

	names := registry.GetAgentNames()
	if len(names) != 3 {
		t.Errorf("expected 3 names, got %d", len(names))
	}

	// Check that all expected names are present (order doesn't matter)
	expectedNames := map[string]bool{
		"builder":   false,
		"validator": false,
		"architect": false,
	}

	for _, name := range names {
		if _, ok := expectedNames[name]; ok {
			expectedNames[name] = true
		} else {
			t.Errorf("unexpected agent name: %s", name)
		}
	}

	for name, found := range expectedNames {
		if !found {
			t.Errorf("expected to find agent name: %s", name)
		}
	}
}

func TestAgentRegistry_Count(t *testing.T) {
	registry := NewAgentRegistry()
	if registry.Count() != 0 {
		t.Errorf("expected count 0, got %d", registry.Count())
	}

	registry.agents["builder"] = "/path/to/builder.json"
	if registry.Count() != 1 {
		t.Errorf("expected count 1, got %d", registry.Count())
	}

	registry.agents["validator"] = "/path/to/validator.json"
	registry.agents["architect"] = "/path/to/architect.json"
	if registry.Count() != 3 {
		t.Errorf("expected count 3, got %d", registry.Count())
	}
}
