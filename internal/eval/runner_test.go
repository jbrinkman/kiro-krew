package eval

import (
	"testing"
)

func TestScoreDeterministic_AcceptanceCriteriaTestability(t *testing.T) {
	criterion := Criterion{
		Name:    "acceptance_criteria_testability",
		Scoring: "1-5",
	}

	tests := []struct {
		name           string
		output         string
		expectedScore  int
		expectedSkip   bool
	}{
		{
			name:          "high testability - many patterns",
			output:        "Test that the command executes successfully. Verify the file is created. Check that validation passes. Must validate output.",
			expectedScore: 5,
			expectedSkip:  false,
		},
		{
			name:          "medium testability - some patterns",
			output:        "Should work correctly and file must be present.",
			expectedScore: 5,
			expectedSkip:  false,
		},
		{
			name:          "low testability - few patterns",
			output:        "The feature should be implemented.",
			expectedScore: 1,
			expectedSkip:  false,
		},
		{
			name:          "no output",
			output:        "",
			expectedScore: 0,
			expectedSkip:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := TestCase{Output: tt.output}
			score, _, skipped := scoreDeterministic(criterion, tc)
			if score != tt.expectedScore {
				t.Errorf("expected score %d, got %d", tt.expectedScore, score)
			}
			if skipped != tt.expectedSkip {
				t.Errorf("expected skipped %v, got %v", tt.expectedSkip, skipped)
			}
		})
	}
}

func TestScoreDeterministic_TestExecution(t *testing.T) {
	criterion := Criterion{
		Name:    "test_execution",
		Scoring: "1-5",
	}

	tests := []struct {
		name           string
		output         string
		expectedScore  int
		expectedSkip   bool
	}{
		{
			name:          "clear execution evidence",
			output:        "Command executed successfully with exit code 0. Output shows passed tests.",
			expectedScore: 5,
			expectedSkip:  false,
		},
		{
			name:          "some execution evidence",
			output:        "Tests were run and passed successfully.",
			expectedScore: 5,
			expectedSkip:  false,
		},
		{
			name:          "minimal execution evidence",
			output:        "The command was executed.",
			expectedScore: 2,
			expectedSkip:  false,
		},
		{
			name:          "no execution evidence",
			output:        "The feature was implemented correctly.",
			expectedScore: 1,
			expectedSkip:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := TestCase{Output: tt.output}
			score, _, skipped := scoreDeterministic(criterion, tc)
			if score != tt.expectedScore {
				t.Errorf("expected score %d, got %d", tt.expectedScore, score)
			}
			if skipped != tt.expectedSkip {
				t.Errorf("expected skipped %v, got %v", tt.expectedSkip, skipped)
			}
		})
	}
}

func TestScoreDeterministic_CodeCorrectness(t *testing.T) {
	criterion := Criterion{
		Name:    "code_correctness",
		Scoring: "1-5",
	}

	tests := []struct {
		name           string
		output         string
		expectedScore  int
		expectedSkip   bool
	}{
		{
			name:          "clear success indicators",
			output:        "Code compiled successfully with no errors and runs correctly.",
			expectedScore: 5,
			expectedSkip:  false,
		},
		{
			name:          "success with working code",
			output:        "The implementation is working and tests passed.",
			expectedScore: 5,
			expectedSkip:  false,
		},
		{
			name:          "error indicators present",
			output:        "Compilation failed with syntax error in main.go.",
			expectedScore: 1,
			expectedSkip:  false,
		},
		{
			name:          "no clear indicators",
			output:        "The feature has been implemented as requested.",
			expectedScore: 2,
			expectedSkip:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := TestCase{Output: tt.output}
			score, _, skipped := scoreDeterministic(criterion, tc)
			if score != tt.expectedScore {
				t.Errorf("expected score %d, got %d", tt.expectedScore, score)
			}
			if skipped != tt.expectedSkip {
				t.Errorf("expected skipped %v, got %v", tt.expectedSkip, skipped)
			}
		})
	}
}

func TestScoreDeterministic_TestCoverage(t *testing.T) {
	criterion := Criterion{
		Name:    "test_coverage",
		Scoring: "1-5",
	}

	tests := []struct {
		name           string
		output         string
		expectedScore  int
		expectedSkip   bool
	}{
		{
			name:          "comprehensive test coverage",
			output:        "Added test_user.go with comprehensive test coverage. All functionality is tested.",
			expectedScore: 5,
			expectedSkip:  false,
		},
		{
			name:          "some test coverage",
			output:        "Tests were added to verify the new functionality.",
			expectedScore: 2,
			expectedSkip:  false,
		},
		{
			name:          "minimal test references",
			output:        "The feature includes test cases.",
			expectedScore: 2,
			expectedSkip:  false,
		},
		{
			name:          "no test coverage",
			output:        "The feature has been implemented successfully.",
			expectedScore: 1,
			expectedSkip:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := TestCase{Output: tt.output}
			score, _, skipped := scoreDeterministic(criterion, tc)
			if score != tt.expectedScore {
				t.Errorf("expected score %d, got %d", tt.expectedScore, score)
			}
			if skipped != tt.expectedSkip {
				t.Errorf("expected skipped %v, got %v", tt.expectedSkip, skipped)
			}
		})
	}
}

func TestScoreDeterministic_ExistingCheckers(t *testing.T) {
	// Test that existing checkers still work
	tests := []struct {
		name       string
		criterion  Criterion
		output     string
		expectSkip bool
	}{
		{
			name: "completeness checker",
			criterion: Criterion{
				Name:    "completeness",
				Scoring: "1-5",
			},
			output:     "## Overview\n### Details",
			expectSkip: false,
		},
		{
			name: "file_naming checker",
			criterion: Criterion{
				Name:    "file_naming",
				Scoring: "1-5",
			},
			output:     "Created app_docs/feature-auth.md",
			expectSkip: false,
		},
		{
			name: "unknown checker",
			criterion: Criterion{
				Name:    "unknown_checker",
				Scoring: "1-5",
			},
			output:     "some output",
			expectSkip: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := TestCase{Output: tt.output}
			_, _, skipped := scoreDeterministic(tt.criterion, tc)
			if skipped != tt.expectSkip {
				t.Errorf("expected skipped %v, got %v", tt.expectSkip, skipped)
			}
		})
	}
}