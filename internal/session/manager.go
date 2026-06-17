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

// Save persists a session to disk
func (sm *SessionManager) Save(id string, state *SessionState) error {
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

	// Write to file
	filename := filepath.Join(sm.sessionsDir, id+".json")
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
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
		if recovered := sm.recoverCorruptedSession(filename, data); recovered != nil {
			return recovered, nil
		}
		return nil, fmt.Errorf("failed to deserialize session (corruption detected): %w", err)
	}

	// Validate session integrity
	if err := sm.validateSession(state); err != nil {
		return nil, fmt.Errorf("session validation failed: %w", err)
	}

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
func (sm *SessionManager) recoverCorruptedSession(filename string, data []byte) *SessionState {
	// Try to extract type from partial JSON
	var partial struct {
		Type SessionType `json:"type"`
	}

	if json.Unmarshal(data, &partial) == nil && partial.Type != "" {
		// Create new session with recovered type
		recovered := NewSessionState(partial.Type)

		// Create backup of corrupted file
		backupName := filename + ".corrupt." + time.Now().Format("20060102-150405")
		_ = os.Rename(filename, backupName)

		return recovered
	}

	return nil
}

// validateSession checks session integrity
func (sm *SessionManager) validateSession(state *SessionState) error {
	if state == nil {
		return fmt.Errorf("session state is nil")
	}

	if state.Type != Console && state.Type != Planning {
		return fmt.Errorf("invalid session type: %s", state.Type)
	}

	if state.History == nil {
		state.History = make([]Message, 0)
	}

	return nil
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
