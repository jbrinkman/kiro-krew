# Design Specification: Fix Agent Tab Log Flickering

**Issue**: #66  
**Closes**: #66

## Problem Analysis

The current issue stems from agent tabs sharing the same `OutputView` logic that displays output from ALL agents instead of filtering to the specific agent ID associated with each tab. The `refreshContent()` method in `output_view.go` processes all agents and displays their combined output, causing visual flickering when multiple agents are running.

### Root Cause

1. **Shared OutputView Logic**: Both main tab and individual agent tabs use the same `OutputView.refreshContent()` method
2. **No Agent Filtering**: The `refreshContent()` method always processes all agents from `manager.List()`
3. **No Agent-Specific State**: Agent tabs don't maintain their own filtered state or scroll position independently

## Solution Approach

Create agent-specific filtering capability in `OutputView` while maintaining existing behavior for the main tab. This requires:

1. **Agent-Specific Output View**: Modify `OutputView` to support agent ID filtering
2. **Independent Scroll State**: Each agent tab maintains its own viewport and scroll position
3. **Preserve Main Tab Behavior**: Keep existing all-agents view in main tab unchanged

## Relevant Files

### Files to Modify

- `internal/tui/output_view.go` - Add agent filtering capability to `refreshContent()`
- `internal/tui/agent_tab.go` - Pass agent ID to OutputView for filtering
- `internal/agent/manager.go` - Add method to get output lines for specific agent (if needed)

### Files Referenced (No Changes Required)

- `internal/tui/tab_manager.go` - Tab management remains unchanged
- `internal/tui/main_tab.go` - Main tab behavior preserved
- `internal/agent/output_capture.go` - Output capture logic unchanged

## Team Orchestration

This is a focused UI filtering fix that can be implemented independently:

1. **No Breaking Changes**: Existing main tab behavior remains identical
2. **No Agent Manager Changes**: Filtering happens at UI level using existing output format
3. **No Test Infrastructure Changes**: Existing tests continue to work

## Step-by-Step Task Breakdown

### Task 1: Add Agent Filtering to OutputView
**Acceptance Criteria:**
- OutputView accepts optional agent ID parameter for filtering
- When agent ID is provided, only show output for that specific agent
- When agent ID is nil/empty, preserve current behavior (show all agents)
- Maintain existing scroll behavior and viewport state

**Implementation Details:**
- Add `agentID` field to `OutputView` struct
- Modify `NewOutputView` constructor to accept optional agent ID
- Update `refreshContent()` to filter by agent ID when specified
- Extract agent filtering logic into separate method for clarity

### Task 2: Create Agent-Specific OutputView in AgentTab
**Acceptance Criteria:**
- Each agent tab creates OutputView with its specific agent ID
- Agent tabs display only their agent's logs without flickering
- Independent scroll positions maintained between tabs
- Agent status indicators still appear in filtered view

**Implementation Details:**
- Modify `NewAgentTab` to pass agent ID to OutputView constructor
- Ensure each agent tab has its own OutputView instance with proper filtering
- Preserve agent header and status indicator in filtered view

### Task 3: Maintain Main Tab Behavior
**Acceptance Criteria:**
- Main tab continues to show all agents (existing behavior)
- No regression in main tab functionality
- All-agents view preserves existing formatting and separators

**Implementation Details:**
- Main tab continues using OutputView without agent ID filter
- No changes required to main tab rendering logic

## Validation Commands

### Manual Testing
```bash
# Start multiple agents to test flickering scenario
kiro-krew watch start

# In separate terminals, create multiple test issues to trigger multiple agents
gh issue create --title "Test Issue 1" --body "Test content 1"
gh issue create --title "Test Issue 2" --body "Test content 2"

# Verify in TUI:
# 1. Main tab shows all agents
# 2. Individual agent tabs show only their specific agent logs
# 3. No flickering when switching between agent tabs
# 4. Each tab maintains independent scroll position
```

### Automated Validation
```bash
# Run existing TUI tests to ensure no regression
go test ./internal/tui/... -v

# Run integration tests
go test ./internal/tui/integration_test.go -v

# Test tab management functionality
go test ./internal/tui/tab_manager_test.go -v
```

### Specific Test Scenarios
1. **Two Agent Scenario**: Verify agent tabs show only their specific logs
2. **Scroll Independence**: Scroll in one agent tab, switch to another, verify position maintained
3. **Agent Status Updates**: Verify status indicators update correctly in filtered views
4. **Main Tab Integrity**: Confirm main tab still shows all agents with proper formatting

## Technical Constraints

1. **Zero Breaking Changes**: Main tab behavior must remain identical
2. **Performance**: Filtering should not impact rendering performance
3. **Memory**: Each agent tab maintains independent viewport state
4. **Thread Safety**: Agent filtering must work safely with concurrent agent updates

## Implementation Notes

- The solution leverages existing agent prefix format: `[agent issue-<number>]`
- No changes needed to OutputCapture or agent output format
- Agent filtering happens at UI rendering level, not data capture level
- Each OutputView instance maintains its own viewport for independent scrolling