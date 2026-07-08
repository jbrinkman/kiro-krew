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

// PlanningTabState represents the different states a planning tab can be in
type PlanningTabState int

const (
	PlanningStateIdle PlanningTabState = iota
	PlanningStateActive
	PlanningStateCompleted
	PlanningStateFailed
	PlanningStateReadOnly
)

// Message represents a single conversation message
type Message struct {
	Role      string    `json:"role"`      // "user" or "assistant"
	Content   string    `json:"content"`   // message content
	Timestamp time.Time `json:"timestamp"` // when message was created
}

// ACPConnectionMetadata holds ACP connection details
type ACPConnectionMetadata struct {
	Agent          string        `json:"agent"`           // Agent name (e.g., "kiro-agent")
	Model          string        `json:"model"`           // Model being used
	Connected      bool          `json:"connected"`       // Current connection status
	LastActivity   time.Time     `json:"last_activity"`   // Last message timestamp
	Timeout        time.Duration `json:"timeout"`         // Connection timeout
	ResponseFormat string        `json:"response_format"` // "text", "json", etc.
	Streaming      bool          `json:"streaming"`       // Whether streaming is enabled
}

// ContextUsage represents context usage tracking
type ContextUsage struct {
	Used  int `json:"used"`  // Current context usage in tokens
	Total int `json:"total"` // Total context limit in tokens
}

// PlanningSessionData holds Planning Tab-specific session state
type PlanningSessionData struct {
	TabID         string                `json:"tab_id"`         // Unique planning tab identifier
	Title         string                `json:"title"`          // Planning tab title
	State         PlanningTabState      `json:"state"`          // Current tab state
	ACPConnection ACPConnectionMetadata `json:"acp_connection"` // ACP connection details
	ContextUsage  ContextUsage          `json:"context_usage"`  // Context usage tracking
	CreatedAt     time.Time             `json:"created_at"`     // Session creation timestamp
	LastActivity  time.Time             `json:"last_activity"`  // Last user/assistant interaction
}

// SessionState holds the current session state and conversation history
type SessionState struct {
	Type    SessionType `json:"type"`    // current session type
	History []Message   `json:"history"` // conversation history

	// Planning-specific data (only populated for Planning session type)
	PlanningData *PlanningSessionData `json:"planning_data,omitempty"`
}

// NewSessionState creates a new session state
func NewSessionState(sessionType SessionType) *SessionState {
	return &SessionState{
		Type:    sessionType,
		History: make([]Message, 0),
	}
}

// NewPlanningSessionState creates a new planning session state with ACP data
func NewPlanningSessionState(tabID, title string) *SessionState {
	now := time.Now()
	return &SessionState{
		Type:    Planning,
		History: make([]Message, 0),
		PlanningData: &PlanningSessionData{
			TabID:        tabID,
			Title:        title,
			State:        PlanningStateIdle,
			CreatedAt:    now,
			LastActivity: now,
			ACPConnection: ACPConnectionMetadata{
				Agent:          "kiro-agent",
				Model:          "claude-sonnet-4",
				Connected:      false,
				Timeout:        60 * time.Second,
				ResponseFormat: "text",
				Streaming:      true,
			},
			ContextUsage: ContextUsage{
				Used:  0,
				Total: 200000, // Default Claude context limit
			},
		},
	}
}

// AddMessage adds a message to the conversation history
func (s *SessionState) AddMessage(role, content string) {
	s.History = append(s.History, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})

	// Update last activity for planning sessions
	if s.PlanningData != nil {
		s.PlanningData.LastActivity = time.Now()
	}
}

// UpdatePlanningState updates the state of a planning session
func (s *SessionState) UpdatePlanningState(state PlanningTabState) {
	if s.PlanningData != nil {
		s.PlanningData.State = state
		s.PlanningData.LastActivity = time.Now()
	}
}

// UpdateACPConnection updates ACP connection metadata
func (s *SessionState) UpdateACPConnection(connected bool, agent, model string) {
	if s.PlanningData != nil {
		s.PlanningData.ACPConnection.Connected = connected
		if agent != "" {
			s.PlanningData.ACPConnection.Agent = agent
		}
		if model != "" {
			s.PlanningData.ACPConnection.Model = model
		}
		s.PlanningData.ACPConnection.LastActivity = time.Now()
		s.PlanningData.LastActivity = time.Now()
	}
}

// UpdateContextUsage updates context usage tracking
func (s *SessionState) UpdateContextUsage(used, total int) {
	if s.PlanningData != nil {
		s.PlanningData.ContextUsage.Used = used
		if total > 0 {
			s.PlanningData.ContextUsage.Total = total
		}
		s.PlanningData.LastActivity = time.Now()
	}
}

// IsPlanning returns true if this is a planning session
func (s *SessionState) IsPlanning() bool {
	// A session is considered a planning session if its type is Planning
	// It may or may not have PlanningData depending on how it was created
	return s.Type == Planning
}

// GetTabID returns the planning tab ID, or empty string if not a planning session or no planning data
func (s *SessionState) GetTabID() string {
	if s.PlanningData != nil {
		return s.PlanningData.TabID
	}
	return ""
}

// GetPlanningState returns the current planning state, or PlanningStateIdle if no planning data
func (s *SessionState) GetPlanningState() PlanningTabState {
	if s.PlanningData != nil {
		return s.PlanningData.State
	}
	return PlanningStateIdle
}

// CanRestore returns true if this planning session can be restored (not in active state)
func (s *SessionState) CanRestore() bool {
	if !s.IsPlanning() {
		return false
	}
	// Sessions without planning data cannot be restored as planning tabs
	if s.PlanningData == nil {
		return false
	}
	state := s.PlanningData.State
	return state != PlanningStateActive && state != PlanningStateReadOnly
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
