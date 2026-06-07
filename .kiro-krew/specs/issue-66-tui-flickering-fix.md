# Design Specification: Fix Agent Tab Log Flickering

**Issue:** #66 - Fix agent tab log flickering - individual tabs show content from other agents

Closes #66

## Problem Analysis

### Root Cause
The flickering issue occurs because all `AgentTab` instances share the same `OutputView` implementation that displays **all agents** in `refreshContent()`. When multiple agents are running:

1. Each agent tab creates an `OutputView` via `NewOutputView(manager, styles)`
2. All `OutputView` instances call the same `refreshContent()` method
3. `refreshContent()` iterates through `manager.List()` and shows **all agents**
4. This causes visual flickering as tabs switch between showing all agents vs. the expected single agent

### Key Technical Findings
- `OutputView.refreshContent()` (lines 88-141) displays all agents from `manager.List()`
- Agent output filtering exists but only for prefixed lines, not agent isolation
- Each `AgentTab` uses the same `OutputView` logic without agent-specific filtering
- Agent prefix format is `[agent issue-N]` where N is the issue number

## Solution Approach

**Core Strategy:** Make `OutputView` agent-aware through constructor injection, enabling per-agent filtering while preserving existing multi-agent display for the main tab.

### Architectural Changes
1. **Agent-Scoped OutputView**: Modify `OutputView` to accept optional agent ID for filtering
2. **Filtered Display Logic**: Update `refreshContent()` to show only specified agent when scoped
3. **Backward Compatibility**: Preserve existing all-agents behavior for main tab
4. **Performance**: Filter at display time, not capture time

## Relevant Files

### Files to Modify

**`internal/tui/output_view.go`** (Primary Fix)
- Add `agentID *string` field to `OutputView` struct  
- Modify `NewOutputView()` constructor to accept optional agent ID
- Update `refreshContent()` method with agent filtering logic
- Add helper function for agent ID extraction from prefixed lines

**`internal/tui/agent_tab.go`** (Integration)
- Update `NewAgentTab()` to pass agent ID to `NewOutputView()`
- Ensure each agent tab gets agent-scoped OutputView instance

### Files Referenced (No Changes)
- `internal/tui/tab_manager.go` - Tab lifecycle management (correct)
- `internal/tui/main_tab.go` - Main tab implementation (preserved)
- `internal/agent/manager.go` - Output capture and agent management (correct)
- `internal/agent/output_capture.go` - Line capturing mechanism (correct)

## Team Orchestration

**Single Component Focus**: This is a TUI-layer fix requiring no changes to:
- Agent management logic
- Output capture mechanism  
- Tab management system
- GitHub integration

**Coordination Requirements**: None - isolated change within TUI display logic.

## Step-by-Step Task Breakdown

### Task 1: Add Agent ID Field to OutputView
**File**: `internal/tui/output_view.go`

**Changes**:
```go
type OutputView struct {
    viewport     viewport.Model
    manager      *agent.Manager  
    styles       *Styles
    width        int
    height       int
    lastUpdate   time.Time
    cachedOutput []string
    agentID      *string  // NEW: nil = show all agents
}
```

**Acceptance Criteria**:
- [ ] `OutputView` struct includes optional `agentID` field
- [ ] Field is pointer to string (nil means show all agents)
- [ ] No breaking changes to existing struct usage

### Task 2: Update OutputView Constructor
**File**: `internal/tui/output_view.go`

**Changes**:
```go
// Updated constructor with optional agent ID
func NewOutputView(manager *agent.Manager, styles *Styles, agentID *string) *OutputView {
    vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(24))
    return &OutputView{
        viewport:   vp,
        manager:    manager,
        styles:     styles,
        lastUpdate: time.Now(),
        agentID:    agentID,  // NEW: store agent filter
    }
}
```

**Acceptance Criteria**:
- [ ] Constructor accepts optional `agentID *string` parameter
- [ ] `agentID` is stored in struct for filtering
- [ ] Backward compatibility maintained for existing callers

### Task 3: Implement Agent Filtering Logic
**File**: `internal/tui/output_view.go`

**New Helper Function**:
```go
// extractIssueNumber extracts issue number from agent prefix format: [agent issue-N]
func extractIssueNumber(line string) (int, bool) {
    // Parse "[agent issue-N]" to extract N
    // Return issue number and whether line is agent-prefixed
}

// matchesAgent checks if a line belongs to the specified agent
func (ov *OutputView) matchesAgent(line string, targetAgentID string) bool {
    // Extract issue number from agent prefix
    // Compare with target agent's issue number
}
```

