package plan

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestIntegration_ParallelExecution verifies that independent tasks execute concurrently
func TestIntegration_ParallelExecution(t *testing.T) {
	executionLog := &taskExecutionLog{
		events: []taskEvent{},
	}

	spawner := createMockSpawnerWithLog(executionLog, 100*time.Millisecond)
	executor := NewExecutor(spawner)

	// Plan with 3 independent tasks (no dependencies)
	plan := &Plan{
		Version: "1.0",
		Tasks: []Task{
			{ID: "task-1", Agent: "builder", Description: "Task 1", Dependencies: []string{}},
			{ID: "task-2", Agent: "builder", Description: "Task 2", Dependencies: []string{}},
			{ID: "task-3", Agent: "builder", Description: "Task 3", Dependencies: []string{}},
		},
	}

	start := time.Now()
	summary, err := executor.ExecutePlan(plan)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("ExecutePlan failed: %v", err)
	}

	// Verify all tasks succeeded
	if summary.Successful != 3 {
		t.Errorf("Expected 3 successful tasks, got %d", summary.Successful)
	}

	// Verify parallel execution: should complete in ~100ms, not 300ms (sequential)
	// Allow some overhead, but should be significantly less than sequential
	if elapsed > 200*time.Millisecond {
		t.Errorf("Parallel execution took too long: %v (expected ~100ms)", elapsed)
	}

	// Verify all tasks started before any completed (true parallelism)
	startEvents := executionLog.getEventsByType("start")
	completeEvents := executionLog.getEventsByType("complete")

	if len(startEvents) != 3 || len(completeEvents) != 3 {
		t.Fatalf("Expected 3 start and 3 complete events, got %d starts and %d completes",
			len(startEvents), len(completeEvents))
	}

	// All start events should occur before all complete events
	lastStartTime := startEvents[len(startEvents)-1].timestamp
	firstCompleteTime := completeEvents[0].timestamp

	if !firstCompleteTime.After(lastStartTime) {
		t.Error("Tasks did not execute in parallel (complete events before all start events)")
	}
}

// TestIntegration_SequentialExecution verifies linear dependency chain executes sequentially
func TestIntegration_SequentialExecution(t *testing.T) {
	executionLog := &taskExecutionLog{
		events: []taskEvent{},
	}

	spawner := createMockSpawnerWithLog(executionLog, 50*time.Millisecond)
	executor := NewExecutor(spawner)

	// Plan with linear dependency chain: task-1 → task-2 → task-3
	plan := &Plan{
		Version: "1.0",
		Tasks: []Task{
			{ID: "task-1", Agent: "builder", Description: "Task 1", Dependencies: []string{}},
			{ID: "task-2", Agent: "builder", Description: "Task 2", Dependencies: []string{"task-1"}},
			{ID: "task-3", Agent: "builder", Description: "Task 3", Dependencies: []string{"task-2"}},
		},
	}

	start := time.Now()
	summary, err := executor.ExecutePlan(plan)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("ExecutePlan failed: %v", err)
	}

	// Verify all tasks succeeded
	if summary.Successful != 3 {
		t.Errorf("Expected 3 successful tasks, got %d", summary.Successful)
	}

	// Verify sequential execution: should take ~150ms (3 * 50ms)
	if elapsed < 140*time.Millisecond || elapsed > 200*time.Millisecond {
		t.Errorf("Sequential execution time unexpected: %v (expected ~150ms)", elapsed)
	}

	// Verify execution order: task-1 completes before task-2 starts, etc.
	events := executionLog.events
	if len(events) != 6 { // 3 starts + 3 completes
		t.Fatalf("Expected 6 events, got %d", len(events))
	}

	// Verify order: start-1, complete-1, start-2, complete-2, start-3, complete-3
	expectedOrder := []string{"task-1:start", "task-1:complete", "task-2:start", "task-2:complete", "task-3:start", "task-3:complete"}
	for i, expected := range expectedOrder {
		actual := fmt.Sprintf("%s:%s", events[i].taskID, events[i].eventType)
		if actual != expected {
			t.Errorf("Event %d: expected %s, got %s", i, expected, actual)
		}
	}
}

