package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SessionManager handles CRUD operations and persistence for sessions
type SessionManager struct {
	sessionsDir string
}

// ValidationError represents a session validation failure
type ValidationError struct {
	SessionID string
	Field     string
	Message   string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("session %s: %s validation failed: %s", e.SessionID, e.Field, e.Message)
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessionsDir: ".kiro-krew/sessions",
	}
}

// NewSessionManagerWithDir creates a SessionManager using the specified directory
func NewSessionManagerWithDir(dir string) *SessionManager {
	return &SessionManager{
		sessionsDir: dir,
	}
}

// Create creates a new session and returns its ID
func (sm *SessionManager) Create(sessionType SessionType) (string, error) {
	// Generate session ID
	id, err := generateSessionID()
	if err != nil {
		return "", err
	}

	// Create session state
	state := NewSessionState(sessionType)

	// Save to disk
	err = sm.Save(id, state)
	if err != nil {
		return "", err
	}

	return id, nil
}

// Save persists a session to disk with validation
func (sm *SessionManager) Save(id string, state *SessionState) error {
	// Repair before validation to enforce limits
	sm.RepairSession(id, state)

	// Validate session before saving
	if err := sm.ValidateSession(id, state); err != nil {
		return fmt.Errorf("session validation failed: %w", err)
	}

	return sm.writeSession(id, state)
}

// SaveQuiet persists a session to disk without full validation (for high-frequency background saves)
func (sm *SessionManager) SaveQuiet(id string, state *SessionState) error {
	if state == nil {
		return fmt.Errorf("cannot save nil session state")
	}
	return sm.writeSession(id, state)
}

// writeSession handles the atomic file write
func (sm *SessionManager) writeSession(id string, state *SessionState) error {
	// Ensure sessions directory exists
	err := os.MkdirAll(sm.sessionsDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create sessions directory: %w", err)
	}

	// Serialize session
	data, err := state.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize session: %w", err)
	}

	// Write to file atomically (write to temp file first, then rename)
	filename := filepath.Join(sm.sessionsDir, id+".json")
	tempFile := filename + ".tmp"

	err = os.WriteFile(tempFile, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	err = os.Rename(tempFile, filename)
	if err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to finalize session file: %w", err)
	}

	return nil
}

// Load reads a session from disk with corruption recovery
func (sm *SessionManager) Load(id string) (*SessionState, error) {
	filename := filepath.Join(sm.sessionsDir, id+".json")

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	state, err := FromJSON(data)
	if err != nil {
		// Attempt corruption recovery
		if recovered := sm.recoverCorruptedSession(id, filename, data); recovered != nil {
			return recovered, nil
		}
		return nil, fmt.Errorf("failed to deserialize session (corruption detected): %w", err)
	}

	// Validate and repair session integrity
	if err := sm.ValidateSession(id, state); err != nil {
		return nil, fmt.Errorf("session validation failed: %w", err)
	}
	sm.RepairSession(id, state)

	return state, nil
}

// Delete removes a session from disk
func (sm *SessionManager) Delete(id string) error {
	filename := filepath.Join(sm.sessionsDir, id+".json")
	err := os.Remove(filename)
	if err != nil {
		return fmt.Errorf("failed to delete session file: %w", err)
	}
	return nil
}

// List returns all session IDs
func (sm *SessionManager) List() ([]string, error) {
	entries, err := os.ReadDir(sm.sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	var sessions []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			id := strings.TrimSuffix(entry.Name(), ".json")
			sessions = append(sessions, id)
		}
	}

	return sessions, nil
}

