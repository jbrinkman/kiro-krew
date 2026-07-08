package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ContextUsage represents real-time context information for Planning Tabs
type ContextUsage struct {
	Used  int // Current context usage in tokens
	Total int // Total context limit in tokens
}

// PlanningContext holds context information specific to planning sessions
type PlanningContext struct {
	Model     string // Currently active model (e.g., "claude-sonnet-4")
	Directory string // Current working directory
	Usage     ContextUsage
}

// ContextTracker manages real-time context information for display in footer
type ContextTracker struct {
	planningContext *PlanningContext
	lastUpdate      time.Time
	updateInterval  time.Duration
	// contextLimitOverride allows ACP session metadata to override hardcoded limits
	contextLimitOverride int
}

// NewContextTracker creates a new context tracker with default settings
func NewContextTracker() *ContextTracker {
	return &ContextTracker{
		planningContext: nil,
		updateInterval:  time.Second, // Update every second
	}
}

// StartPlanningSession initializes context tracking for a planning session
func (ct *ContextTracker) StartPlanningSession(model string) {
	currentDir, err := os.Getwd()
	if err != nil {
		// Fallback to home directory display
		if homeDir, homeErr := os.UserHomeDir(); homeErr == nil {
			currentDir = homeDir
		} else {
			currentDir = "~"
		}
	}

	// Initialize context usage with default values
	// These will be updated in real-time as the conversation progresses
	ct.planningContext = &PlanningContext{
		Model:     model,
		Directory: ct.formatDirectory(currentDir),
		Usage: ContextUsage{
			Used:  0,
			Total: ct.getModelContextLimit(model),
		},
	}
	ct.lastUpdate = time.Now()
}

// StartPlanningSessionWithValidation initializes context tracking with validation
func (ct *ContextTracker) StartPlanningSessionWithValidation(model string) error {
	if model == "" {
		return fmt.Errorf("model name cannot be empty")
	}

	currentDir, err := os.Getwd()
	if err != nil {
		// This is not a critical error - we can continue with a fallback
		if homeDir, homeErr := os.UserHomeDir(); homeErr == nil {
			currentDir = homeDir
		} else {
			currentDir = "~"
		}
	}

	// Validate model context limits
	contextLimit := ct.getModelContextLimit(model)
	if contextLimit <= 0 {
		return fmt.Errorf("invalid context limit for model %s", model)
	}

	// Initialize context usage with validated values
	ct.planningContext = &PlanningContext{
		Model:     model,
		Directory: ct.formatDirectory(currentDir),
		Usage: ContextUsage{
			Used:  0,
			Total: contextLimit,
		},
	}
	ct.lastUpdate = time.Now()

	return nil
}

// StopPlanningSession ends context tracking for the planning session
func (ct *ContextTracker) StopPlanningSession() {
	ct.planningContext = nil
}

// IsActive returns whether planning context is currently being tracked
func (ct *ContextTracker) IsActive() bool {
	return ct.planningContext != nil
}

// GetPlanningContext returns the current planning context, or nil if inactive
func (ct *ContextTracker) GetPlanningContext() *PlanningContext {
	return ct.planningContext
}

// UpdateContextUsage updates the real-time context usage information
func (ct *ContextTracker) UpdateContextUsage(used int) {
	if ct.planningContext == nil {
		return
	}

	ct.planningContext.Usage.Used = used
	ct.lastUpdate = time.Now()
}

// UpdateModel changes the active model and adjusts context limits accordingly
func (ct *ContextTracker) UpdateModel(model string) {
	if ct.planningContext == nil {
		return
	}

	ct.planningContext.Model = model
	ct.planningContext.Usage.Total = ct.getModelContextLimit(model)
	ct.lastUpdate = time.Now()
}

// UpdateDirectory changes the current working directory display
func (ct *ContextTracker) UpdateDirectory(directory string) {
	if ct.planningContext == nil {
		return
	}

	ct.planningContext.Directory = ct.formatDirectory(directory)
	ct.lastUpdate = time.Now()
}

// ShouldUpdate returns true if the context information should be refreshed
func (ct *ContextTracker) ShouldUpdate() bool {
	return time.Since(ct.lastUpdate) >= ct.updateInterval
}

// FormatContextUsage formats the context usage for display in footer
func (ct *ContextTracker) FormatContextUsage() string {
	if ct.planningContext == nil {
		return ""
	}

	usage := ct.planningContext.Usage

	// Format with appropriate units (k for thousands)
	usedFormatted := ct.formatTokenCount(usage.Used)
	totalFormatted := ct.formatTokenCount(usage.Total)

	return fmt.Sprintf("ctx: %s/%s", usedFormatted, totalFormatted)
}

// formatDirectory formats the directory path for compact display
func (ct *ContextTracker) formatDirectory(dir string) string {
	// Get the base name of the directory for compact display
	baseName := filepath.Base(dir)

	// If it's the root directory, show the full path
	if baseName == "/" || baseName == "." || baseName == dir {
		return dir
	}

	return baseName
}

// formatTokenCount formats token counts with k suffix for thousands
func (ct *ContextTracker) formatTokenCount(count int) string {
	if count >= 1000 {
		return fmt.Sprintf("%dk", count/1000)
	}
	return fmt.Sprintf("%d", count)
}

// getModelContextLimit returns the context limit for a given model
func (ct *ContextTracker) getModelContextLimit(model string) int {
	if ct.contextLimitOverride > 0 {
		return ct.contextLimitOverride
	}
	// Default context limits for common models
	// These can be updated as new models are supported
	switch model {
	case "claude-sonnet-4", "claude-4-sonnet":
		return 200000
	case "claude-3.5-sonnet", "claude-sonnet-3.5":
		return 200000
	case "claude-3-sonnet", "claude-sonnet-3":
		return 200000
	case "claude-3-haiku", "claude-haiku-3":
		return 200000
	case "gpt-4", "gpt-4-turbo":
		return 128000
	case "gpt-4o", "gpt-4o-mini":
		return 128000
	case "gpt-3.5-turbo":
		return 16000
	default:
		// Default fallback
		return 128000
	}
}

// SetContextLimit sets an override for the model context limit from ACP session metadata
func (ct *ContextTracker) SetContextLimit(limit int) {
	ct.contextLimitOverride = limit
}

// GetUsage returns the current context usage (used, total)
func (ct *ContextTracker) GetUsage() (int, int) {
	if ct.planningContext == nil {
		return 0, 0
	}
	return ct.planningContext.Usage.Used, ct.planningContext.Usage.Total
}

// GetCurrentModel returns the currently active model, or empty string if inactive
func (ct *ContextTracker) GetCurrentModel() string {
	if ct.planningContext == nil {
		return ""
	}
	return ct.planningContext.Model
}