// TestIntegration_MixedDependencies verifies layered execution with mixed dependencies
func TestIntegration_MixedDependencies(t *testing.T) {
	executionLog := &taskExecutionLog{
		events: []taskEvent{},
	}

	spawner := createMockSpawnerWithLog(executionLog, 50*time.Millisecond)
	executor := NewExecutor(spawner)

	// Plan with mixed dependencies:
	// Layer 0: task-1, task-2 (parallel)
	// Layer 1: task-3 (depends on task-1 and task-2)
	// Layer 2: task-4, task-5 (both depend on task-3, parallel)
	plan := &Plan{
		Version: "1.0",
		Tasks: []Task{
			{ID: "task-1", Agent: "builder", Description: "Task 1", Dependencies: []string{}},
			{ID: "task-2", Agent: "builder", Description: "Task 2", Dependencies: []string{}},
			{ID: "task-3", Agent: "builder", Description: "Task 3", Dependencies: []string{"task-1", "task-2"}},
			{ID: "task-4", Agent: "builder", Description: "Task 4", Dependencies: []string{"task-3"}},
			{ID: "task-5", Agent: "builder", Description: "Task 5", Dependencies: []string{"task-3"}},
		},
	}

	start := time.Now()
	summary, err := executor.ExecutePlan(plan)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("ExecutePlan failed: %v", err)
	}

	// Verify all tasks succeeded
	if summary.Successful != 5 {
		t.Errorf("Expected 5 successful tasks, got %d", summary.Successful)
	}

	// Verify layered execution: ~150ms (3 layers * 50ms)
	// Layer 0: 50ms, Layer 1: 50ms, Layer 2: 50ms
	if elapsed < 140*time.Millisecond || elapsed > 200*time.Millisecond {
		t.Errorf("Layered execution time unexpected: %v (expected ~150ms)", elapsed)
	}

	// Verify task-1 and task-2 start at approximately the same time
	task1Start := executionLog.getEventTime("task-1", "start")
	task2Start := executionLog.getEventTime("task-2", "start")

	if task1Start.IsZero() || task2Start.IsZero() {
		t.Fatal("Could not find start times for task-1 or task-2")
	}

	parallelThreshold := 10 * time.Millisecond
	diff := task1Start.Sub(task2Start)
	if diff < 0 {
		diff = -diff
	}
	if diff > parallelThreshold {
		t.Error("task-1 and task-2 did not start in parallel")
	}

	// Verify task-3 starts after both task-1 and task-2 complete
	task1Complete := executionLog.getEventTime("task-1", "complete")
	task2Complete := executionLog.getEventTime("task-2", "complete")
	task3Start := executionLog.getEventTime("task-3", "start")

	if task3Start.Before(task1Complete) || task3Start.Before(task2Complete) {
		t.Error("task-3 started before dependencies completed")
	}

	// Verify task-4 and task-5 start at approximately the same time
	task4Start := executionLog.getEventTime("task-4", "start")
	task5Start := executionLog.getEventTime("task-5", "start")

	diff2 := task4Start.Sub(task5Start)
	if diff2 < 0 {
		diff2 = -diff2
	}
	if diff2 > parallelThreshold {
		t.Error("task-4 and task-5 did not start in parallel")
	}
}

// TestIntegration_InvalidPlanParsing tests handling of malformed plan artifacts
func TestIntegration_InvalidPlanParsing(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		wantErr  bool
	}{
		{
			name:     "valid plan",
			markdown: "# Spec\n```kiro-plan\nversion: \"1.0\"\ntasks:\n  - id: task-1\n    agent: builder\n    description: Test\n    dependencies: []\n```",
			wantErr:  false,
		},
		{
			name:     "no plan artifact",
			markdown: "# Spec\nThis is a legacy spec without a plan.",
			wantErr:  false, // Should return nil, nil (not an error)
		},
		{
			name:     "malformed YAML",
			markdown: "# Spec\n```kiro-plan\nversion: 1.0\ntasks:\n  - id: task-1\n    agent: builder\n    description: Test\n    dependencies: [\n```",
			wantErr:  true,
		},
		{
			name:     "multiple plan blocks",
			markdown: "# Spec\n```kiro-plan\nversion: \"1.0\"\ntasks: []\n```\n\n```kiro-plan\nversion: \"1.0\"\ntasks: []\n```",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := ParsePlanFromMarkdown([]byte(tt.markdown))

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				// For "no plan artifact", we expect nil plan
				if tt.name == "no plan artifact" && plan != nil {
					t.Error("Expected nil plan for legacy spec")
				}
			}
		})
	}
}

// TestIntegration_TaskFailureHandling verifies executor handles task failures gracefully
func TestIntegration_TaskFailureHandling(t *testing.T) {
	// Create spawner that fails task-2
	spawner := func(task Task) (int, error) {
		if task.ID == "task-2" {
			return 1, fmt.Errorf("task-2 failed")
		}
		return 0, nil
	}

	executor := NewExecutor(spawner)

	// Plan with dependencies:
	// task-1 (succeeds) → task-3 (should skip due to task-2 failure)
	// task-2 (fails) → task-3
	// task-4 (independent, should succeed)
	plan := &Plan{
		Version: "1.0",
		Tasks: []Task{
			{ID: "task-1", Agent: "builder", Description: "Task 1", Dependencies: []string{}},
			{ID: "task-2", Agent: "builder", Description: "Task 2", Dependencies: []string{}},
			{ID: "task-3", Agent: "builder", Description: "Task 3", Dependencies: []string{"task-1", "task-2"}},
			{ID: "task-4", Agent: "builder", Description: "Task 4", Dependencies: []string{}},
		},
	}

	summary, err := executor.ExecutePlan(plan)

	if err != nil {
		t.Fatalf("ExecutePlan failed: %v", err)
	}

	// Verify results
	if summary.Successful != 2 { // task-1 and task-4
		t.Errorf("Expected 2 successful tasks, got %d", summary.Successful)
	}

	if summary.Failed != 1 { // task-2
		t.Errorf("Expected 1 failed task, got %d", summary.Failed)
	}

	if summary.Skipped != 1 { // task-3
		t.Errorf("Expected 1 skipped task, got %d", summary.Skipped)
	}

	// Verify task-3 was skipped (not failed)
	for _, result := range summary.Results {
		if result.TaskID == "task-3" && result.Status != TaskStatusSkipped {
			t.Errorf("task-3 should be skipped, got status: %s", result.Status)
		}
	}
}

