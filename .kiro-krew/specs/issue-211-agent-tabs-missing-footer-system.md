# Design Specification: Agent Tabs Missing Footer System

**Issue**: #211  
**Title**: Agent Tabs Missing Footer System  
**Closes**: #211

## Problem Statement

Agent tabs currently do not display the two-row footer system established in Issue #195. The footer system is consistently rendered for Main tabs and Planning tabs, but Agent tabs show no footer information at all. This creates an inconsistent user experience across different tab types.

## Current Architecture Analysis

### Footer System (Issue #195)

The two-row footer system was established with the following structure:

**Row 1 (Input Row)**: Command entry area with prompt  
**Row 2 (Status Row)**: Contextual information based on tab type

The footer is managed by `FooterManager` in `internal/tui/footer.go` and provides:
- `RenderFooter(activeTabType TabType)` - Renders both rows based on tab type
- `RenderWithSeparator(activeTabType TabType)` - Adds separator line above footer
- Context-aware rendering based on `TabType` enum

### Current Footer Behavior by Tab Type

1. **Main Tab (`TabTypeMain`)**
   - Shows: `theme: {theme}`
   - Rendered via `renderTabContentWithFooter()` in main TUI View loop

2. **Planning Tab (`TabTypePlanning`)**
   - Shows: `theme: {theme} | ctx: {usage} | model: {model} | 📁 {directory}`
   - Additional status information from ACP connection
   - Rendered via `renderTabContentWithFooter()` in main TUI View loop

3. **Agent Tab (`TabTypeAgent`)** ❌
   - Shows: **Nothing** (no footer rendered)
   - Tab content comes from `OutputView.View()` which only renders agent output
   - **Gap**: Agent tabs bypass footer rendering entirely

### Root Cause

The issue occurs in `internal/tui/tui.go` in the `View()` method:

```go
func (m model) View() tea.View {
    // ...
    var content string
    activeTab := m.tabManager.GetActiveTab()
    if activeTab != nil && activeTab.Type() != TabTypeMain {
        // For non-main tabs, get their content and apply unified rendering with footer
        tabContent := activeTab.View()
        content = m.renderTabContentWithFooter(tabContent, activeTab.Type())
    }
    // ...
}
```

While `renderTabContentWithFooter()` is called for all non-main tabs (including agent tabs), the `FooterManager.renderStatusRow()` method only provides enhanced information for `TabTypePlanning`:

```go
func (fm *FooterManager) renderStatusRow(activeTabType TabType) string {
    baseInfo := fm.renderBaseInfo() // theme: {theme}
    
    if activeTabType == TabTypePlanning {
        // Enhanced planning info...
        return fm.joinStatusInfo(baseInfo, planningInfo)
    }
    
    return baseInfo // Agent tabs only get base info
}
```

**Agent tabs DO receive the footer system**, but they only show the base information (`theme: {theme}`), which may not be visible or useful enough.

## Solution Approach

Enhance the `FooterManager` to provide contextual information for agent tabs, similar to how planning tabs display additional metadata. Agent tabs should display:

1. **Base Information**: Theme (consistent with all tabs)
2. **Agent-Specific Information**:
   - Issue number being processed
   - Agent status (running, completed, failed)
   - Elapsed time since agent started
   - Agent ID for reference

This approach:
- Maintains consistency with the established footer system pattern
- Leverages existing `Agent` struct metadata (ID, IssueNumber, IssueTitle, Status, StartTime)
- Follows the same rendering path as planning tabs
- Requires minimal changes to the architecture

## Relevant Files

### Files to Modify

1. **`internal/tui/footer.go`**
   - Add `renderAgentInfo()` method to format agent-specific footer information
   - Modify `renderStatusRow()` to handle `TabTypeAgent` case
   - Extract agent metadata from active tab and format it appropriately

2. **`internal/tui/agent_tab.go`**
   - Potentially add helper methods to expose agent metadata if needed
   - No structural changes required (footer is rendered at TUI level, not tab level)

### Files Referenced

