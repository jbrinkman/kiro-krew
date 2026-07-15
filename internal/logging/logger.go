package logging

import (
	"fmt"
	"io"
	"sync"

	"github.com/charmbracelet/log"
)

var (
	globalLogger *log.Logger
	loggerMutex  sync.RWMutex
	isActive     bool
)

// Initialize creates and configures the global logger with the specified level and handlers.
// The logger starts inactive (no handlers) and is activated when the log viewer opens.
func Initialize(level string, handlers ...io.Writer) error {
	loggerMutex.Lock()
	defer loggerMutex.Unlock()

	// Create output writer from handlers
	var output io.Writer
	if len(handlers) == 0 {
		output = io.Discard
	} else if len(handlers) == 1 {
		output = handlers[0]
	} else {
		output = io.MultiWriter(handlers...)
	}

	// Create new logger instance
	globalLogger = log.New(output)

	// Set log level
	logLevel, err := parseLevel(level)
	if err != nil {
		return err
	}
	globalLogger.SetLevel(logLevel)

	// Set formatting options for structured output
	globalLogger.SetReportTimestamp(true)
	globalLogger.SetReportCaller(false) // Can be enabled for debugging
	// Use JSON formatter for structured output that can be parsed reliably
	globalLogger.SetFormatter(log.JSONFormatter)

	// Mark as active if handlers were provided
	isActive = len(handlers) > 0

	return nil
}

// SetLevel dynamically changes the log level of the global logger.
// Valid levels: debug, info, warn, error
func SetLevel(level string) error {
	loggerMutex.Lock()
	defer loggerMutex.Unlock()

	if globalLogger == nil {
		return nil // Silently ignore if logger not initialized
	}

	logLevel, err := parseLevel(level)
	if err != nil {
		return err
	}

	globalLogger.SetLevel(logLevel)
	return nil
}

// GetLogger returns the global logger instance for direct access.
// Returns nil if the logger has not been initialized.
func GetLogger() *log.Logger {
	loggerMutex.RLock()
	defer loggerMutex.RUnlock()
	return globalLogger
}

// IsActive returns whether logging is currently active (has handlers attached).
func IsActive() bool {
	loggerMutex.RLock()
	defer loggerMutex.RUnlock()
	return isActive
}

// Activate attaches handlers to the logger and marks it as active.
func Activate(handlers ...io.Writer) {
	loggerMutex.Lock()
	defer loggerMutex.Unlock()

	// Create output writer from handlers
	var output io.Writer
	if len(handlers) == 0 {
		output = io.Discard
	} else if len(handlers) == 1 {
		output = handlers[0]
	} else {
		output = io.MultiWriter(handlers...)
	}

	if globalLogger == nil {
		// Initialize with default level if not already initialized
		globalLogger = log.New(output)
		globalLogger.SetLevel(log.InfoLevel)
		globalLogger.SetReportTimestamp(true)
		// Use JSON formatter for structured output that can be parsed reliably
		globalLogger.SetFormatter(log.JSONFormatter)
	} else {
		// Add new handlers to existing logger
		globalLogger.SetOutput(output)
	}

	isActive = true
}

// Deactivate removes all handlers from the logger and marks it as inactive.
func Deactivate() {
	loggerMutex.Lock()
	defer loggerMutex.Unlock()

	if globalLogger != nil {
		// Set output to discard to effectively disable logging
		globalLogger.SetOutput(io.Discard)
	}

	isActive = false
}

// Helper functions for convenient logging with structured fields

// Debug logs a debug-level message with optional key-value pairs.
func Debug(msg string, keyvals ...interface{}) {
	loggerMutex.RLock()
	defer loggerMutex.RUnlock()

	if globalLogger != nil && isActive {
		globalLogger.Debug(msg, keyvals...)
	}
}

// Info logs an info-level message with optional key-value pairs.
func Info(msg string, keyvals ...interface{}) {
	loggerMutex.RLock()
	defer loggerMutex.RUnlock()

	if globalLogger != nil && isActive {
		globalLogger.Info(msg, keyvals...)
	}
}

// Warn logs a warning-level message with optional key-value pairs.
func Warn(msg string, keyvals ...interface{}) {
	loggerMutex.RLock()
	defer loggerMutex.RUnlock()

	if globalLogger != nil && isActive {
		globalLogger.Warn(msg, keyvals...)
	}
}

// Error logs an error-level message with optional key-value pairs.
func Error(msg string, keyvals ...interface{}) {
	loggerMutex.RLock()
	defer loggerMutex.RUnlock()

	if globalLogger != nil && isActive {
		globalLogger.Error(msg, keyvals...)
	}
}

// With returns a new logger with the given key-value pairs added to all log messages.
// This is useful for adding context to a series of related log messages.
func With(keyvals ...interface{}) *log.Logger {
	loggerMutex.RLock()
	defer loggerMutex.RUnlock()

	if globalLogger != nil {
		return globalLogger.With(keyvals...)
	}
	return nil
}

// parseLevel converts a string level to a log.Level constant.
func parseLevel(level string) (log.Level, error) {
	switch level {
	case "debug":
		return log.DebugLevel, nil
	case "info":
		return log.InfoLevel, nil
	case "warn":
		return log.WarnLevel, nil
	case "error":
		return log.ErrorLevel, nil
	default:
		return log.InfoLevel, fmt.Errorf("invalid log level %q: must be debug, info, warn, or error", level)
	}
}

// ValidLevels returns a slice of all valid log level strings.
// This is the single source of truth for level validation.
func ValidLevels() []string {
	return []string{LevelDebug, LevelInfo, LevelWarn, LevelError}
}

// IsValidLevel checks if the given level string is valid.
func IsValidLevel(level string) bool {
	for _, validLevel := range ValidLevels() {
		if level == validLevel {
			return true
		}
	}
	return false
}
