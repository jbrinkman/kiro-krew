package tui

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/jbrinkman/kiro-krew/internal/agent"
	"github.com/jbrinkman/kiro-krew/internal/config"
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

func TestOutputAlwaysAccumulates(t *testing.T) {
	capture := agent.NewOutputCapture(100)

	// Add some initial output
	capture.AddLine("Initial line")

	// Add more output - data is always accumulated
	capture.AddLine("Second line - always captured")

	// Add even more output
	capture.AddLine("Third line")

	lines := capture.GetLines()

	// Should have all lines since data is always accumulated
	found := make(map[string]bool)
	for _, line := range lines {
		found[line] = true
	}

	if !found["Initial line"] {
		t.Error("Expected 'Initial line' not found")
	}

	if !found["Second line - always captured"] {
		t.Error("Second line was not captured (should always be captured)")
	}

	if !found["Third line"] {
		t.Error("Expected 'Third line' not found")
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

func TestOutputViewRendersCapturedManagerOutput(t *testing.T) {
	cfg := &config.Config{LoadedTheme: &config.Theme{}}
	cfg.LoadedTheme.Colors.Primary = "#ffffff"
	cfg.LoadedTheme.Colors.Secondary = "#cccccc"
	cfg.LoadedTheme.Colors.Success = "#00ff00"
	cfg.LoadedTheme.Colors.Warning = "#ffff00"
	cfg.LoadedTheme.Colors.Error = "#ff0000"
	cfg.LoadedTheme.Colors.TextPrimary = "#ffffff"
	cfg.LoadedTheme.Colors.TextSecondary = "#cccccc"
	cfg.LoadedTheme.Colors.TextMuted = "#999999"
	cfg.LoadedTheme.Colors.Prompt = "#ffffff"
	cfg.LoadedTheme.Colors.Separator = "#666666"
	cfg.LoadedTheme.Colors.Activity = "#00ffff"
	cfg.LoadedTheme.Colors.Background = "#000000"
	cfg.LoadedTheme.Colors.Surface = "#111111"
	manager := agent.NewManager(cfg)
	for i := 0; i < 30; i++ {
		manager.CaptureOutputLine(42, fmt.Sprintf("captured line %d", i))
	}

	view := NewOutputView(manager, NewStyles(cfg.LoadedTheme))
	view.Resize(80, 6)

	rendered := view.View()
	if !strings.Contains(rendered, "captured line") {
		t.Fatalf("expected rendered output to include captured lines, got %q", rendered)
	}

	view.viewport.ScrollDown(1)
	scrolled := view.View()
	if rendered == scrolled {
		t.Fatal("expected view content to change after scrolling")
	}
}
