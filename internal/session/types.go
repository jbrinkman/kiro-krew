package session

import (
	"encoding/json"
	"time"
)

// SessionType represents the type of session mode
type SessionType string

const (
	Console  SessionType = "console"
	Planning SessionType = "planning"
)

// Message represents a single conversation message
type Message struct {
	Role      string    `json:"role"`      // "user" or "assistant"
	Content   string    `json:"content"`   // message content
	Timestamp time.Time `json:"timestamp"` // when message was created
}

// SessionState holds the current session state and conversation history
type SessionState struct {
	Type    SessionType `json:"type"`    // current session type
	History []Message   `json:"history"` // conversation history
}

// NewSessionState creates a new session state
func NewSessionState(sessionType SessionType) *SessionState {
	return &SessionState{
		Type:    sessionType,
		History: make([]Message, 0),
	}
}

// AddMessage adds a message to the conversation history
func (s *SessionState) AddMessage(role, content string) {
	s.History = append(s.History, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
}

// ToJSON serializes the session state to JSON
func (s *SessionState) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

// FromJSON deserializes JSON data into a session state
func FromJSON(data []byte) (*SessionState, error) {
	var state SessionState
	err := json.Unmarshal(data, &state)
	if err != nil {
		return nil, err
	}
	return &state, nil
}
