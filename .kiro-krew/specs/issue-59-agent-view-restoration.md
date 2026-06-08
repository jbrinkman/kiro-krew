# Design Specification: Agent View Restoration from Status Dialog

**Issue:** #59 - Add ability to reopen agent views from status dialog  
**Closes:** #59

## Solution Approach

Enhance the existing status dialog to provide clickable agent links that reopen closed agent view tabs. This builds upon the tabbed agent views system (issue #58) by adding interactive restoration capabilities to the status overlay. The solution focuses on minimal code changes while providing both keyboard shortcuts and mouse interaction support.

### Key Architectural Decisions

1. **Extend Existing Status Dialog**: Modify `handleStatus()` to add interactive agent entries without changing overlay architecture
2. **Leverage Existing Tab System**: Use existing `TabManager` methods and agent tab creation patterns  
3. **State Preservation**: Restore agent views with current log state using existing `OutputView` functionality
4. **Dual Input Support**: Support both keyboard shortcuts (1-9) and mouse clicks for accessibility
5. **Minimal UI Changes**: Keep existing status dialog layout while adding interactive elements

## Relevant Files

### Files to Modify
- `internal/tui/commands.go` - Add interactive agent entries to `handleStatus()`
- `internal/tui/tui.go` - Add keyboard handling for agent selection in status overlay
- `internal/tui/tab_manager.go` - Add `OpenAgentTab()` method for restoration

### Files Referenced (No Changes)
- `internal/tui/agent_tab.go` - Agent tab implementation (from issue #58)
- `internal/agent/manager.go` - Agent status and lifecycle information
- `internal/tui/styles.go` - Existing styling system

## Team Orchestration

**Single Implementation Unit**: This enhancement builds on completed tabbed agent views (issue #58) and can be implemented as a coordinated set of minimal changes to the status dialog and tab management systems.

**Integration Points**:
- Status Dialog: Add numbered agent entries with visual indicators
- Tab Manager: Simple method to open/switch to agent tabs  
- TUI Input: Basic number key handling in status overlay mode

## Step-by-Step Task Breakdown

### Task 1: Add Interactive Agent List to Status Dialog
**Acceptance Criteria:**
- Show running agents with number prefixes (1-9) in status dialog
- Display click indicators (→) for actionable agents
- Include "[open]" suffix for agents with existing tabs
- Maintain existing status dialog format and information

### Task 2: Implement Agent Tab Opening Method
**Acceptance Criteria:**
- Add `OpenAgentTab(agentID string)` method to TabManager
- Switch to existing tab if agent tab already open
- Create new agent tab if none exists for the agent
- Set new/existing agent tab as active after opening

### Task 3: Add Status Dialog Keyboard Navigation  
**Acceptance Criteria:**
- Handle number keys (1-9) to select agents in status dialog
- Close status dialog and open selected agent tab
- Maintain existing ESC key behavior (close dialog)
- Only handle keys when status dialog is active

### Task 4: Add Mouse Click Support
**Acceptance Criteria:**
- Detect mouse clicks on agent entries in status dialog
- Close dialog and open clicked agent's tab
- Preserve existing mouse handling for other UI elements
- Work consistently with keyboard shortcuts

### Task 5: Update Status Dialog Instructions
**Acceptance Criteria:**
- Add instruction text for agent restoration shortcuts
- Show "1-9: Open agent" and "Click: Select agent"
- Keep instructions concise to fit overlay constraints
- Maintain existing help text for other status features

## Technical Implementation Details

### Status Dialog Agent Entry Format
```
Agents (3 running)
  1. → Issue #45 - Fix console logging (running, 2m15s)
  2. → Issue #52 - Table formatting (running, 45s) [open]  
  3. → Issue #58 - Tabbed views (completed, 5m30s)

Instructions:
  1-9: Open agent    Click: Select    ESC: Close
```

### TabManager OpenAgentTab Method
```go
// OpenAgentTab opens or switches to an agent tab
func (tm *TabManager) OpenAgentTab(agentID string, manager *agent.Manager, styles *Styles) {
    // Check for existing tab
    if tabIndex := tm.FindTabByAgentID(agentID); tabIndex != -1 {
        tm.SetActiveTab(tabIndex)
        return
    }
    
    // Create new agent tab
    agentTab := NewAgentTab(agentID, manager, styles)
    tm.AddTab(agentTab)
    tm.SetActiveTab(len(tm.tabs) - 1)
}
```

### Status Dialog Key Handling
```go
// In tui.go Update method, add to status overlay handling:
case tea.KeyPressMsg:
    if m.activeOverlay == overlayStatus {
        if msg.String() >= "1" && msg.String() <= "9" {
            agentIndex, _ := strconv.Atoi(msg.String())
            if m.openAgentByIndex(agentIndex - 1) {
                m = m.clearOverlay()
            }
            return m, nil
        }
    }
```

### Mouse Click Detection
- Extend existing mouse handling in status overlay mode
- Calculate agent entry positions based on content layout
- Map click coordinates to agent indices
- Reuse agent opening logic for consistency

## Validation Commands

### Basic Functionality Test
```bash
# Start kiro-krew and create multiple agents
kiro-krew watch start

# In TUI, create several agents:
# - Let some run to completion
# - Keep some running  
# - Close some agent tabs while agents continue

# Test status dialog restoration:
status
# Verify: agents show with numbers 1-9
# Verify: running agents show → indicator
# Verify: agents with open tabs show [open] suffix

# Test keyboard shortcuts:
# Press 1-9 to open different agent tabs
# Verify: status dialog closes
# Verify: selected agent tab opens and becomes active
# Verify: existing tabs switch (don't duplicate)

# Test mouse interaction:
# Click on agent entries in status dialog
# Verify: same behavior as keyboard shortcuts
```

### Integration Testing
```bash
# Test with existing features:
# - F2 toggle works after restoration
# - Tab navigation ([ ]) includes restored tabs  
# - Ctrl+W closes restored tabs properly
# - Agent output continues updating in restored tabs

# Test edge cases:
# - Restore agent with no existing tab
# - Restore agent with existing open tab
# - Status dialog with >9 agents (only first 9 get shortcuts)
# - Multiple restorations in sequence
```

### Accessibility Validation  
```bash
# Keyboard-only operation:
# - All restoration functions work without mouse
# - Clear visual indicators for actionable items
# - Instructions visible in status dialog

# Mouse-only operation:  
# - All functions accessible via clicking
# - Click areas clearly defined
# - Consistent with keyboard functionality
```

## Constraints & Implementation Notes

### UI Constraints
- Status dialog size limited by overlay system (60% of terminal size)
- Agent shortcuts limited to 1-9 (first 9 agents only)
- Preserve existing status dialog content and layout
- Instructions must fit within overlay space constraints

### Agent State Handling
- Restored tabs show current agent log position (not from beginning)
- Completed/failed agents show static final output
- Running agents continue live output updates after restoration
- Agent tab restoration works regardless of previous tab closure method

### Input Processing
- Number key handling only active when status overlay is displayed
- Mouse clicks processed through existing overlay mouse handling  
- ESC key maintains existing behavior (close overlay without action)
- All input blocked when other overlays are active

### Tab Integration
- Restored agent tabs integrate seamlessly with existing tab navigation
- F2 toggle works with restored tabs (toggles to first agent tab)
- Tab closing (Ctrl+W) works on restored tabs without affecting agents
- Tab manager maintains correct active tab index after restoration

## Future Enhancements (Out of Scope)

- Persistent tab restoration across TUI sessions
- Agent search/filtering in status dialog
- Bulk agent tab restoration
- Custom shortcut keys beyond 1-9
- Agent tab preview in status dialog