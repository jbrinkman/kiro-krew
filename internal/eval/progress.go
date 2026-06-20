package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ProgressTracker handles real-time progress display and time estimation.
type ProgressTracker struct {
	resultsDir     string
	totalTests     int
	completedTests int
	currentTest    string
	startTime      time.Time
	testStartTime  time.Time
	testDurations  []time.Duration
	mu             sync.RWMutex
	stopChan       chan struct{}
}

// ProgressState persists progress information for resumption.
type ProgressState struct {
	ResultsDir     string          `json:"results_dir"`
	TotalTests     int             `json:"total_tests"`
	CompletedTests int             `json:"completed_tests"`
	TestDurations  []time.Duration `json:"test_durations"`
	StartTime      time.Time       `json:"start_time"`
}

// NewProgressTracker creates a progress tracker for evaluation runs.
func NewProgressTracker(resultsDir string, totalTests int) *ProgressTracker {
	return &ProgressTracker{
		resultsDir:     resultsDir,
		totalTests:     totalTests,
		completedTests: 0,
		startTime:      time.Now(),
		testDurations:  make([]time.Duration, 0),
		stopChan:       make(chan struct{}),
	}
}

// LoadProgressState restores progress from a previous run.
func LoadProgressState(stateFile string) (*ProgressTracker, error) {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, err
	}

	var state ProgressState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &ProgressTracker{
		resultsDir:     state.ResultsDir,
		totalTests:     state.TotalTests,
		completedTests: state.CompletedTests,
		testDurations:  state.TestDurations,
		startTime:      state.StartTime,
		stopChan:       make(chan struct{}),
	}, nil
}

// SaveState persists current progress for resumption.
func (pt *ProgressTracker) SaveState(stateFile string) error {
	pt.mu.RLock()
	state := ProgressState{
		ResultsDir:     pt.resultsDir,
		TotalTests:     pt.totalTests,
		CompletedTests: pt.completedTests,
		TestDurations:  pt.testDurations,
		StartTime:      pt.startTime,
	}
	pt.mu.RUnlock()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	stateDir := filepath.Dir(stateFile)
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return err
	}

	return os.WriteFile(stateFile, data, 0644)
}

// Start displays initial progress information and starts periodic updates.
func (pt *ProgressTracker) Start() {
	fmt.Printf("📂 Results directory: %s\n", pt.resultsDir)
	fmt.Printf("🔄 Starting evaluation of %d test cases...\n\n", pt.totalTests)

	// Start periodic updates every 5 seconds
	go pt.updateLoop()
}

// StartTest begins tracking for a specific test case.
func (pt *ProgressTracker) StartTest(testName string) {
	pt.mu.Lock()
	pt.currentTest = testName
	pt.testStartTime = time.Now()
	pt.mu.Unlock()

	pt.displayProgress()
}

// CompleteTest marks a test as completed and records its duration.
func (pt *ProgressTracker) CompleteTest() {
	pt.mu.Lock()
	if !pt.testStartTime.IsZero() {
		duration := time.Since(pt.testStartTime)
		pt.testDurations = append(pt.testDurations, duration)
	}
	pt.completedTests++
	pt.currentTest = ""
	pt.testStartTime = time.Time{}
	pt.mu.Unlock()

	pt.displayProgress()
}

// Stop halts periodic updates.
func (pt *ProgressTracker) Stop() {
	close(pt.stopChan)
}

// updateLoop provides periodic progress updates during long-running tests.
func (pt *ProgressTracker) updateLoop() {
	ticker := time.NewTicker(7 * time.Second) // Update every 7 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pt.displayProgress()
		case <-pt.stopChan:
			return
		}
	}
}

// displayProgress shows current progress with time estimates.
func (pt *ProgressTracker) displayProgress() {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if pt.currentTest == "" && pt.completedTests == 0 {
		return // No progress to show yet
	}

	// Current test progress
	testProgress := fmt.Sprintf("[%d/%d]", pt.completedTests+1, pt.totalTests)
	if pt.currentTest != "" {
		elapsed := time.Since(pt.testStartTime)
		fmt.Printf("\r%s %s (elapsed: %s)", testProgress, pt.currentTest, elapsed.Truncate(time.Second))
	} else if pt.completedTests > 0 {
		fmt.Printf("\r%s Completed", testProgress)
	}

	// Time estimation
	if len(pt.testDurations) > 0 {
		avgDuration := pt.averageDuration()
		remaining := pt.totalTests - pt.completedTests
		if pt.currentTest != "" {
			remaining-- // Current test is in progress
		}

		if remaining > 0 {
			estimate := time.Duration(remaining) * avgDuration
			fmt.Printf(" | ETA: %s", estimate.Truncate(time.Second))
		}
	}

	if pt.currentTest != "" || pt.completedTests < pt.totalTests {
		fmt.Print("        ") // Clear any remaining characters
	} else {
		fmt.Println() // Final newline when done
	}
}

// averageDuration calculates the average test duration from completed tests.
func (pt *ProgressTracker) averageDuration() time.Duration {
	if len(pt.testDurations) == 0 {
		return 0
	}

	var total time.Duration
	for _, d := range pt.testDurations {
		total += d
	}
	return total / time.Duration(len(pt.testDurations))
}

// GetStateFile returns the path where progress state should be saved.
func (pt *ProgressTracker) GetStateFile() string {
	return filepath.Join(pt.resultsDir, ".progress")
}
