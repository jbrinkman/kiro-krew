package watcher

import (
	"reflect"
	"sort"
	"testing"
)

func TestParseDependencies(t *testing.T) {
	parser := &DependencyParser{}

	tests := []struct {
		name     string
		input    string
		expected []int
	}{
		{
			name:     "depends on issue format",
			input:    "Depends on Issue #88",
			expected: []int{88},
		},
		{
			name:     "dependencies comma format",
			input:    "Dependencies: #88, #89",
			expected: []int{88, 89},
		},
		{
			name:     "blocked by format",
			input:    "Blocked by: #90",
			expected: []int{90},
		},
		{
			name:     "markdown link format",
			input:    "Depends on [Issue #88](https://github.com/user/repo/issues/88)",
			expected: []int{88},
		},
		{
			name:     "multiple formats mixed",
			input:    "Depends on Issue #88\nDependencies: #89, #90\nBlocked by: #91",
			expected: []int{88, 89, 90, 91},
		},
		{
			name:     "no dependencies",
			input:    "This is a regular issue with no dependencies",
			expected: []int{},
		},
		{
			name:     "malformed input",
			input:    "Depends on Issue # \nDependencies: #abc, #",
			expected: []int{},
		},
		{
			name:     "duplicate issue numbers",
			input:    "Depends on Issue #88\nDependencies: #88, #89",
			expected: []int{88, 89},
		},
		{
			name:     "case insensitive",
			input:    "depends on issue #88\nDEPENDENCIES: #89\nblocked by: #90",
			expected: []int{88, 89, 90},
		},
		{
			name:     "dependencies without hash",
			input:    "Dependencies: 88, 89",
			expected: []int{88, 89},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ParseDependencies(tt.input)

			// Sort both slices for comparison since order doesn't matter
			sort.Ints(result)
			sort.Ints(tt.expected)

			// Handle nil vs empty slice comparison
			if len(result) != len(tt.expected) {
				t.Errorf("ParseDependencies() = %v, expected %v", result, tt.expected)
			} else if len(result) > 0 && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseDependencies() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestBackoffTracker(t *testing.T) {
	t.Run("new issue should be checked", func(t *testing.T) {
		bt := NewBackoffTracker()
		bt.IncrementRound()
		if !bt.ShouldCheck(1) {
			t.Error("expected new issue to be checked")
		}
	})

	t.Run("backoff progression 2x 4x 8x 16x max", func(t *testing.T) {
		bt := NewBackoffTracker()
		bt.IncrementRound() // round 1

		// First failure: delay 2 rounds (next check at round 3)
		bt.RecordFailure(1)
		if bt.ShouldCheck(1) {
			t.Error("expected issue to be in backoff after first failure")
		}
		if got := bt.GetRoundsUntilCheck(1); got != 2 {
			t.Errorf("expected 2 rounds until check, got %d", got)
		}

		bt.IncrementRound() // round 2
		if bt.ShouldCheck(1) {
			t.Error("expected issue still in backoff at round 2")
		}

		bt.IncrementRound() // round 3
		if !bt.ShouldCheck(1) {
			t.Error("expected issue to be checkable at round 3")
		}

		// Second failure: delay 4 rounds (next check at round 7)
		bt.RecordFailure(1)
		if bt.ShouldCheck(1) {
			t.Error("expected issue to be in backoff after second failure")
		}
		if got := bt.GetRoundsUntilCheck(1); got != 4 {
			t.Errorf("expected 4 rounds until check, got %d", got)
		}

		// Third failure: delay 8 rounds
		for i := 0; i < 4; i++ {
			bt.IncrementRound()
		}
		bt.RecordFailure(1)
		if got := bt.GetRoundsUntilCheck(1); got != 8 {
			t.Errorf("expected 8 rounds until check, got %d", got)
		}

		// Fourth failure: delay 16 rounds (max)
		for i := 0; i < 8; i++ {
			bt.IncrementRound()
		}
		bt.RecordFailure(1)
		if got := bt.GetRoundsUntilCheck(1); got != 16 {
			t.Errorf("expected 16 rounds until check (max), got %d", got)
		}

		// Fifth failure: still capped at 16
		for i := 0; i < 16; i++ {
			bt.IncrementRound()
		}
		bt.RecordFailure(1)
		if got := bt.GetRoundsUntilCheck(1); got != 16 {
			t.Errorf("expected 16 rounds (cap), got %d", got)
		}
	})
}
