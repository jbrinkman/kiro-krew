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
		// Create new session with recovered type
		recovered := NewSessionState(sessionType)

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

	// Fix zero timestamps
	for i := range state.History {
		if state.History[i].Timestamp.IsZero() {
			state.History[i].Timestamp = time.Now()
		}
	}

	// Trim excessive history (memory protection)
	if len(state.History) > 10000 {
		state.History = state.History[len(state.History)-5000:]
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
