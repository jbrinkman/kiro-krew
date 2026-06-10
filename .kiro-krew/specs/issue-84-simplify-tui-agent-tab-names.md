# Design Specification: Simplify TUI Agent Tab Names

**Issue:** #84  
**Title:** Simplify TUI agent tab names to show issue numbers clearly  
**Closes:** #84

## Problem Analysis

The issue describes agent tab names showing as "Agent agent-80-18752" instead of the desired "Issue 80" format. However, upon code inspection, the current implementation in `internal/tui/agent_tab.go` already includes the correct logic:

```go
func (at *AgentTab) Title() string {
	if agent := at.outputView.manager.GetAgent(at.agentID); agent != nil {
		return fmt.Sprintf("Issue %d", agent.IssueNumber)
	}
	return "Agent " + at.agentID
}
```

The fallback to the old format (`"Agent " + at.agentID`) only occurs when the agent lookup fails.

## Root Cause Investigation

The issue likely occurs in one of these scenarios:
1. **Agent lookup failure**: The `manager.GetAgent(at.agentID)` call returns `nil`
2. **Timing issue**: Tab is created before agent is fully registered in the manager
3. **Race condition**: Agent is removed from manager while tab still exists
4. **Test environment**: Mock agents or test scenarios where IssueNumber isn't set

## Solution Approach

**Primary Solution**: Enhance robustness of the existing implementation by:
1. Adding fallback parsing logic to extract issue number from agent ID when direct lookup fails
2. Improving error handling and logging for debugging
3. Adding validation tests to ensure proper tab naming

**Alternative Solution**: If primary approach fails, implement agent ID parsing as the main strategy.

## Relevant Files

### Files to Modify
- `internal/tui/agent_tab.go` - Enhance `Title()` method with fallback parsing
- `internal/tui/agent_tab_test.go` - Add comprehensive tests for title generation (to be created)

### Files for Reference
- `internal/agent/manager.go` - Understanding agent structure and ID generation
- `internal/tui/tab_manager_test.go` - Existing test patterns for tabs
- `internal/tui/output_view.go` - Agent lookup patterns

## Implementation Strategy

### Phase 1: Robust Fallback Implementation

```go
func (at *AgentTab) Title() string {
	// Primary: Use direct agent lookup
	if agent := at.outputView.manager.GetAgent(at.agentID); agent != nil {
		return fmt.Sprintf("Issue %d", agent.IssueNumber)
	}
	
	// Fallback: Parse issue number from agent ID format "agent-{issueNumber}-{timestamp}"
	if issueNum, err := extractIssueNumberFromAgentID(at.agentID); err == nil {
		return fmt.Sprintf("Issue %d", issueNum)
	}
	
	// Last resort: Use old format
	return "Agent " + at.agentID
}

func extractIssueNumberFromAgentID(agentID string) (int, error) {
	// Parse "agent-{issueNumber}-{timestamp}" format
	parts := strings.Split(agentID, "-")
	if len(parts) >= 3 && parts[0] == "agent" {
		return strconv.Atoi(parts[1])
	}
	return 0, fmt.Errorf("invalid agent ID format: %s", agentID)
}
```

### Phase 2: Comprehensive Testing

Add test coverage for:
- Normal operation with valid agents
- Agent lookup failures
- Invalid agent ID formats
- Edge cases and race conditions

### Phase 3: Validation and Monitoring

Add logging to identify when fallback parsing is used, helping diagnose any underlying agent management issues.

## Team Orchestration

**Single Developer Task**: This is a focused implementation that can be handled by one developer as it involves:
- Minimal code changes (one file primarily)
- Self-contained logic
- Straightforward testing

**No Cross-Team Dependencies**: The change is isolated to the TUI layer and doesn't affect:
- Agent spawning logic
- Agent management workflows
- External APIs or interfaces

## Step-by-Step Task Breakdown

### Task 1: Implement Robust Title Generation
**Acceptance Criteria:**
- [ ] Agent tab titles show "Issue N" format when agent lookup succeeds
- [ ] Fallback parsing extracts issue number from agent ID when lookup fails
- [ ] Graceful degradation to "Agent {agentID}" for invalid formats
- [ ] Implementation handles all edge cases without panicking

**Implementation Steps:**
1. Add `extractIssueNumberFromAgentID` helper function
2. Enhance `Title()` method with fallback logic
3. Add appropriate imports (`strings`, `strconv`)
4. Add debug logging for troubleshooting

### Task 2: Create Comprehensive Tests
**Acceptance Criteria:**
- [ ] Test cases cover normal agent lookup success
- [ ] Test cases cover agent lookup failure with valid agent ID format
- [ ] Test cases cover invalid agent ID formats
- [ ] Test cases verify exact title format ("Issue N")
- [ ] All tests pass consistently

**Implementation Steps:**
1. Create `internal/tui/agent_tab_test.go`
2. Implement mock agent manager for testing
3. Test all code paths in `Title()` method
4. Verify title format consistency

### Task 3: Integration Validation
**Acceptance Criteria:**
- [ ] Manual testing confirms correct tab names in real TUI
- [ ] No regression in existing tab functionality
- [ ] Performance remains acceptable
- [ ] Edge cases handled gracefully

**Implementation Steps:**
1. Run existing test suite to ensure no regressions
2. Manual testing with actual agent spawning
3. Verify behavior during agent lifecycle (start, run, complete, cleanup)
4. Test tab switching and closing functionality

## Validation Commands

```bash
# Run unit tests
go test ./internal/tui -v

# Run integration tests
go test ./internal/tui -tags=integration -v

# Manual validation - spawn agents and check TUI
./kiro-krew --config-file .kiro-krew/config.yaml --repo jbrinkman/kiro-krew --label kiro-krew

# Check for race conditions with rapid agent spawning
for i in {1..5}; do ./kiro-krew spawn-agent $i & done; wait
```

## Risk Assessment

**Low Risk Change:**
- Minimal code modification
- Backwards compatible fallback behavior
- Isolated to UI display logic only
- No impact on agent functionality

**Potential Issues:**
- Performance impact of string parsing (minimal)
- Edge cases with malformed agent IDs (handled gracefully)
- Regression in tab functionality (mitigated by comprehensive testing)

## Success Metrics

1. **Functional**: All agent tabs display "Issue N" format
2. **Performance**: No measurable impact on tab rendering performance  
3. **Reliability**: No crashes or errors during agent lifecycle
4. **User Experience**: Clear, consistent tab naming across all scenarios

## Future Considerations

- Consider standardizing agent ID format validation
- Potential for extending this pattern to other UI components
- Monitoring capability for agent lookup failures to identify systemic issues