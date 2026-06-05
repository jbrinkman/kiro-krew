package agent

import (
	"bytes"
	"io"
	"regexp"
	"sync"
)

// OutputCapture captures and stores agent output with ANSI stripping
type OutputCapture struct {
	mu        sync.RWMutex
	buffer    []string
	maxSize   int
	ansiRegex *regexp.Regexp
	suspended bool
}

// NewOutputCapture creates a new output capture with specified buffer size
func NewOutputCapture(maxSize int) *OutputCapture {
	if maxSize <= 0 {
		maxSize = 1
	}
	return &OutputCapture{
		buffer:    make([]string, 0, maxSize),
		maxSize:   maxSize,
		ansiRegex: regexp.MustCompile(`\x1b\[[0-9;]*[mK]`),
	}
}

// AddLine adds a line to the buffer, stripping ANSI sequences
func (oc *OutputCapture) AddLine(line string) {
	oc.mu.Lock()
	defer oc.mu.Unlock()

	// Skip capturing if suspended
	if oc.suspended {
		return
	}

	stripped := oc.ansiRegex.ReplaceAllString(line, "")

	if len(oc.buffer) >= oc.maxSize {
		copy(oc.buffer, oc.buffer[1:])
		oc.buffer[len(oc.buffer)-1] = stripped
	} else {
		oc.buffer = append(oc.buffer, stripped)
	}
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

// GetLines returns a copy of all captured lines
func (oc *OutputCapture) GetLines() []string {
	oc.mu.RLock()
	defer oc.mu.RUnlock()

	result := make([]string, len(oc.buffer))
	copy(result, oc.buffer)
	return result
}

// CaptureWriter wraps prefixedWriter to capture output
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

// Write implements io.Writer, capturing lines while delegating to prefixedWriter
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