// Cleanup removes sessions older than the specified duration
func (sm *SessionManager) Cleanup(maxAge time.Duration) error {
	sessions, err := sm.List()
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-maxAge)
	var errors []string

	for _, id := range sessions {
		filename := filepath.Join(sm.sessionsDir, id+".json")
		info, err := os.Stat(filename)
		if err != nil {
			continue // Skip files we can't stat
		}

		if info.ModTime().Before(cutoff) {
			err = sm.Delete(id)
			if err != nil {
				errors = append(errors, fmt.Sprintf("session %s: %v", id, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %s", strings.Join(errors, "; "))
	}
	return nil
}

// CleanupOnExit performs graceful cleanup on system exit
func (sm *SessionManager) CleanupOnExit() error {
	// Clean up orphaned sessions (older than 24 hours)
	if err := sm.Cleanup(24 * time.Hour); err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	// Remove corrupt session files
	sessions, err := sm.List()
	if err != nil {
		return err
	}

	for _, id := range sessions {
		if _, err := sm.Load(id); err != nil {
			if strings.Contains(err.Error(), "corruption") {
				_ = sm.Delete(id) // Best effort cleanup
			}
		}
	}

	return nil
}

// recoverCorruptedSession attempts to recover a corrupted session
func (sm *SessionManager) recoverCorruptedSession(id, filename string, data []byte) *SessionState {
	// Try to extract type from partial JSON - look for the first valid type pattern
	dataStr := string(data)

	// Look for type field patterns
	var sessionType SessionType
	if strings.Contains(dataStr, `"type": "console"`) || strings.Contains(dataStr, `"type":"console"`) {
		sessionType = Console
	} else if strings.Contains(dataStr, `"type": "planning"`) || strings.Contains(dataStr, `"type":"planning"`) {
		sessionType = Planning
	} else {
		// Try JSON unmarshaling for partial extraction as fallback
		var partial struct {
			Type SessionType `json:"type"`
		}
		if json.Unmarshal(data, &partial) == nil && partial.Type != "" {
			sessionType = partial.Type
		}
	}

	if sessionType != "" {
		var recovered *SessionState

		if sessionType == Planning {
			// Try to extract planning data for better recovery
			var planningData struct {
				PlanningData *PlanningSessionData `json:"planning_data"`
			}

			tabID := "recovered-" + id
			title := "Recovered Planning Tab"

			// Attempt to extract planning metadata if possible
			if json.Unmarshal(data, &planningData) == nil && planningData.PlanningData != nil {
				if planningData.PlanningData.TabID != "" {
					tabID = planningData.PlanningData.TabID
				}
				if planningData.PlanningData.Title != "" {
					title = planningData.PlanningData.Title
				}
			}

			recovered = NewPlanningSessionState(tabID, title)
		} else {
			recovered = NewSessionState(sessionType)
		}

		// Try to recover conversation history
		var historyData struct {
			History []Message `json:"history"`
		}
		if json.Unmarshal(data, &historyData) == nil && len(historyData.History) > 0 {
			recovered.History = historyData.History
		}

		// Create backup of corrupted file
		backupName := filename + ".corrupt." + time.Now().Format("20060102-150405")
		_ = os.Rename(filename, backupName)

		// Save recovered session
		if err := sm.writeSession(id, recovered); err != nil {
			return nil
		}

		return recovered
	}

	return nil
}

// ValidateSession performs read-only session integrity checks
func (sm *SessionManager) ValidateSession(id string, state *SessionState) error {
	if state == nil {
		return &ValidationError{SessionID: id, Field: "state", Message: "session state is nil"}
	}

	if state.Type != Console && state.Type != Planning {
		return &ValidationError{SessionID: id, Field: "type", Message: fmt.Sprintf("invalid session type: %s", state.Type)}
	}

	// Validate message integrity
	for i, msg := range state.History {
		if msg.Role != "user" && msg.Role != "assistant" && msg.Role != "system" {
			return &ValidationError{SessionID: id, Field: "history", Message: fmt.Sprintf("message %d has invalid role: %s", i, msg.Role)}
		}
	}

	// Validate planning-specific data
	if state.Type == Planning {
		// Planning data is only required for sessions created with NewPlanningSessionState
		// Sessions created with NewSessionState(Planning) don't have planning data by design
		if state.PlanningData != nil {
			if state.PlanningData.TabID == "" {
				return &ValidationError{SessionID: id, Field: "planning_data.tab_id", Message: "planning session missing tab ID"}
			}

			if state.PlanningData.Title == "" {
				return &ValidationError{SessionID: id, Field: "planning_data.title", Message: "planning session missing title"}
			}

			// Validate planning state enum
			planningState := state.PlanningData.State
			if planningState < PlanningStateIdle || planningState > PlanningStateReadOnly {
				return &ValidationError{SessionID: id, Field: "planning_data.state", Message: fmt.Sprintf("invalid planning state: %d", planningState)}
			}

			// Note: ACP connection fields (Agent, Model), context usage (Total, Used),
			// and timestamp fields (CreatedAt, LastActivity) are not validated here
			// as they can be repaired by RepairSession with appropriate defaults
		}
	}

	return nil
}

// RepairSession fixes recoverable issues in session state (nil history, zero timestamps, excessive history)
func (sm *SessionManager) RepairSession(id string, state *SessionState) {
	if state == nil {
		return
	}

	if state.History == nil {
		state.History = make([]Message, 0)
	}

	// Fix zero timestamps in message history
	for i := range state.History {
		if state.History[i].Timestamp.IsZero() {
			state.History[i].Timestamp = time.Now()
		}
	}

	// Trim excessive history (memory protection)
	if len(state.History) > 10000 {
		state.History = state.History[len(state.History)-5000:]
	}

	// Repair planning-specific data
	if state.Type == Planning {
		if state.PlanningData == nil {
			// Reconstruct planning data with defaults if missing
			now := time.Now()
			state.PlanningData = &PlanningSessionData{
				TabID:         "recovered-" + id,
				Title:         "Recovered Planning Tab",
				State:         PlanningStateIdle,
				CreatedAt:     now,
				LastActivity:  now,
				ACPConnection: DefaultACPConnectionMetadata(),
				ContextUsage:  DefaultContextUsage(),
			}
		} else {
			// Fix individual fields in existing planning data
			if state.PlanningData.TabID == "" {
				state.PlanningData.TabID = "recovered-" + id
			}

			if state.PlanningData.Title == "" {
				state.PlanningData.Title = "Recovered Planning Tab"
			}

			now := time.Now()
			if state.PlanningData.CreatedAt.IsZero() {
				state.PlanningData.CreatedAt = now
			}

			if state.PlanningData.LastActivity.IsZero() {
				state.PlanningData.LastActivity = now
			}

			// Fix ACP connection defaults
			if state.PlanningData.ACPConnection.Agent == "" {
				state.PlanningData.ACPConnection.Agent = "kiro-agent"
			}

			if state.PlanningData.ACPConnection.Model == "" {
				state.PlanningData.ACPConnection.Model = "claude-sonnet-4"
			}

			if state.PlanningData.ACPConnection.Timeout == 0 {
				state.PlanningData.ACPConnection.Timeout = 60 * time.Second
			}

			if state.PlanningData.ACPConnection.ResponseFormat == "" {
				state.PlanningData.ACPConnection.ResponseFormat = "text"
			}

			if state.PlanningData.ACPConnection.LastActivity.IsZero() {
				state.PlanningData.ACPConnection.LastActivity = now
			}

			// Fix context usage defaults
			if state.PlanningData.ContextUsage.Total <= 0 {
				state.PlanningData.ContextUsage.Total = 200000
			}

			if state.PlanningData.ContextUsage.Used < 0 {
				state.PlanningData.ContextUsage.Used = 0
			}
		}
	}
}

// generateSessionID creates a random session identifier
func generateSessionID() (string, error) {
	bytes := make([]byte, 8)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// LogModeSwitch is a no-op hook for future diagnostics during mode transitions
func (sm *SessionManager) LogModeSwitch(sessionID string, fromMode, toMode SessionType, preservedData interface{}) {
	// Reserved for future diagnostic use
}

// ValidateSessionFlow ensures data integrity during rapid mode switching
func (sm *SessionManager) ValidateSessionFlow(sessionID string) error {
	state, err := sm.Load(sessionID)
	if err != nil {
		return fmt.Errorf("session flow validation failed: %w", err)
	}

	if err := sm.ValidateSession(sessionID, state); err != nil {
		return fmt.Errorf("session flow validation failed: %w", err)
	}

	return nil
}

// Planning Tab Session Management

// CreatePlanningSession creates a new planning session with ACP metadata
func (sm *SessionManager) CreatePlanningSession(tabID, title string) (string, *SessionState, error) {
	// Generate session ID
	sessionID, err := generateSessionID()
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	// Create planning session state
	state := NewPlanningSessionState(tabID, title)

	// Save to disk
	if err := sm.Save(sessionID, state); err != nil {
		return "", nil, fmt.Errorf("failed to save planning session: %w", err)
	}

	return sessionID, state, nil
}

// LoadPlanningSession loads a planning session by session ID
func (sm *SessionManager) LoadPlanningSession(sessionID string) (*SessionState, error) {
	state, err := sm.Load(sessionID)
	if err != nil {
		return nil, err
	}

	if !state.IsPlanning() {
		return nil, fmt.Errorf("session %s is not a planning session", sessionID)
	}

	return state, nil
}

// FindPlanningSessionByTabID finds a planning session by tab ID
func (sm *SessionManager) FindPlanningSessionByTabID(tabID string) (string, *SessionState, error) {
	sessions, err := sm.List()
	if err != nil {
		return "", nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	for _, sessionID := range sessions {
		state, err := sm.Load(sessionID)
		if err != nil {
			continue // Skip corrupted sessions
		}

		if state.IsPlanning() && state.GetTabID() == tabID {
			return sessionID, state, nil
		}
	}

	return "", nil, fmt.Errorf("no planning session found for tab ID: %s", tabID)
}

// ListPlanningSessions returns all planning session IDs and their states
func (sm *SessionManager) ListPlanningSessions() (map[string]*SessionState, error) {
	sessions, err := sm.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	planningSessions := make(map[string]*SessionState)
	for _, sessionID := range sessions {
		state, err := sm.Load(sessionID)
		if err != nil {
			continue // Skip corrupted sessions
		}

		if state.IsPlanning() {
			planningSessions[sessionID] = state
		}
	}

	return planningSessions, nil
}

// UpdatePlanningSessionState updates the state of a planning session
func (sm *SessionManager) UpdatePlanningSessionState(sessionID string, state PlanningTabState) error {
	sessionState, err := sm.LoadPlanningSession(sessionID)
	if err != nil {
		return err
	}

	sessionState.UpdatePlanningState(state)
	return sm.SaveQuiet(sessionID, sessionState)
}

// UpdatePlanningSessionACP updates ACP connection metadata for a planning session
func (sm *SessionManager) UpdatePlanningSessionACP(sessionID string, connected bool, agent, model string) error {
	sessionState, err := sm.LoadPlanningSession(sessionID)
	if err != nil {
		return err
	}

	sessionState.UpdateACPConnection(connected, agent, model)
	return sm.SaveQuiet(sessionID, sessionState)
}

// UpdatePlanningSessionContext updates context usage for a planning session
func (sm *SessionManager) UpdatePlanningSessionContext(sessionID string, used, total int) error {
	sessionState, err := sm.LoadPlanningSession(sessionID)
	if err != nil {
		return err
	}

	sessionState.UpdateContextUsage(used, total)
	return sm.SaveQuiet(sessionID, sessionState)
}

// CleanupCompletedPlanningSessions removes completed/failed planning sessions older than maxAge
func (sm *SessionManager) CleanupCompletedPlanningSessions(maxAge time.Duration) (int, error) {
	sessions, err := sm.ListPlanningSessions()
	if err != nil {
		return 0, fmt.Errorf("failed to list planning sessions: %w", err)
	}

	cutoff := time.Now().Add(-maxAge)
	cleaned := 0

	for sessionID, state := range sessions {
		planningState := state.GetPlanningState()

		// Only cleanup completed or failed sessions
		if planningState == PlanningStateCompleted || planningState == PlanningStateFailed {
			if state.PlanningData.LastActivity.Before(cutoff) {
				if err := sm.Delete(sessionID); err == nil {
					cleaned++
				}
			}
		}
	}

	return cleaned, nil
}

// CleanupOrphanedPlanningSessions removes planning sessions for tabs that no longer exist
func (sm *SessionManager) CleanupOrphanedPlanningSessions(activeTabIDs []string) (int, error) {
	sessions, err := sm.ListPlanningSessions()
	if err != nil {
		return 0, fmt.Errorf("failed to list planning sessions: %w", err)
	}

	// Create lookup map of active tab IDs
	activeMap := make(map[string]bool)
	for _, tabID := range activeTabIDs {
		activeMap[tabID] = true
	}

	cleaned := 0
	for sessionID, state := range sessions {
		tabID := state.GetTabID()
		if !activeMap[tabID] {
			// This session belongs to a tab that no longer exists
			if err := sm.Delete(sessionID); err == nil {
				cleaned++
			}
		}
	}

	return cleaned, nil
}
