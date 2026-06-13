package watcher

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/jbrinkman/kiro-krew/internal/github"
)

// DependencyParser extracts issue numbers from issue body text
type DependencyParser struct{}

// DependencyBackoff tracks backoff state for a single issue
type DependencyBackoff struct {
	issueNumber   int
	failureCount  int
	nextCheckRound int
}

// BackoffTracker manages backoff tracking for multiple issues
type BackoffTracker struct {
	backoffs    map[int]*DependencyBackoff
	currentRound int
	mu          sync.RWMutex
}

// ParseDependencies extracts all issue numbers from issue body text using multiple formats
func (dp *DependencyParser) ParseDependencies(issueBody string) []int {
	return dp.extractIssueNumbers(issueBody)
}

// extractIssueNumbers finds issue numbers in text using various dependency formats
func (dp *DependencyParser) extractIssueNumbers(text string) []int {
	var issues []int
	seen := make(map[int]bool)
	
	patterns := []string{
		`(?i)depends\s+on\s+issue\s+#(\d+)`,       // "Depends on Issue #N"
		`(?i)dependencies:\s*#?(\d+)(?:\s*,\s*#?(\d+))*`, // "Dependencies: #N, #M" 
		`(?i)blocked\s+by:\s*#(\d+)`,              // "Blocked by: #N"
		`(?i)depends\s+on\s+\[issue\s+#(\d+)\]`,   // "Depends on [Issue #88](url)"
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(text, -1)
		
		for _, match := range matches {
			for i := 1; i < len(match); i++ {
				if match[i] != "" {
					if num, err := strconv.Atoi(match[i]); err == nil && !seen[num] {
						issues = append(issues, num)
						seen[num] = true
					}
				}
			}
		}
	}
	
	// Handle comma-separated format specially for "Dependencies: #N, #M"
	depPattern := regexp.MustCompile(`(?i)dependencies:\s*(.+?)(?:\n|$)`)
	depMatches := depPattern.FindAllStringSubmatch(text, -1)
	for _, match := range depMatches {
		if len(match) > 1 {
			deps := strings.Split(match[1], ",")
			for _, dep := range deps {
				dep = strings.TrimSpace(dep)
				dep = strings.TrimPrefix(dep, "#")
				if num, err := strconv.Atoi(dep); err == nil && !seen[num] {
					issues = append(issues, num)
					seen[num] = true
				}
			}
		}
	}
	
	return issues
}

// NewBackoffTracker creates a new backoff tracker
func NewBackoffTracker() *BackoffTracker {
	return &BackoffTracker{
		backoffs: make(map[int]*DependencyBackoff),
		currentRound: 0,
	}
}

// ShouldCheck returns true if the issue should be checked in the current round
func (bt *BackoffTracker) ShouldCheck(issueNumber int) bool {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	
	backoff, exists := bt.backoffs[issueNumber]
	if !exists {
		return true
	}
	
	return bt.currentRound >= backoff.nextCheckRound
}

// RecordFailure records a dependency check failure and calculates next check round
func (bt *BackoffTracker) RecordFailure(issueNumber int) {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	
	backoff, exists := bt.backoffs[issueNumber]
	if !exists {
		backoff = &DependencyBackoff{
			issueNumber: issueNumber,
			failureCount: 0,
		}
		bt.backoffs[issueNumber] = backoff
	}
	
	backoff.failureCount++
	
	// Calculate backoff delay: 2^failure_count rounds (max 16x)
	delay := 1 << backoff.failureCount
	if delay > 16 {
		delay = 16
	}
	
	backoff.nextCheckRound = bt.currentRound + delay
}

// GetRoundsUntilCheck returns how many rounds until the next check for an issue
func (bt *BackoffTracker) GetRoundsUntilCheck(issueNumber int) int {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	
	backoff, exists := bt.backoffs[issueNumber]
	if !exists {
		return 0
	}
	
	remaining := backoff.nextCheckRound - bt.currentRound
	if remaining < 0 {
		return 0
	}
	
	return remaining
}

// IncrementRound increments the current polling round
func (bt *BackoffTracker) IncrementRound() {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	
	bt.currentRound++
}

// ValidationResult contains the result of dependency validation
type ValidationResult struct {
	IsValid               bool
	UnresolvedDependencies []int
	CircularDependencies   []int
}

// DependencyValidator validates issue dependencies
type DependencyValidator struct {
	parser *DependencyParser
}

// NewDependencyValidator creates a new dependency validator
func NewDependencyValidator() *DependencyValidator {
	return &DependencyValidator{
		parser: &DependencyParser{},
	}
}

// ValidateIssue checks if an issue's dependencies are satisfied
func (dv *DependencyValidator) ValidateIssue(repo string, issueNumber int) (*ValidationResult, error) {
	// Get issue details
	details, err := github.GetIssueDetails(repo, issueNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue details: %w", err)
	}

	// Parse dependencies from issue body
	dependencies := dv.parser.ParseDependencies(details.Body)
	log.Printf("[watcher] debug: parsed %d dependencies for issue #%d: %v", len(dependencies), issueNumber, dependencies)
	
	if len(dependencies) == 0 {
		return &ValidationResult{IsValid: true}, nil
	}

	// Check for circular dependencies
	visited := make(map[int]bool)
	circular := dv.checkCircularDependencies(repo, issueNumber, visited)

	// Check dependency states
	var unresolved []int
	for _, dep := range dependencies {
		depDetails, err := github.GetIssueDetails(repo, dep)
		if err != nil {
			log.Printf("[watcher] error checking dependency #%d for issue #%d: %v", dep, issueNumber, err)
			unresolved = append(unresolved, dep)
			continue
		}

		if depDetails.State != "closed" {
			log.Printf("[watcher] debug: dependency #%d for issue #%d is in state '%s' (not closed)", dep, issueNumber, depDetails.State)
			unresolved = append(unresolved, dep)
		}
	}

	return &ValidationResult{
		IsValid:               len(unresolved) == 0,
		UnresolvedDependencies: unresolved,
		CircularDependencies:   circular,
	}, nil
}

// checkCircularDependencies detects circular dependencies
func (dv *DependencyValidator) checkCircularDependencies(repo string, issueNumber int, visited map[int]bool) []int {
	if visited[issueNumber] {
		return []int{issueNumber}
	}

	visited[issueNumber] = true

	details, err := github.GetIssueDetails(repo, issueNumber)
	if err != nil {
		return nil
	}

	dependencies := dv.parser.ParseDependencies(details.Body)
	for _, dep := range dependencies {
		if circular := dv.checkCircularDependencies(repo, dep, visited); len(circular) > 0 {
			return append(circular, issueNumber)
		}
	}

	delete(visited, issueNumber)
	return nil
}