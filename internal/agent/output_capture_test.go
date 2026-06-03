package agent

import (
	"testing"
)

func TestOutputCaptureSuspendResume(t *testing.T) {
	capture := NewOutputCapture(10)
	
	// Add some lines before suspension
	capture.AddLine("line1")
	capture.AddLine("line2")
	
	lines := capture.GetLines()
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines before suspension, got %d", len(lines))
	}
	
	// Suspend capture
	capture.Suspend()
	
	// Add lines while suspended (should be ignored)
	capture.AddLine("suspended1")
	capture.AddLine("suspended2")
	
	lines = capture.GetLines()
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines after suspension, got %d", len(lines))
	}
	
	// Resume capture
	capture.Resume()
	
	// Add lines after resumption
	capture.AddLine("line3")
	capture.AddLine("line4")
	
	lines = capture.GetLines()
	if len(lines) != 4 {
		t.Errorf("Expected 4 lines after resumption, got %d", len(lines))
	}
	
	expected := []string{"line1", "line2", "line3", "line4"}
	for i, line := range lines {
		if line != expected[i] {
			t.Errorf("Expected line %d to be %s, got %s", i, expected[i], line)
		}
	}
}