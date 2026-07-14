package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileHandler implements io.Writer for file-based logging with automatic rotation.
// It writes logs to files in a configurable directory with timestamp-based naming.
// When the file size exceeds the configured threshold, it automatically rotates
// to a new file with an updated timestamp.
type FileHandler struct {
	mu             sync.Mutex
	config         FileOutputConfig
	currentFile    *os.File
	currentSize    int64
	currentPath    string
	closeRequested bool
}

// NewFileHandler creates a new FileHandler with the given configuration.
// It creates the log directory if it doesn't exist and opens the initial log file.
func NewFileHandler(config FileOutputConfig) (*FileHandler, error) {
	fh := &FileHandler{
		config: config,
	}

	// Ensure log directory exists
	if err := fh.ensureLogDir(); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open initial log file
	if err := fh.rotate(); err != nil {
		return nil, fmt.Errorf("failed to create initial log file: %w", err)
	}

	return fh, nil
}

// Write implements io.Writer interface. It writes log data to the current file
// and rotates to a new file if the size threshold is exceeded.
func (fh *FileHandler) Write(p []byte) (n int, err error) {
	fh.mu.Lock()
	defer fh.mu.Unlock()

	// Check if close was requested
	if fh.closeRequested {
		return 0, io.ErrClosedPipe
	}

	// Check if rotation is needed before writing
	if fh.needsRotation(len(p)) {
		if err := fh.rotate(); err != nil {
			// Log to stderr if rotation fails
			fmt.Fprintf(os.Stderr, "log rotation failed: %v\n", err)
			// If we no longer have an open file, fail this write
			if fh.currentFile == nil {
				return 0, fmt.Errorf("log rotation failed and no file is open: %w", err)
			}
		}
	}

	// Write to current file
	n, err = fh.currentFile.Write(p)
	if err != nil {
		return n, fmt.Errorf("failed to write to log file: %w", err)
	}

	// Update current size
	fh.currentSize += int64(n)

	return n, nil
}

// Close closes the current log file and marks the handler as closed.
// Subsequent writes will return io.ErrClosedPipe.
func (fh *FileHandler) Close() error {
	fh.mu.Lock()
	defer fh.mu.Unlock()

	fh.closeRequested = true

	if fh.currentFile != nil {
		err := fh.currentFile.Close()
		fh.currentFile = nil
		return err
	}

	return nil
}

// ensureLogDir creates the log directory if it doesn't exist.
// Directory is created with 0700 permissions (rwx------).
func (fh *FileHandler) ensureLogDir() error {
	// Get absolute path
	absPath, err := filepath.Abs(fh.config.LogDir)
	if err != nil {
		return fmt.Errorf("failed to resolve log directory path: %w", err)
	}

	// Check if directory exists
	info, err := os.Stat(absPath)
	if err == nil {
		// Directory exists, verify it's a directory
		if !info.IsDir() {
			return fmt.Errorf("log path exists but is not a directory: %s", absPath)
		}
		return nil
	}

	// Directory doesn't exist, create it
	if os.IsNotExist(err) {
		if err := os.MkdirAll(absPath, 0700); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}
		return nil
	}

	// Other error occurred
	return fmt.Errorf("failed to check log directory: %w", err)
}

// needsRotation checks if the current file needs rotation based on size.
// Returns true if adding the next write would exceed the configured max size.
func (fh *FileHandler) needsRotation(nextWriteSize int) bool {
	if fh.currentFile == nil {
		return true
	}

	maxBytes := int64(fh.config.MaxFileSizeMB) * 1024 * 1024
	return (fh.currentSize + int64(nextWriteSize)) >= maxBytes
}

// rotate closes the current file and opens a new one with an updated timestamp.
// File naming format: debug-YYYY-MM-DD-HHMM.log (minute resolution)
func (fh *FileHandler) rotate() error {
	// Generate new filename with current timestamp
	now := time.Now()
	filename := fmt.Sprintf("debug-%s.log", now.Format("2006-01-02-1504"))

	// Get absolute path for log file
	absDir, err := filepath.Abs(fh.config.LogDir)
	if err != nil {
		return fmt.Errorf("failed to resolve log directory: %w", err)
	}

	logPath := filepath.Join(absDir, filename)

	// Check if file already exists (handle minute-resolution collisions)
	if _, err := os.Stat(logPath); err == nil {
		// File exists, append a sequence number
		for i := 1; i < 100; i++ {
			seqFilename := fmt.Sprintf("debug-%s-%d.log", now.Format("2006-01-02-1504"), i)
			logPath = filepath.Join(absDir, seqFilename)
			if _, err := os.Stat(logPath); os.IsNotExist(err) {
				break
			}
		}
	}

	// Open new log file with create/append flags
	// Using 0600 permissions (rw-------)
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Get current file size (in case file already exists)
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to stat log file: %w", err)
	}

	// Close old file only after successfully opening the new one
	oldFile := fh.currentFile

	// Update handler state
	fh.currentFile = file
	fh.currentPath = logPath
	fh.currentSize = info.Size()

	// Close old file if it exists
	if oldFile != nil {
		if err := oldFile.Close(); err != nil {
			// Log error but don't fail rotation since new file is ready
			fmt.Fprintf(os.Stderr, "failed to close old log file during rotation: %v\n", err)
		}
	}

	return nil
}

// GetCurrentFilePath returns the path of the current log file.
// Returns empty string if no file is open.
func (fh *FileHandler) GetCurrentFilePath() string {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	return fh.currentPath
}

// GetCurrentSize returns the size of the current log file in bytes.
func (fh *FileHandler) GetCurrentSize() int64 {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	return fh.currentSize
}
