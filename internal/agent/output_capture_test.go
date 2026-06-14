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

func TestOutputCaptureRingBuffer(t *testing.T) {
	capture := NewOutputCapture(3)

	capture.AddLine("a")
	capture.AddLine("b")
	capture.AddLine("c")

	lines := capture.GetLines()
	if len(lines) != 3 {
		t.Fatalf("Expected 3 lines, got %d", len(lines))
	}
	if lines[0] != "a" || lines[1] != "b" || lines[2] != "c" {
		t.Fatalf("Expected [a b c], got %v", lines)
	}

	// Overflow — oldest line (a) should be evicted
	capture.AddLine("d")
	lines = capture.GetLines()
	if len(lines) != 3 {
		t.Fatalf("Expected 3 lines after overflow, got %d", len(lines))
	}
	if lines[0] != "b" || lines[1] != "c" || lines[2] != "d" {
		t.Fatalf("Expected [b c d], got %v", lines)
	}

	// Another overflow
	capture.AddLine("e")
	capture.AddLine("f")
	lines = capture.GetLines()
	if lines[0] != "d" || lines[1] != "e" || lines[2] != "f" {
		t.Fatalf("Expected [d e f], got %v", lines)
	}
}

func TestOutputCaptureGeneration(t *testing.T) {
	capture := NewOutputCapture(10)

	gen0 := capture.Generation()
	if gen0 != 0 {
		t.Fatalf("Expected initial generation 0, got %d", gen0)
	}

	capture.AddLine("x")
	gen1 := capture.Generation()
	if gen1 != 1 {
		t.Fatalf("Expected generation 1 after AddLine, got %d", gen1)
	}

	// Suspended lines don't increment generation
	capture.Suspend()
	capture.AddLine("ignored")
	gen2 := capture.Generation()
	if gen2 != 1 {
		t.Fatalf("Expected generation still 1 after suspended AddLine, got %d", gen2)
	}

	capture.Resume()
	capture.AddLine("y")
	gen3 := capture.Generation()
	if gen3 != 2 {
		t.Fatalf("Expected generation 2, got %d", gen3)
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
