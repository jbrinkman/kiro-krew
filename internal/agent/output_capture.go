package agent

import (
	"regexp"
	"sync"
)

// OutputCapture captures and stores agent output with ANSI stripping
type OutputCapture struct {
	mu     sync.RWMutex
	buffer []string
	maxSize int
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
	*prefixedWriter
	capture *OutputCapture
	lineBuf []byte
}

// NewCaptureWriter creates a writer that captures output while preserving existing behavior
func NewCaptureWriter(pw *prefixedWriter, capture *OutputCapture) *CaptureWriter {
	return &CaptureWriter{
		prefixedWriter: pw,
		capture:        capture,
		lineBuf:        make([]byte, 0, 256),
	}
}

// Write implements io.Writer, capturing lines while delegating to prefixedWriter
func (cw *CaptureWriter) Write(p []byte) (int, error) {
	// First write to the original prefixedWriter
	n, err := cw.prefixedWriter.Write(p)
	
	// Then capture the output for the buffer
	for _, b := range p {
		if b == '\n' {
			if len(cw.lineBuf) > 0 {
				cw.capture.AddLine(string(cw.lineBuf))
				cw.lineBuf = cw.lineBuf[:0]
			}
		} else {
			cw.lineBuf = append(cw.lineBuf, b)
		}
	}
	
	return n, err
}