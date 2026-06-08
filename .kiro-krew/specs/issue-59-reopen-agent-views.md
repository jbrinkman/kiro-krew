# Design Specification: Add ability to reopen agent views from status dialog

**Issue**: #59  
**Closes**: #59

## Solution Approach

This feature enables users to reopen closed agent view tabs directly from the status dialog. The solution builds on the existing tabbed agent views system (issue #58) by extending the status overlay with clickable agent entries that restore agent views when selected.

### High-Level Strategy

1. **Extend Status Dialog**: Modify the status overlay to include clickable agent entries for running agents
2. **Agent View Restoration**: Implement logic to reopen agent tabs and restore their current log state
3. **Navigation Integration**: Seamlessly integrate with existing tab navigation and status dialog workflow
4. **State Management**: Ensure agent tabs restore from current log position, not replay entire history

### Architectural Decisions

- **Modal Interaction**: Status dialog remains modal - selecting an agent closes the dialog and opens the tab
- **State Restoration**: Agent views restore from current log buffer state maintained by OutputCapture
- **Tab Lifecycle**: Reuse existing TabManager infrastructure for consistent tab behavior
- **Keyboard + Mouse Support**: Support both mouse clicks and keyboard shortcuts for accessibility

## Relevant Files

### Files to Modify

- **`internal/tui/commands.go`**
  - Modify `handleStatus()` to include clickable agent links
  - Add agent view restoration logic
  - Handle keyboard shortcuts in status dialog

- **`internal/tui/tui.go`**
  - Extend overlay handling to support interactive elements
  - Add mouse/keyboard event routing for status overlay
  - Integrate agent tab restoration with existing tab system

- **`internal/tui/tab_manager.go`**
  - Add method to check if agent tab already exists: `HasAgentTab(agentID string) bool`  
  - Add method to restore/reopen agent tab: `RestoreAgentTab(agentID string) bool`
  - Ensure consistent tab switching behavior

### Files Referenced

- **`internal/tui/agent_tab.go`** - Agent tab implementation (already supports restoration)
- **`internal/tui/output_view.go`** - Output view with current state restoration
- **`internal/agent/manager.go`** - Agent status and lifecycle management
- **`internal/agent/output_capture.go`** - Log buffer management for state restoration

## Team Orchestration

### Component Dependencies

1. **Status Dialog Enhancement** (commands.go)
   - Depends on: Existing agent listing and status overlay infrastructure
   - Provides: Interactive agent selection interface

2. **Event Handling** (tui.go) 
   - Depends on: Status dialog agent list format
   - Provides: Mouse/keyboard event routing to appropriate handlers

3. **Tab Management Integration** (tab_manager.go)
   - Depends on: Event handling for agent selection
   - Provides: Tab restoration and navigation coordination

### Integration Points

- **Agent Manager**: Read-only access to running agents list
- **Output Capture**: Access to current log buffer state for restoration
- **Tab System**: Reuse existing tab creation, navigation, and lifecycle management
- **Overlay System**: Extend existing modal overlay handling for interactivity

## Step-by-Step Task Breakdown

### Phase 1: Status Dialog Enhancement

**Task 1.1: Modify status dialog content generation**
- **File**: `internal/tui/commands.go`
- **Changes**:
  - Add running agents section with clickable indicators 
  - Include keyboard shortcut hints (e.g., "Press 1-9 to open agent tab")
  - Format agent entries to show clickability (e.g., numbered list or action indicators)
- **Acceptance Criteria**:
  - Status dialog shows running agents with clear selection indicators
  - Each agent entry displays issue number, title, and selection method
  - Dialog includes help text explaining how to select agents
  - Non-running agents are visually distinct and non-selectable

**Task 1.2: Add agent selection data structure**
- **File**: `internal/tui/commands.go`
- **Changes**:
  - Create agent index mapping for keyboard shortcuts (1-9, a-z, etc.)
  - Store agent-to-shortcut mapping in status overlay context
  - Handle overflow when more agents than available shortcuts
- **Acceptance Criteria**:
  - Keyboard shortcuts mapped consistently to running agents
  - Support up to reasonable number of concurrent agents (35: 1-9, a-z)
  - Clear indication when more agents exist than shortcuts available

### Phase 2: Interactive Event Handling

**Task 2.1: Extend overlay input handling** 
- **File**: `internal/tui/tui.go`
- **Changes**:
  - Modify `tea.KeyPressMsg` handling in `Update()` to route status overlay keystrokes
  - Add logic to detect agent selection keystrokes (1-9, a-z) when status overlay is active
  - Implement mouse click detection for agent entries in status overlay
- **Acceptance Criteria**:
  - Keyboard shortcuts (1-9, a-z) work within status dialog
  - Mouse clicks on agent entries trigger selection
  - ESC continues to close dialog without action
  - Invalid shortcuts are ignored gracefully

**Task 2.2: Agent selection event handling**
- **File**: `internal/tui/tui.go`
- **Changes**:
  - Add helper function to map shortcut to agent ID
  - Implement agent tab restoration workflow
  - Close status dialog after successful agent selection
- **Acceptance Criteria**:
  - Agent selection closes status dialog immediately
  - Selected agent tab opens and becomes active
  - Error handling for invalid or failed selections
  - Graceful fallback if agent no longer running

### Phase 3: Tab Management Integration

**Task 3.1: Add tab existence checking**
- **File**: `internal/tui/tab_manager.go` 
- **Changes**:
  - Implement `HasAgentTab(agentID string) bool` method
  - Reuse existing `FindTabByAgentID()` logic
- **Acceptance Criteria**:
  - Method correctly identifies existing agent tabs
  - Returns false for non-existent or closed tabs
  - Consistent with existing tab ID conventions

**Task 3.2: Implement tab restoration**
- **File**: `internal/tui/tab_manager.go`
- **Changes**:
  - Implement `RestoreAgentTab(agentID string) bool` method 
  - Create new agent tab if doesn't exist, or switch to existing tab
  - Integrate with existing tab creation and activation logic
- **Acceptance Criteria**:
  - Creates new tab when agent tab doesn't exist
  - Switches to existing tab when agent tab already open
  - New tabs restore from current agent log state
  - Tab switching works consistently with existing navigation

**Task 3.3: Agent view state restoration**
- **File**: Integration between tab_manager.go and output_view.go
- **Changes**:
  - Ensure OutputView restoration works from current log buffer
  - Verify agent tab shows current state, not full history replay
- **Acceptance Criteria**:
  - Reopened agent views show current log state
  - No unnecessary log replay or duplication
  - Scrolling and navigation work normally in restored views
  - Output continues updating in real-time after restoration

### Phase 4: User Experience Enhancements

**Task 4.1: Status dialog visual improvements**
- **File**: `internal/tui/commands.go` + `internal/tui/styles.go`
- **Changes**:
  - Add visual styling for clickable agent entries
  - Include clear instructions and shortcuts in dialog
  - Improve layout for better usability
- **Acceptance Criteria**:
  - Agent entries visually distinct and clearly clickable
  - Keyboard shortcuts prominently displayed
  - Dialog remains readable with multiple agents
  - Consistent with existing TUI styling

**Task 4.2: Error handling and edge cases**
- **File**: `internal/tui/tui.go`, `internal/tui/commands.go`
- **Changes**:
  - Handle case where agent stops while dialog is open
  - Graceful handling of agent tab creation failures
  - User feedback for successful/failed operations
- **Acceptance Criteria**:
  - Appropriate error messages for edge cases
  - Dialog refreshes if agent states change
  - No crashes from race conditions or invalid states
  - Clear user feedback for all operations

## Validation Commands

### Unit Tests
```bash
cd internal/tui
go test -v -run TestTabManager
go test -v -run TestAgentTab
```

### Integration Tests  
```bash
cd internal/tui
go test -v -run TestIntegration
```

### Manual Testing Scenarios

1. **Basic Agent Reopening**:
   ```bash
   # Start kiro-krew with agent running
   kiro-krew
   # Close agent tab with Ctrl+W
   # Open status with 'status' command  
   # Select agent with keyboard shortcut
   # Verify agent tab reopens with current state
   ```

2. **Multiple Agent Handling**:
   ```bash
   # Start multiple agents on different issues
   # Close various agent tabs
   # Verify status dialog shows all running agents
   # Test reopening different agents via shortcuts
   # Verify each shows correct current state
   ```

3. **Edge Case Testing**:
   ```bash
   # Test with no running agents
   # Test with more agents than available shortcuts
   # Test agent stopping while status dialog open
   # Test rapid open/close of status dialog
   ```

4. **Accessibility Testing**:
   ```bash
   # Test all keyboard shortcuts work as documented
   # Test mouse clicks on agent entries (if terminal supports)
   # Test with various terminal sizes and color themes
   # Verify screen readers can access dialog content
   ```

### Performance Validation
- Status dialog should open/close instantly even with 10+ agents
- Agent tab restoration should complete within 100ms
- No memory leaks from repeated dialog open/close cycles
- No impact on existing agent performance or TUI responsiveness

### Integration Validation
- Feature works seamlessly with existing F2 toggle functionality  
- Compatible with existing tab navigation ([ ] keys, Ctrl+W)
- Works correctly with planning mode switching
- Maintains compatibility with all existing TUI commands and hotkeys
