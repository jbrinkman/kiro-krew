# Design Specification: Agent View Reopening from Status Dialog

**Issue:** #59 - Add ability to reopen agent views from status dialog  
**Closes:** #59

## Solution Approach

This feature extends the existing status dialog to provide direct access for reopening closed agent views. Building upon the tabbed agent views system implemented in issue #58, this enhancement transforms the status dialog into an interactive interface that allows users to restore visibility into running agents whose view tabs have been closed.

### High-Level Strategy

1. **Interactive Status Dialog**: Transform the current read-only status dialog into an interactive menu for agent view restoration
2. **Multi-Modal Access**: Support both keyboard shortcuts (number keys 1-9) and mouse clicks for selecting agents
3. **Seamless Integration**: Maintain existing status dialog workflow while adding restoration capabilities
4. **State Preservation**: Restore agent views from their current log state, not from the beginning

### Key Architectural Decisions

- **Overlay Enhancement**: Extend existing `overlayStatus` functionality rather than creating new overlay types
- **Tab Manager Integration**: Leverage existing `TabManager` methods for agent tab creation and navigation
- **Agent State Tracking**: Use existing agent manager data to identify running vs stopped agents
- **Modal Workflow**: Status dialog closes automatically when an agent is selected, maintaining existing UX patterns

## Relevant Files

### Files to Modify

#### `internal/tui/commands.go`
- **Primary Changes**: Enhance `handleStatus()` to add interactive agent selection
- **Purpose**: Transform status overlay from display-only to interactive selection interface
- **Key Functions**: Add agent enumeration, keyboard shortcut hints, click handling

#### `internal/tui/tui.go`  
- **Primary Changes**: Extend key handling in `Update()` for status overlay interactions
- **Purpose**: Process number key presses (1-9) when status overlay is active
- **Key Functions**: Route agent selection to tab restoration logic

#### `internal/tui/tab_manager.go`
- **Enhancement**: Add method to check if agent tab already exists
- **Purpose**: Prevent duplicate agent tabs and support view restoration
- **Key Functions**: `HasAgentTab(agentID string) bool`, `RestoreOrFocusAgentTab(agentID string)`

### Files Referenced (No Changes)

#### `internal/tui/agent_tab.go`
- **Usage**: Agent tab creation patterns for restoration logic
- **Context**: Existing NewAgentTab constructor for recreating closed tabs

#### `internal/agent/manager.go`
- **Usage**: Agent state information for status display
- **Context**: List() method provides running agent details

#### `internal/tui/output_view.go`
- **Usage**: Agent output rendering and state management
- **Context**: OutputView restoration from current agent log state

## Team Orchestration

**Single Developer Implementation**: This enhancement can be implemented as a coordinated change across three core TUI files. The changes are isolated to the user interface layer and don't require modifications to agent execution, file watching, or external integrations.