- **`internal/tui/tui.go`** - Main view rendering loop (already calls `renderTabContentWithFooter`)
- **`internal/agent/manager.go`** - Agent struct definition with metadata (ID, IssueNumber, Status, StartTime)
- **`internal/tui/output_view.go`** - Agent tab content rendering (no changes needed)
- **`internal/tui/tabs.go`** - TabType enum definition (no changes needed)

## Team Orchestration

This implementation can be completed as a single cohesive unit with sequential tasks:

1. **Backend Enhancement** (FooterManager logic)
   - Implement agent footer information extraction and formatting
   - Add agent status rendering logic
   
2. **Integration and Validation**
   - Verify footer renders correctly for agent tabs
   - Ensure no regressions for other tab types
   - Test with multiple agent states (running, completed, failed)

**Dependencies**: Task 2 depends on Task 1 completion.

## Step-by-Step Task Breakdown

### Task 1: Implement Agent Footer Information Rendering

**File**: `internal/tui/footer.go`

**Acceptance Criteria**:
- Add `renderAgentInfo()` method that:
  - Takes the active agent tab as input
  - Extracts agent metadata (issue number, status, elapsed time, agent ID)
  - Formats the information consistently with planning tab style
  - Returns formatted string with agent contextual information
- Modify `renderStatusRow()` to:
  - Detect `TabTypeAgent` case
  - Call `renderAgentInfo()` when active tab is an agent tab
  - Combine base info with agent-specific info using `joinStatusInfo()`
- Implement elapsed time formatting (e.g., "2m 34s", "1h 15m")
- Add appropriate visual indicators for agent status:
  - Running: `● running`
  - Completed: `✓ completed`
  - Failed: `✗ failed`

**Implementation Details**:
```go
// Example footer format:
// theme: dark | issue: #123 | status: ● running | elapsed: 2m 34s | agent: abc123
```

**Dependencies**: None

---

### Task 2: Integrate Agent Metadata Access in Footer Manager

**File**: `internal/tui/footer.go`

