package plan

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// AgentRegistry maintains a mapping of agent names to their configuration file paths
type AgentRegistry struct {
	agents map[string]string // agent name -> config file path
	mu     sync.RWMutex
}

// NewAgentRegistry creates a new empty agent registry
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		agents: make(map[string]string),
	}
}

// DiscoverAgents reads all *.json files from the specified directory and builds
// an agent registry by extracting agent names from their configurations
func DiscoverAgents(agentDir string) (*AgentRegistry, error) {
	registry := NewAgentRegistry()

	// Find all .json files in the agent directory
	pattern := filepath.Join(agentDir, "*.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob agent files: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no agent configuration files found in %s", agentDir)
	}

	// Parse each agent configuration file
	for _, filePath := range files {
		name, err := extractAgentName(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to extract agent name from %s: %w", filePath, err)
		}

		// Check for duplicate agent names
		if _, exists := registry.agents[name]; exists {
			return nil, fmt.Errorf("duplicate agent name '%s' found in %s (already registered from %s)",
				name, filePath, registry.agents[name])
		}

		registry.agents[name] = filePath
	}

	return registry, nil
}

// extractAgentName reads a JSON file and extracts the "name" field
func extractAgentName(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	var config struct {
		Name string `json:"name"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	if config.Name == "" {
		return "", fmt.Errorf("agent name field is empty")
	}

	return config.Name, nil
}

// Contains checks if an agent with the given name exists in the registry
func (r *AgentRegistry) Contains(agentName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.agents[agentName]
	return exists
}

// GetConfigPath returns the configuration file path for a given agent name
func (r *AgentRegistry) GetConfigPath(agentName string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	path, exists := r.agents[agentName]
	return path, exists
}

// GetAgentNames returns a slice of all registered agent names
func (r *AgentRegistry) GetAgentNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.agents))
	for name := range r.agents {
		names = append(names, name)
	}
	return names
}

// Count returns the number of registered agents
func (r *AgentRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.agents)
}