**Integration Dependencies**:
- **Tab System**: Must work seamlessly with existing tabbed agent views (issue #58)
- **Agent Management**: Relies on existing agent lifecycle and status tracking
- **Overlay System**: Extends current modal overlay functionality

**Testing Coordination**: Changes can be tested as a unit since they form a complete user workflow from status display through agent selection to tab restoration.

## Step-by-Step Task Breakdown

### Task 1: Enhance Status Dialog Interactivity
**Acceptance Criteria:**
- Status dialog displays numbered list of running agents (1-9)
- Each agent entry shows issue number, title, status, and elapsed time
- Running agents are clearly distinguished from stopped agents
- Keyboard shortcuts (1-9) are indicated in the interface
- Support mouse clicks on agent entries

**Implementation Details:**
- Modify `handleStatus()` in `commands.go` to format agents with number prefixes
- Add instructional text explaining number key navigation
- Maintain existing status information layout while adding interactivity

### Task 2: Implement Agent Selection Logic
**Acceptance Criteria:**
- Number keys 1-9 select corresponding agents when status overlay is active
- Mouse clicks on agent entries trigger agent selection
- Invalid selections (non-existent agents) are handled gracefully
- Status overlay closes immediately upon valid agent selection

**Implementation Details:**
- Extend `Update()` method in `tui.go` to handle number key presses during status overlay
- Add mouse click handling for agent selection areas
- Route valid selections to tab restoration logic

### Task 3: Implement Tab Restoration Logic
**Acceptance Criteria:**
- Create new agent tab if one doesn't exist for the selected agent
- Focus existing agent tab if it's already open
- Agent view displays current log state, not historical replay
- Tab switching is smooth and immediate after selection

**Implementation Details:**
- Add `RestoreOrFocusAgentTab()` method to `TabManager`
- Check for existing agent tab before creating new one
- Use existing `NewAgentTab()` constructor for tab creation
- Set focus to restored/found agent tab

### Task 4: Agent State Validation
**Acceptance Criteria:**
- Only display running agents in the interactive list
- Handle edge cases where agents stop between status display and selection
- Provide user feedback if selected agent is no longer running
- Maintain correct agent numbering when agents start/stop

**Implementation Details:**
- Filter agent list to only include running agents
- Validate agent status before tab restoration
- Display appropriate error messages for invalid selections
- Refresh agent list each time status dialog is opened

### Task 5: User Experience Polish
**Acceptance Criteria:**
- Clear visual indication of interactive vs display-only elements
- Consistent styling with existing TUI theme system
- Helpful instructional text for discovery of new functionality
- Keyboard accessibility for all interactions

**Implementation Details:**
- Use existing styles for consistent theming
- Add clear separators between display and interactive sections
- Include concise usage instructions in status overlay
- Ensure all interactions work without mouse dependency

## Technical Implementation Details

### Status Dialog Enhancement Structure
```go
// Enhanced status content format:
// "Running Agents (press number to open view):"
// "1. Issue #42: Fix bug in authentication (running, 2m30s)"
// "2. Issue #58: Add tabbed views (running, 45s)"
// ""
// "Stopped Agents:"
// "   Issue #39: Completed (completed, 15m ago)"
```

### Key Binding Extensions
- **Number Keys (1-9)**: Select corresponding running agent from status dialog
- **Mouse Clicks**: Click on numbered agent entries in status dialog
- **ESC**: Close status dialog (existing behavior)

### Agent Selection Flow
1. User opens status dialog (existing `status` command)
2. Dialog shows numbered list of running agents (enhanced display)
3. User presses number key or clicks agent entry
4. Status dialog closes and agent tab opens/focuses
5. Agent view shows current log state

### Error Handling
- **Invalid Agent Number**: Silent ignore (no error message needed)
- **Agent Stopped**: Display "Agent no longer running" message
- **Tab Creation Failure**: Display error and return to status dialog

## Validation Commands

### Functional Testing
```bash
# Start kiro-krew with multiple agents
kiro-krew watch start

# Create several agents, then close some agent tabs
# Test status dialog interactivity:

# 1. Open status dialog
status

# 2. Test number key selection (1-9)
# Verify each number opens corresponding agent view
# Verify out-of-range numbers are ignored

# 3. Test mouse click selection
# Click on different agent entries
# Verify status dialog closes and agent tab opens

# 4. Test edge cases
# Select agent that stops between status display and selection
# Try to select from empty running agents list
```

### Integration Testing
```bash
# Test with existing tab system:
# - Agent already open: Should focus existing tab
# - Agent tab closed: Should recreate tab with current log state
# - Multiple agents: Should handle selection of any agent
# - Agent completion: Should handle stopped agents appropriately

# Test with overlay system:
# - Other overlays: Should not interfere with number key handling
# - ESC behavior: Should still close status dialog normally
# - Multiple status opens: Should refresh agent list each time
```

### User Experience Testing
```bash
# Verify discoverability:
# - Status dialog clearly indicates interactive capability
# - Number shortcuts are obvious to users
# - Mouse interaction works intuitively

# Verify accessibility:
# - All functionality available via keyboard
# - Clear visual distinction between interactive/static content
# - Consistent with existing TUI navigation patterns
```

## Future Enhancements (Out of Scope)

- **Extended Agent Selection**: Support for more than 9 running agents through pagination
- **Quick Agent Actions**: Stop/restart agents directly from status dialog  
- **Agent Grouping**: Organize agents by status or issue type in status display
- **Agent Search**: Filter agents in status dialog by issue number or title
- **Persistent Agent Tabs**: Remember closed agent tabs across TUI sessions
- **Agent Tab Ordering**: Allow reordering of agent tabs through status interface