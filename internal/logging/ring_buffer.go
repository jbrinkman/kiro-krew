package logging

import (
	"sync"
	"time"

	"github.com/charmbracelet/log"
)

// RingBuffer implements a thread-safe circular buffer for log entries with FIFO behavior.
// It provides O(1) performance for add and remove operations using slice-based storage
// with head and tail pointers.
type RingBuffer struct {
	entries  []LogEntry   // Pre-allocated slice for log entries
	head     int          // Index of oldest entry (read position)
	tail     int          // Index of next write position
	size     int          // Current number of entries in buffer
	capacity int          // Maximum number of entries (FIFO when full)
	mutex    sync.RWMutex // Protects concurrent access
}

// NewRingBuffer creates a new ring buffer with the specified maximum capacity.
// The buffer uses FIFO (First In, First Out) behavior - when full, the oldest
// entries are automatically overwritten by new entries.
func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		capacity = DefaultMaxBufferLines
	}

	return &RingBuffer{
		entries:  make([]LogEntry, capacity),
		head:     0,
		tail:     0,
		size:     0,
		capacity: capacity,
	}
}

// Add appends a new log entry to the buffer. If the buffer is full, the oldest
// entry is overwritten (FIFO behavior). This operation is O(1).
func (rb *RingBuffer) Add(level log.Level, message string, keyvals ...interface{}) {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()

	// Parse key-value pairs into metadata map
	metadata := make(map[string]interface{})
	for i := 0; i < len(keyvals); i += 2 {
		if i+1 < len(keyvals) {
			if key, ok := keyvals[i].(string); ok {
				metadata[key] = keyvals[i+1]
			}
		}
	}

	// Create log entry
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Metadata:  metadata,
	}

	// Write to tail position
	rb.entries[rb.tail] = entry
	rb.tail = (rb.tail + 1) % rb.capacity

	// Update size and head position
	if rb.size < rb.capacity {
		rb.size++
	} else {
		// Buffer is full, advance head (overwrite oldest)
		rb.head = (rb.head + 1) % rb.capacity
	}
}

// Get retrieves all log entries from the buffer without removing them.
// Entries are returned in chronological order (oldest first).
// This operation is O(n) where n is the number of entries.
func (rb *RingBuffer) Get() []LogEntry {
	rb.mutex.RLock()
	defer rb.mutex.RUnlock()

	if rb.size == 0 {
		return []LogEntry{}
	}

	result := make([]LogEntry, rb.size)

	// Copy entries from head to tail
	if rb.head < rb.tail {
		// Simple case: head < tail, no wrap around
		copy(result, rb.entries[rb.head:rb.tail])
	} else {
		// Wrap around case: copy from head to end, then from start to tail
		n := copy(result, rb.entries[rb.head:])
		copy(result[n:], rb.entries[:rb.tail])
	}

	return result
}

// GetRecent retrieves the last N log entries from the buffer without removing them.
// If N is greater than the buffer size, all entries are returned.
// Entries are returned in chronological order (oldest first among the N entries).
// This operation is O(min(n, size)).
func (rb *RingBuffer) GetRecent(n int) []LogEntry {
	rb.mutex.RLock()
	defer rb.mutex.RUnlock()

	if rb.size == 0 || n <= 0 {
		return []LogEntry{}
	}

	// Limit n to available entries
	if n > rb.size {
		n = rb.size
	}

	result := make([]LogEntry, n)

	// Calculate start position for the last N entries
	startPos := (rb.head + rb.size - n) % rb.capacity

	// Copy entries
	if startPos < rb.tail || (startPos >= rb.tail && rb.size < rb.capacity) {
		// Simple case: no wrap around for the slice we want
		endPos := (startPos + n) % rb.capacity
		if endPos > startPos {
			copy(result, rb.entries[startPos:endPos])
		} else {
			// Wrap around: copy from startPos to end, then from 0 to endPos
			copied := copy(result, rb.entries[startPos:])
			copy(result[copied:], rb.entries[:endPos])
		}
	} else {
		// Wrap around case
		copied := copy(result, rb.entries[startPos:])
		copy(result[copied:], rb.entries[:rb.tail])
	}

	return result
}

// Clear removes all entries from the buffer, resetting it to empty state.
// This operation is O(1).
func (rb *RingBuffer) Clear() {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()

	rb.head = 0
	rb.tail = 0
	rb.size = 0
}

// Size returns the current number of entries in the buffer.
// This operation is O(1).
func (rb *RingBuffer) Size() int {
	rb.mutex.RLock()
	defer rb.mutex.RUnlock()
	return rb.size
}

// Capacity returns the maximum number of entries the buffer can hold.
// This operation is O(1).
func (rb *RingBuffer) Capacity() int {
	rb.mutex.RLock()
	defer rb.mutex.RUnlock()
	return rb.capacity
}

// IsFull returns true if the buffer is at capacity.
// This operation is O(1).
func (rb *RingBuffer) IsFull() bool {
	rb.mutex.RLock()
	defer rb.mutex.RUnlock()
	return rb.size == rb.capacity
}

// IsEmpty returns true if the buffer contains no entries.
// This operation is O(1).
func (rb *RingBuffer) IsEmpty() bool {
	rb.mutex.RLock()
	defer rb.mutex.RUnlock()
	return rb.size == 0
}

// Iterator provides a read-only iterator interface for consuming log entries
// without modifying the buffer. This allows multiple consumers to read the
// same buffer concurrently.
type Iterator struct {
	buffer   *RingBuffer
	position int
	count    int
}

// NewIterator creates a new iterator for reading entries from the ring buffer.
// The iterator snapshots the current state and iterates through entries in
// chronological order.
func (rb *RingBuffer) NewIterator() *Iterator {
	rb.mutex.RLock()
	defer rb.mutex.RUnlock()

	return &Iterator{
		buffer:   rb,
		position: rb.head,
		count:    0,
	}
}

// Next advances the iterator to the next entry and returns it.
// Returns nil when no more entries are available.
func (it *Iterator) Next() *LogEntry {
	it.buffer.mutex.RLock()
	defer it.buffer.mutex.RUnlock()

	// Check if we've consumed all entries
	if it.count >= it.buffer.size {
		return nil
	}

	// Get current entry
	entry := it.buffer.entries[it.position]

	// Advance position
	it.position = (it.position + 1) % it.buffer.capacity
	it.count++

	return &entry
}

// HasNext returns true if there are more entries to iterate over.
func (it *Iterator) HasNext() bool {
	it.buffer.mutex.RLock()
	defer it.buffer.mutex.RUnlock()

	return it.count < it.buffer.size
}
