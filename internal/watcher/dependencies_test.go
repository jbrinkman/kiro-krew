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