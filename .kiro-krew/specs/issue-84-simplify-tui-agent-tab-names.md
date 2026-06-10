# Design Specification: Simplify TUI Agent Tab Names

**Issue**: Simplify TUI agent tab names to show issue numbers clearly  
**Closes #84**

## Problem Statement

Current TUI agent tab names display as "Agent agent-80-18752" which creates several issues:
- Redundant "Agent agent-" prefix is confusing and wastes space
- Long names get truncated, hiding important information
- Issue number is buried within the agent ID timestamp
- Users cannot quickly identify which GitHub issue each agent is working on

## Solution Approach

Replace the current tab naming from "Agent {agentID}" to "Issue {issueNumber}" format by:
1. Modifying the `AgentTab.Title()` method to access the agent's `IssueNumber` field directly
2. Using the agent manager to look up the agent data by ID
3. Falling back to current behavior if agent lookup fails (for robustness)

This approach leverages the existing `Agent.IssueNumber` field rather than parsing the agent ID, making the solution cleaner and more maintainable.

## Relevant Files

### Files to Modify
- `internal/tui/agent_tab.go` - Update `Title()` method to return "Issue N" format
- `internal/agent/manager.go` - Add `GetAgent(id string)` method for agent lookup

### Files for Reference
- `internal/tui/output_view.go` - Shows how agent manager is used to access agent data
- `internal/tui/tui.go` - Shows how agent tabs are created and managed

## Team Orchestration

This is a focused UI improvement that touches two components:
1. **Agent Manager**: Add simple getter method for agent lookup
2. **TUI Agent Tab**: Modify title generation to use issue number

No coordination between different teams required - this is a self-contained change within the TUI display layer.

## Step-by-Step Task Breakdown

### Task 1: Add Agent Lookup Method to Manager
**File**: `internal/agent/manager.go`
**Description**: Add `GetAgent(id string) *Agent` method to enable agent lookup by ID
**Implementation**:
- Add method that returns agent from `m.agents[id]` map with appropriate locking
- Return `nil` if agent not found
**Acceptance Criteria**:
- Method safely accesses agents map with read lock
- Returns correct agent for valid IDs
- Returns `nil` for non-existent IDs
- No impact on existing functionality

### Task 2: Update Agent Tab Title Method
**File**: `internal/tui/agent_tab.go` 
**Description**: Modify `Title()` method to use agent's issue number
**Implementation**:
- Access agent manager through output view to get agent data
- Extract issue number from agent struct
- Format as "Issue {number}"
- Fallback to current behavior if agent lookup fails
**Acceptance Criteria**:
- Tab titles display as "Issue 80" instead of "Agent agent-80-18752"
- Handles case when agent is no longer found gracefully
- No changes to tab functionality beyond display
- Maintains backward compatibility

### Task 3: Verify Integration
**Description**: Ensure the changes work correctly in the full TUI context
**Acceptance Criteria**:
- All agent tabs show "Issue N" format in TUI
- Tab switching still works correctly
- Tab close functionality unchanged
- No performance impact on tab operations

## Validation Commands

```bash
# Build and run the application to test TUI
go build -o kiro-krew ./cmd/kiro-krew
./kiro-krew watch start

# In another terminal, create test issues to verify tab naming
# (Exact commands depend on your GitHub setup)

# Verify no regressions in existing functionality
go test ./internal/tui/...
go test ./internal/agent/...
```

## Implementation Notes

- **Minimal Change Principle**: Only modify what's necessary for the display change
- **Robustness**: Include fallback behavior to prevent crashes if agent lookup fails
- **Performance**: Agent lookup is O(1) map access, no performance concerns
- **Thread Safety**: Use existing locking patterns in agent manager
- **Testing**: Focus on edge cases like agent cleanup during tab access

## Risk Assessment

**Low Risk**: This is purely a display change with no impact on:
- Agent spawning/management logic
- Tab functionality (switching, closing)
- Agent process lifecycle
- Data persistence or state

**Mitigation**: Fallback to current behavior ensures no crashes if agent data becomes unavailable.