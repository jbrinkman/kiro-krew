package tui

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/jbrinkman/kiro-krew/internal/agent"
)

func TestMultipleAgentsOutputCapture(t *testing.T) {
	capture := agent.NewOutputCapture(1000)
	
	// Simulate multiple agents producing output
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			// Simulate agent output
			for j := 0; j < 10; j++ {
				output := fmt.Sprintf("Agent %d output line %d", id, j)
				capture.AddLine(output)
				time.Sleep(5 * time.Millisecond)
			}
		}(i)
	}
	
	wg.Wait()

	// Verify output was captured
	lines := capture.GetLines()
	if len(lines) == 0 {
		t.Error("No output captured from multiple agents")
	}

	// Check that output from different agents is present
	found := make(map[int]bool)
	for _, line := range lines {
		for i := 0; i < 5; i++ {
			if strings.Contains(line, fmt.Sprintf("Agent %d", i)) {
				found[i] = true
			}
		}
	}
	
	if len(found) < 3 { // At least 3 agents should have output
		t.Errorf("Expected output from at least 3 agents, found %d", len(found))
	}
}

func TestANSIStripping(t *testing.T) {
	capture := agent.NewOutputCapture(100)

	testCases := []struct {
		input    string
		expected string
	}{
		{"\033[31mRed text\033[0m", "Red text"},
		{"\033[1;32mBold Green\033[0m", "Bold Green"},
		{"\033[2K\rClearing line", "\rClearing line"}, // \r is not stripped by ANSI regex
		{"Normal text", "Normal text"},
		{"\033[38;5;196mExtended color\033[0m", "Extended color"},
	}

	for _, tc := range testCases {
		capture.AddLine(tc.input)
	}
	
	lines := capture.GetLines()
	for i, tc := range testCases {
		if i < len(lines) {
			if lines[i] != tc.expected {
				t.Errorf("ANSI stripping failed: expected %q, got %q", tc.expected, lines[i])
			}
		}
	}
}

func TestViewManagerTransitions(t *testing.T) {
	vm := NewViewManager()
	
	// Test initial state
	if vm.CurrentView() != ViewConsole {
		t.Error("Expected initial view to be console")
	}
	
	// Test toggle to agent output
	vm.ToggleView()
	if vm.CurrentView() != ViewAgentOutput {
		t.Error("Failed to toggle to agent output view")
	}
	
	// Test toggle back to console
	vm.ToggleView()
	if vm.CurrentView() != ViewConsole {
		t.Error("Failed to toggle back to console view")
	}
	
	// Test explicit set
	vm.SetView(ViewAgentOutput)
	if vm.CurrentView() != ViewAgentOutput {
		t.Error("Failed to explicitly set agent output view")
	}
}

func TestHighVolumeOutput(t *testing.T) {
	capture := agent.NewOutputCapture(500) // Smaller buffer for test
	
	// Generate high volume output
	for i := 0; i < 1000; i++ {
		output := fmt.Sprintf("Line %d: %s", i, strings.Repeat("x", 50))
		capture.AddLine(output)
	}
	
	// Verify buffer size is maintained
	lines := capture.GetLines()
	if len(lines) > 500 {
		t.Errorf("Buffer exceeded max size: %d > 500", len(lines))
	}
	
	// Verify we have the most recent lines (buffer rotates)
	if len(lines) == 500 {
		lastLine := lines[len(lines)-1]
		if !strings.Contains(lastLine, "Line 999") {
			t.Error("Buffer rotation not working correctly - missing recent data")
		}
	}
}

func TestTerminalResizeHandling(t *testing.T) {
	vm := NewViewManager()
	
	// Test resize message
	resizeMsg := tea.WindowSizeMsg{Width: 120, Height: 30}
	_ = vm.Update(resizeMsg)
	
	// Verify dimensions are stored (indirectly through no panic/error)
	// The actual width/height are private, so we test the behavior works
	if vm.CurrentView() != ViewConsole {
		t.Error("View manager state corrupted after resize")
	}
}

func TestOutputSuspendResume(t *testing.T) {
	capture := agent.NewOutputCapture(100)
	
	// Add some initial output
	capture.AddLine("Before suspend")
	
	// Suspend and try to add more
	capture.Suspend()
	capture.AddLine("During suspend - should not appear")
	
	// Resume and add more
	capture.Resume()
	capture.AddLine("After resume")
	
	lines := capture.GetLines()
	
	// Should have first and last line, but not the middle one
	found := make(map[string]bool)
	for _, line := range lines {
		found[line] = true
	}
	
	if !found["Before suspend"] {
		t.Error("Expected 'Before suspend' line not found")
	}
	
	if found["During suspend - should not appear"] {
		t.Error("Line added during suspend was captured")
	}
	
	if !found["After resume"] {
		t.Error("Expected 'After resume' line not found")
	}
}

func BenchmarkConcurrentOutput(b *testing.B) {
	capture := agent.NewOutputCapture(10000)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		lineNum := 0
		for pb.Next() {
			output := fmt.Sprintf("Benchmark output %d", lineNum)
			capture.AddLine(output)
			lineNum++
		}
	})
}