**Acceptance Criteria**:
- FooterManager must have access to TabManager to retrieve active agent tab
- `renderAgentInfo()` must safely handle cases where:
  - Agent tab exists but agent is no longer in manager
  - Agent metadata is incomplete
  - Multiple agents are running (show active tab's agent)
- Add nil checks and graceful fallbacks when agent data is unavailable
- Ensure footer renders correctly when switching between tabs

**Implementation Details**:
- FooterManager already has `tabManager *TabManager` field
- Use `tabManager.GetActiveTab()` to get current agent tab
- Cast to `*AgentTab` and retrieve `agentID`
- Use agent manager (via tab's outputView) to get agent metadata
- May require passing agent manager reference to FooterManager

**Dependencies**: None (can run in parallel with Task 1 design)

---

### Task 3: Add Footer Manager Access to Agent Manager

**File**: `internal/tui/tui.go`

**Acceptance Criteria**:
- FooterManager has access to agent.Manager instance
- Pass agent manager reference when constructing FooterManager
- Update NewFooterManager signature if needed
- Verify all FooterManager instantiations are updated

**Implementation Details**:
```go
// In tui.go initialization:
footerManager := NewFooterManager(styles, config, autocompleteInput, tabManager, agentManager)
```

**Dependencies**: Task 1, Task 2

---

### Task 4: Validation and Testing

**Acceptance Criteria**:
- Agent tabs display two-row footer system with separator
- Footer shows contextual information: theme, issue number, status, elapsed time, agent ID
- Footer updates in real-time as agent status changes
- Footer persists when switching between tabs
- No regressions for Main and Planning tab footers
- Footer remains consistent with Issue #195 specification
- Edge cases handled gracefully:
  - Agent no longer exists in manager
  - Agent with missing metadata
  - Rapid tab switching
  - Terminal resizing

**Validation Commands**:
```bash
# Build and run the application
go build ./cmd/kiro-krew

# Start kiro-krew with test configuration
./kiro-krew

# In REPL:
# 1. Start watcher to spawn agent tabs
watch start

# 2. Create a test issue with the configured label
gh issue create --title "Test Issue" --body "Test body" --label "kiro-krew"

# 3. Wait for agent to spawn and verify footer appears on agent tab
# 4. Switch between tabs using '[', ']', or F2 and verify footer persists
# 5. Check footer shows correct agent information
# 6. Wait for agent to complete and verify status updates in footer
```

**Manual Testing Steps**:
1. Launch kiro-krew and start watcher
2. Create labeled issue to spawn an agent
3. Navigate to agent tab (should auto-focus when agent spawns)
4. Verify footer displays:
   - Row 1: Command input area
   - Row 2: `theme: {theme} | issue: #{number} | status: ● running | elapsed: {time} | agent: {id}`
5. Switch to Main tab and back - footer should persist
6. Wait for agent to complete - status should update to `✓ completed`
7. Test with failed agent - status should show `✗ failed`
8. Resize terminal - footer should adapt correctly

**Dependencies**: Task 1, Task 2, Task 3

---

## Implementation Notes

### Agent Metadata Access Pattern

The agent tab has the following data flow:
```
AgentTab -> OutputView -> agent.Manager -> Agent struct
```

To access agent metadata for footer rendering:
```
FooterManager -> TabManager -> AgentTab -> agentID
FooterManager -> agent.Manager -> GetAgent(agentID) -> Agent struct
```

### Status Indicator Consistency

Reuse existing status rendering logic from `output_view.go`:
```go
switch agentItem.Status {
case agent.StatusRunning:
    statusStyle = ov.styles.Success // Green
case agent.StatusCompleted:
    statusStyle = ov.styles.Activity // Blue/Cyan
case agent.StatusFailed:
    statusStyle = ov.styles.Error // Red
}
```

### Elapsed Time Formatting

Implement human-readable time formatting:
- Under 1 minute: "34s"
- Under 1 hour: "2m 34s"
- Over 1 hour: "1h 15m"

### Footer Information Priority

Following Issue #195 pattern, prioritize information display:
1. **Critical**: Issue number and status (always show)
2. **Important**: Elapsed time (helps user understand agent progress)
3. **Reference**: Agent ID (useful for debugging)
4. **Base**: Theme (consistent across all tabs)

Example priority-ordered display:
```
theme: dark | issue: #123 | status: ● running | elapsed: 2m 34s | agent: abc123
```

### Edge Case Handling

1. **Agent not found in manager**: Show generic footer with base theme only
2. **Missing agent metadata**: Gracefully omit unavailable fields
3. **Tab switching during agent lifecycle**: Footer should update reactively based on active tab
4. **Multiple agents**: Show only the active agent tab's information

## Validation Commands

### Build and Run
```bash
# Build the application
go build ./cmd/kiro-krew

# Run with test configuration
./kiro-krew
```

### Functional Testing
```bash
# In kiro-krew REPL:
watch start

# Create test issue (in separate terminal)
gh issue create --title "Test Agent Footer" --body "Testing footer system" --label "kiro-krew"

# Verify agent tab footer displays correctly
# Switch tabs with '[', ']', or F2
# Confirm footer persists and shows correct info
```

### Integration Testing
```bash
# Run existing tests
go test ./internal/tui/... -v

# Run specific footer tests
go test ./internal/tui -run TestFooter -v

# Check for regressions
go test ./... -v
```

### Visual Verification Checklist
- [ ] Agent tab displays two-row footer (separator + input row + status row)
- [ ] Status row shows: theme, issue number, agent status, elapsed time, agent ID
- [ ] Footer updates when agent status changes (running → completed/failed)
- [ ] Footer remains visible when scrolling agent output
- [ ] Footer adapts correctly on terminal resize
- [ ] No visual regressions on Main or Planning tabs
- [ ] Status indicators use consistent colors (green=running, blue=completed, red=failed)

## Success Criteria Summary

1. **Functional Requirements Met**:
   - Agent tabs display the two-row footer system
   - Footer shows appropriate contextual information (issue, status, elapsed time, agent ID)
   - Layout remains consistent with Issue #195 specification

2. **Technical Quality**:
   - Code follows existing footer system patterns
   - No regressions in other tab types
   - Edge cases handled gracefully
   - Performance remains unaffected (footer rendering is lightweight)

3. **User Experience**:
   - Consistent footer across all tab types
   - Real-time updates for agent status and elapsed time
   - Clear visual indicators for agent state
   - Professional and informative footer layout