// TestIntegration_EndToEnd simulates complete workflow from markdown to execution
func TestIntegration_EndToEnd(t *testing.T) {
	markdown := `# Design Specification: Test Feature

## Solution Approach
Implement a test feature with parallel tasks.

## Machine-Readable Plan

` + "```kiro-plan" + `
version: "1.0"
tasks:
  - id: "task-1"
    agent: "builder"
    description: "Implement database layer"
    dependencies: []
    acceptance_criteria:
      - "Create schema"
      - "Add migrations"
    validation_commands:
      - "go test ./internal/db/..."
  
  - id: "task-2"
    agent: "builder"
    description: "Implement API layer"
    dependencies: []
    acceptance_criteria:
      - "Create endpoints"
      - "Add validation"
    validation_commands:
      - "go test ./internal/api/..."
  
  - id: "task-3"
    agent: "validator"
    description: "Integration testing"
    dependencies: ["task-1", "task-2"]
    acceptance_criteria:
      - "All tests pass"
    validation_commands:
      - "go test ./..."
` + "```" + `

## Implementation Details
...
`

	// Parse plan from markdown
	plan, err := ParsePlanFromMarkdown([]byte(markdown))
	if err != nil {
		t.Fatalf("ParsePlanFromMarkdown failed: %v", err)
	}

	if plan == nil {
		t.Fatal("Expected plan, got nil")
	}

	// Validate plan schema
	if err := plan.Validate(); err != nil {
		t.Fatalf("Plan validation failed: %v", err)
	}

	// Validate against mock registry
	registry := &AgentRegistry{
		agents: map[string]string{
			"builder":   "builder.json",
			"validator": "validator.json",
		},
	}

	validator := NewValidator(registry)
	if err := validator.ValidatePlan(plan); err != nil {
		t.Fatalf("Plan validation failed: %v", err)
	}

	// Execute plan
	spawner := func(task Task) (int, error) {
		// Simulate successful execution
		time.Sleep(10 * time.Millisecond)
		return 0, nil
	}

	executor := NewExecutor(spawner)
	summary, err := executor.ExecutePlan(plan)

	if err != nil {
		t.Fatalf("ExecutePlan failed: %v", err)
	}

	// Verify results
	if summary.TotalTasks != 3 {
		t.Errorf("Expected 3 total tasks, got %d", summary.TotalTasks)
	}

	if summary.Successful != 3 {
		t.Errorf("Expected 3 successful tasks, got %d", summary.Successful)
	}

	if summary.Failed != 0 {
		t.Errorf("Expected 0 failed tasks, got %d", summary.Failed)
	}
}

// Helper types and functions for execution logging

type taskEvent struct {
	taskID    string
	eventType string // "start" or "complete"
	timestamp time.Time
}

type taskExecutionLog struct {
	mu     sync.Mutex
	events []taskEvent
}

func (l *taskExecutionLog) logStart(taskID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = append(l.events, taskEvent{
		taskID:    taskID,
		eventType: "start",
		timestamp: time.Now(),
	})
}

func (l *taskExecutionLog) logComplete(taskID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = append(l.events, taskEvent{
		taskID:    taskID,
		eventType: "complete",
		timestamp: time.Now(),
	})
}

func (l *taskExecutionLog) getEventsByType(eventType string) []taskEvent {
	l.mu.Lock()
	defer l.mu.Unlock()
	var filtered []taskEvent
	for _, event := range l.events {
		if event.eventType == eventType {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func (l *taskExecutionLog) getEventTime(taskID, eventType string) time.Time {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, event := range l.events {
		if event.taskID == taskID && event.eventType == eventType {
			return event.timestamp
		}
	}
	return time.Time{}
}

func createMockSpawnerWithLog(log *taskExecutionLog, taskDuration time.Duration) TaskSpawner {
	return func(task Task) (int, error) {
		log.logStart(task.ID)
		time.Sleep(taskDuration)
		log.logComplete(task.ID)
		return 0, nil
	}
}
