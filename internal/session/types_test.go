package session

import (
	"testing"
)

func TestSessionStateJSONMarshaling(t *testing.T) {
	// Create a new session state
	state := NewSessionState(Planning)
	state.AddMessage("user", "Hello")
	state.AddMessage("assistant", "Hi there!")

	// Serialize to JSON
	data, err := state.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	// Deserialize from JSON
	restored, err := FromJSON(data)
	if err != nil {
		t.Fatalf("Failed to deserialize: %v", err)
	}

	// Verify the data
	if restored.Type != Planning {
		t.Errorf("Expected type %v, got %v", Planning, restored.Type)
	}
	if len(restored.History) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(restored.History))
	}
	if restored.History[0].Role != "user" || restored.History[0].Content != "Hello" {
		t.Errorf("First message not restored correctly")
	}
}

func TestSessionTypes(t *testing.T) {
	if Console != "console" {
		t.Errorf("Console constant incorrect")
	}
	if Planning != "planning" {
		t.Errorf("Planning constant incorrect")
	}
}