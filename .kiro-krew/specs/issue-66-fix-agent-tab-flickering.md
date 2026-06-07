# Design Specification: Fix Agent Tab Log Filtering

**Issue**: #66 - Fix agent tab log flickering - individual tabs show content from other agents
**Closes**: #66

## Problem Analysis

The current implementation has agent tabs (`AgentTab`) that share the same `OutputView` logic, causing all tabs to display output from all agents instead of filtering to their specific agent ID. The `refreshContent()` method in `output_view.go` shows all agents, leading to flickering between different agent outputs when viewing individual agent tabs.

## Root Cause

1. **Shared OutputView Logic**: `AgentTab` uses the same `OutputView.refreshContent()` method as the main tab
2. **No Agent Filtering**: `refreshContent()` iterates through all agents instead of filtering to a specific agent ID
3. **Cross-contamination**: Each agent tab receives updates from all agents, causing visual flickering

## Solution Approach

Create agent-specific filtering in the `OutputView` to display only the relevant agent's logs when used within an `AgentTab` context.

**Architecture Decision**: Modify `OutputView` to accept an optional agent filter rather than creating separate view classes, maintaining code reuse while enabling targeted filtering.

## Relevant Files

### Files to Modify:
- `internal/tui/output_view.go` - Add agent filtering to `refreshContent()`
- `internal/tui/agent_tab.go` - Pass agent ID filter to OutputView

### Files Referenced (no changes needed):
- `internal/tui/tab_manager.go` - Tab management logic (working correctly)
- `internal/agent/manager.go` - Agent management (working correctly)
- `internal/agent/output_capture.go` - Output capture mechanism (working correctly)

## Team Orchestration

Single developer task - no cross-team coordination required.

## Step-by-Step Task Breakdown

### Task 1: Add Agent Filtering to OutputView
**File**: `internal/tui/output_view.go`

**Acceptance Criteria**:
- [ ] Add `filterAgentID` field to `OutputView` struct
- [ ] Modify `NewOutputView` to accept optional agent ID parameter
- [ ] Update `refreshContent()` to filter agents when `filterAgentID` is set
- [ ] Preserve existing behavior when no filter is applied (main tab functionality)

**Implementation Details**:
```go
// Add to OutputView struct
filterAgentID string // Empty string means show all agents

// Update constructor
func NewOutputView(manager *agent.Manager, styles *Styles, filterAgentID ...string) *OutputView

// Update refreshContent() logic
if ov.filterAgentID != "" {
    // Filter to single agent
} else {
    // Existing logic for all agents
}
```

### Task 2: Update AgentTab to Use Filtered OutputView
**File**: `internal/tui/agent_tab.go`

**Acceptance Criteria**:
- [ ] Modify `NewAgentTab` to pass agent ID to `NewOutputView`
- [ ] Ensure agent-specific output filtering works correctly
- [ ] Maintain existing scroll and navigation behavior

**Implementation Details**:
```go
// In NewAgentTab()
outputView: NewOutputView(manager, styles, agentID),
```

### Task 3: Verification and Testing
**Acceptance Criteria**:
- [ ] Agent tabs display only their specific agent's logs
- [ ] Main tab continues to show all agents
- [ ] No flickering between different agents' logs
- [ ] Independent scroll positions maintained per tab
- [ ] Works correctly with 2+ agents running simultaneously

## Validation Commands

```bash
# Build and test the application
go build -o kiro-krew ./cmd/kiro-krew

# Manual testing scenario:
# 1. Start monitoring multiple issues
./kiro-krew watch start

# 2. Open multiple agent tabs and verify each shows only its agent's logs
# 3. Switch between tabs to confirm no cross-contamination
# 4. Verify main tab still shows all agents

# Run existing tests to ensure no regressions
go test ./internal/tui/...
```

## Risk Assessment

**Low Risk**: Changes are isolated to display logic with clear fallback behavior. Existing tests provide regression protection.

## Success Metrics

- Zero cross-contamination between agent tabs
- Stable content display (no flickering)
- Preserved main tab functionality (shows all agents)
- Independent scroll state per tab