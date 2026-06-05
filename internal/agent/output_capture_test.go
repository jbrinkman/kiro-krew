package agent

import (
	"strings"
	"sync"
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

func TestCaptureWriterConcurrentWrites(t *testing.T) {
	capture := NewOutputCapture(100)
	writer := NewCaptureWriter(nil, capture, "[agent issue-1] ")

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = writer.Write([]byte(strings.Repeat("x", i%3+1) + "\n"))
		}(i)
	}
	wg.Wait()

	lines := capture.GetLines()
	if len(lines) != 20 {
		t.Fatalf("expected 20 captured lines, got %d", len(lines))
	}
	for _, line := range lines {
		if !strings.HasPrefix(line, "[agent issue-1] ") {
			t.Fatalf("expected captured prefix, got %q", line)
		}
	}
}