**Modified refreshContent()**: 
- When `ov.agentID != nil`: Filter agents list to only matching agent
- When `ov.agentID == nil`: Display all agents (existing behavior)
- Filter captured lines by agent prefix matching

**Acceptance Criteria**:
- [ ] Helper functions correctly parse agent prefix format
- [ ] Agent filtering works with `[agent issue-N]` format
- [ ] Performance optimized for real-time filtering
- [ ] Handles edge cases (malformed prefixes, etc.)

### Task 4: Update AgentTab to Use Agent-Scoped OutputView
**File**: `internal/tui/agent_tab.go`

**Changes**:
```go
func NewAgentTab(agentID string, manager *agent.Manager, styles *Styles) *AgentTab {
    return &AgentTab{
        agentID:    agentID,
        outputView: NewOutputView(manager, styles, &agentID),  // NEW: pass agentID
    }
}
```

**Acceptance Criteria**:
- [ ] Each `AgentTab` creates agent-scoped `OutputView`
- [ ] Agent ID properly passed to `OutputView` constructor
- [ ] Each tab maintains independent output state

### Task 5: Preserve Main Tab All-Agents Behavior
**File**: Update main tab creation (likely in `internal/tui/tui.go`)

**Implementation**:
```go
// Main tab uses nil agentID to show all agents
mainTabOutputView := NewOutputView(manager, styles, nil)
```

**Acceptance Criteria**:
- [ ] Main tab continues showing all agents
- [ ] No change to existing multi-agent display logic
- [ ] Status indicators and formatting preserved

### Task 6: Add Agent ID Extraction from AgentTab
**File**: `internal/tui/agent_tab.go`

**Challenge**: The `agentID` field in `AgentTab` is a string, but we need to correlate it with issue numbers for filtering.

**Solution Options**:
1. **Store Issue Number**: Modify `AgentTab` to store issue number directly
2. **Agent ID Mapping**: Create mapping between agent ID and issue number
3. **Parse from Agent ID**: If agent ID format is predictable (e.g., "issue-N")

**Recommended**: Parse issue number from agent ID if format allows, or store issue number during tab creation.

**Acceptance Criteria**:
- [ ] Agent tabs can determine their associated issue number
- [ ] Issue number used for output filtering
- [ ] Robust handling of agent ID formats

## Validation Commands

### Pre-Implementation Testing
```bash
# Reproduce the issue
kiro-krew watch start
# Open multiple agent tabs and observe flickering
```

### Post-Implementation Testing
```bash
# Terminal 1: Start watcher with multiple agents
kiro-krew watch start

# Terminal 2: Verify behavior
# 1. Open agent tab for issue N - should show only issue N logs
# 2. Open agent tab for issue M - should show only issue M logs  
# 3. Switch between tabs - no flickering
# 4. Main tab should show all agents
# 5. Independent scroll positions maintained

# Performance test
# Run with 3+ agents simultaneously
# Verify no performance degradation
```

### Verification Checklist
- [ ] **Primary Fix**: Agent tabs show only their specific agent's output
- [ ] **No Flickering**: Stable content when viewing agent-specific tabs
- [ ] **Tab Independence**: Each tab maintains independent scroll state
- [ ] **Main Tab Preserved**: Main tab continues showing all agents
- [ ] **Performance**: No noticeable slowdown with multiple agents
- [ ] **Error Handling**: Graceful handling of edge cases
- [ ] **Agent Lifecycle**: Proper behavior when agents complete/fail
- [ ] **Multi-Agent**: Works correctly with 2+ simultaneous agents

### Edge Cases to Test
- [ ] Agent tab opened before agent starts logging
- [ ] Agent completes while tab is open
- [ ] Rapid switching between agent tabs
- [ ] Agent with no captured output
- [ ] Malformed agent prefix in captured lines
- [ ] Agent restart scenarios

## Implementation Notes

### Critical Considerations
1. **Agent ID Format**: Determine reliable way to map agent tab ID to issue number
2. **Performance**: Filter efficiently during display refresh cycles
3. **Memory**: No additional memory overhead for output capture
4. **Thread Safety**: Ensure filtering doesn't introduce race conditions

### Backward Compatibility
- Existing `NewOutputView(manager, styles)` calls need update to include `nil` agent ID
- Main tab functionality must remain unchanged
- All current TUI behaviors preserved

### Potential Issues
1. **Agent ID Correlation**: If agent ID string doesn't contain issue number, need alternative mapping
2. **Dynamic Agents**: Handle scenario where agent list changes while tabs open
3. **Empty States**: Ensure proper display when agent has no output yet

This specification provides the concrete technical approach needed to eliminate the TUI flickering issue while maintaining existing functionality and performance characteristics.