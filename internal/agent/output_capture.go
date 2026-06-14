package agent

import (
	"bytes"
	"io"
	"regexp"
	"sync"
	"sync/atomic"
)

// OutputCapture captures and stores agent output with ANSI stripping.
// Uses an index-based ring buffer to avoid O(n) slice shifting on every line.
type OutputCapture struct {
	mu        sync.RWMutex
	buffer    []string
	maxSize   int
	head      int // next write position
	count     int // number of valid entries (≤ maxSize)
	gen       atomic.Uint64
	ansiRegex *regexp.Regexp
	suspended bool
}

// NewOutputCapture creates a new output capture with specified buffer size
func NewOutputCapture(maxSize int) *OutputCapture {
	if maxSize <= 0 {
		maxSize = 1
	}
	return &OutputCapture{
		buffer:    make([]string, maxSize),
		maxSize:   maxSize,
		ansiRegex: regexp.MustCompile(`\x1b\[[0-9;]*[mK]`),
	}
}

// AddLine adds a line to the ring buffer, stripping ANSI sequences
func (oc *OutputCapture) AddLine(line string) {
	oc.mu.Lock()
	defer oc.mu.Unlock()

	if oc.suspended {
		return
	}

	stripped := oc.ansiRegex.ReplaceAllString(line, "")
	oc.buffer[oc.head] = stripped
	oc.head = (oc.head + 1) % oc.maxSize
	if oc.count < oc.maxSize {
		oc.count++
	}
	oc.gen.Add(1)
}

// Suspend suspends output capture
func (oc *OutputCapture) Suspend() {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	oc.suspended = true
}

// Resume resumes output capture
func (oc *OutputCapture) Resume() {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	oc.suspended = false
}

// Generation returns the current generation counter (lock-free).
// Callers can compare against a previous value to detect new data.
func (oc *OutputCapture) Generation() uint64 {
	return oc.gen.Load()
}

// GetLines returns a copy of all captured lines in chronological order
func (oc *OutputCapture) GetLines() []string {
	oc.mu.RLock()
	defer oc.mu.RUnlock()

	if oc.count == 0 {
		return nil
	}

	result := make([]string, oc.count)
	if oc.count < oc.maxSize {
		// Buffer not yet full — entries are 0..count-1
		copy(result, oc.buffer[:oc.count])
	} else {
		// Buffer full — oldest entry is at head, wrap around
		n := copy(result, oc.buffer[oc.head:oc.maxSize])
		copy(result[n:], oc.buffer[:oc.head])
	}
	return result
}

// CaptureWriter wraps an optional writer to capture output lines
type CaptureWriter struct {
	mu      sync.Mutex
	writer  io.Writer
	capture *OutputCapture
	lineBuf []byte
	prefix  string
}

// NewCaptureWriter creates a writer that captures output while preserving existing behavior
func NewCaptureWriter(writer io.Writer, capture *OutputCapture, prefix string) *CaptureWriter {
	return &CaptureWriter{
		writer:  writer,
		capture: capture,
		lineBuf: make([]byte, 0, 256),
		prefix:  prefix,
	}
}

// Write implements io.Writer, capturing lines while delegating to the wrapped writer
func (cw *CaptureWriter) Write(p []byte) (int, error) {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if cw.writer != nil {
		n, err := cw.writer.Write(p)
		if err != nil {
			return n, err
		}
	}

	for _, chunk := range bytes.SplitAfter(p, []byte("\n")) {
		if len(chunk) == 0 {
			continue
		}

		cw.lineBuf = append(cw.lineBuf, chunk...)
		if cw.lineBuf[len(cw.lineBuf)-1] == '\n' {
			line := bytes.TrimSuffix(cw.lineBuf, []byte("\n"))
			cw.capture.AddLine(cw.prefix + string(line))
			cw.lineBuf = cw.lineBuf[:0]
		}
	}

	return len(p), nil
}
