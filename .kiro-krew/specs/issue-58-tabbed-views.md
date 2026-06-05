# Design Specification: Tabbed Views for Individual Agent Output

**Closes #58**

## Overview

Transform the current single-pane agent output view into a tabbed interface where each active agent gets its own dedicated tab, alongside a permanent main TUI tab that cannot be closed. This enhancement improves multi-agent workflows by eliminating output mixing and providing clear separation between agent activities.

## Solution Approach

### High-Level Strategy

1. **Extend Existing Tab System**: Build upon the current partial tab implementation in `internal/tui/` with full tab bar rendering, navigation, and agent tab lifecycle management.

2. **Agent Tab Lifecycle**: Create agent tabs dynamically when agents start, maintain them during execution, and allow closure without affecting agent processes.

3. **Preserve Existing UX**: Maintain backward compatibility with F2 toggle functionality while adding new tab-specific navigation.

4. **Decoupled Architecture**: Agent tab closure affects only view rendering, not agent execution or state.

### Architectural Decisions

- **Tab Bar Rendering**: Implement a tab bar at the top of the TUI showing all available tabs
- **Dynamic Tab Management**: Agent tabs are created/destroyed based on agent lifecycle events
- **Content Isolation**: Each agent tab renders only its own output, eliminating mixed streams
- **State Preservation**: Agent output is preserved in memory buffers for potential tab reopening
- **Navigation Enhancement**: Support both keyboard shortcuts and mouse interaction for tab switching

## Relevant Files

### Files to Modify

1. **`internal/tui/tui.go`**
   - Integrate tab bar rendering into main view
   - Add tab navigation key handling (Ctrl+1-9, Ctrl+Tab, Ctrl+Shift+Tab)
   - Connect agent lifecycle events to tab management
   - Update view rendering to use active tab content

2. **`internal/tui/tab_manager.go`**
   - Add tab bar rendering methods
   - Implement keyboard navigation logic
   - Add agent tab creation/removal methods
   - Enhance tab switching with proper bounds checking

3. **`internal/tui/tabs.go`**
   - Add tab bar styling interface
   - Extend tab interface with additional metadata

### Files to Create

4. **`internal/tui/agent_tab.go`**
   - Implement AgentTab struct with dedicated agent output rendering
   - Handle agent-specific scrolling and navigation
   - Manage agent output buffering and display
   - Implement tab title generation with agent identification

5. **`internal/tui/tab_bar.go`**
   - Render horizontal tab bar with title, close buttons, and active indicators
   - Handle tab width calculation and truncation for narrow terminals
   - Implement visual styling for active/inactive tabs
   - Support keyboard navigation indicators

### Files Referenced

- **`internal/tui/main_tab.go`**: Already implements non-closable main tab
- **`internal/tui/output_view.go`**: Existing agent output logic to be adapted for individual tabs
- **`internal/tui/styles.go`**: Style definitions for tab bar elements
- **`internal/agent/manager.go`**: Agent lifecycle events for tab synchronization

## Team Orchestration

### Component Dependencies

1. **Tab Manager ← Agent Manager**: Tab manager listens to agent lifecycle events
2. **Agent Tabs ← Output Capture**: Individual tabs consume agent-specific output streams  
3. **Main TUI ← Tab Bar**: Main TUI integrates tab bar rendering at top of interface
4. **Navigation ← Key Handler**: Unified key handling for tab switching and agent-specific navigation

### Integration Points

- **Agent Start Event**: Automatically create new agent tab
- **Agent Stop Event**: Preserve tab but update status indicator
- **Output Capture**: Route agent output to appropriate tab buffer
- **F2 Compatibility**: Maintain existing toggle behavior between main and first agent tab

## Step-by-Step Task Breakdown

### Phase 1: Tab Bar Infrastructure

**Task 1.1: Implement Tab Bar Rendering**
- Create `internal/tui/tab_bar.go` with horizontal tab layout
- Add tab title rendering with width constraints
- Implement active/inactive tab visual styling
- Handle tab overflow with scrolling or truncation
- **Acceptance Criteria**: Tab bar displays at top with proper styling

**Task 1.2: Enhance Tab Manager**
- Add `RenderTabBar()` method to TabManager
- Implement tab navigation by index (Ctrl+1-9)
- Add next/previous tab navigation (Ctrl+Tab/Ctrl+Shift+Tab)
- Add tab finding by agent ID for lifecycle management
- **Acceptance Criteria**: Tab navigation works with keyboard shortcuts

### Phase 2: Agent Tab Implementation

**Task 2.1: Create Agent Tab Component**
- Implement `internal/tui/agent_tab.go` with Tab interface
- Create agent-specific output rendering from OutputCapture
- Implement scrollable viewport for agent output
- Add tab title generation with agent ID and issue number
- **Acceptance Criteria**: Agent tabs display isolated agent output

**Task 2.2: Agent Lifecycle Integration**
- Connect agent start events to tab creation
- Implement agent output routing to specific tabs
- Handle agent completion with status indicators
- Preserve agent output buffers when tabs are closed
- **Acceptance Criteria**: Tabs automatically created/updated with agent state

### Phase 3: Main TUI Integration

**Task 3.1: Integrate Tab Bar in Main View**
- Modify `tui.go` `View()` method to render tab bar at top
- Adjust content area height to accommodate tab bar
- Update window resize handling for tab bar space
- Maintain overlay system compatibility
- **Acceptance Criteria**: Tab bar appears at top, content area properly sized

**Task 3.2: Enhanced Navigation**
- Add tab-specific key handling in main Update loop
- Implement tab closing with 'x' key or close button  
- Maintain F2 toggle compatibility for backward compatibility
- Add tab navigation status indicators
- **Acceptance Criteria**: All navigation methods work correctly

### Phase 4: Polish and Edge Cases

**Task 4.1: Tab Management Edge Cases**
- Handle empty tab state (no agents running)
- Implement proper tab cleanup on exit
- Handle rapid agent creation/destruction
- Add tab title truncation for narrow terminals
- **Acceptance Criteria**: Robust behavior under all conditions

**Task 4.2: Visual Polish**
- Add tab close buttons (x) for closable agent tabs
- Implement hover states for mouse interaction
- Add subtle animations for tab transitions
- Ensure accessibility with proper contrast and indicators
- **Acceptance Criteria**: Professional, polished visual appearance

## Technical Implementation Details

### Tab Bar Layout
```
┌─Main TUI─┬─Agent 42─[x]─┬─Agent 58─[x]─┐
│          │              │              │
└──────────┴──────────────┴──────────────┘
```

### Agent Tab Identification
- **Tab ID**: `agent-{issue-number}` (e.g., `agent-42`)
- **Tab Title**: `Agent {issue-number}` or truncated issue title
- **Close Button**: `[x]` for agent tabs only

### Key Bindings
- **Ctrl+1-9**: Switch to tab by position
- **Ctrl+Tab**: Next tab
- **Ctrl+Shift+Tab**: Previous tab  
- **F2**: Toggle between main and first agent tab (existing)
- **x**: Close active tab (if closable)

### Memory Management
- Agent output buffers limited to configurable size (default 1000 lines)
- Tab cleanup on agent termination removes tab but preserves recent output
- Tab reopening capability reserved for future enhancement

## Validation Commands

### Unit Tests
```bash
go test ./internal/tui/... -v
```

### Integration Tests
```bash
# Test tab creation with mock agents
go test ./internal/tui -run TestTabManagerAgentLifecycle

# Test tab bar rendering
go test ./internal/tui -run TestTabBarRendering

# Test navigation functionality
go test ./internal/tui -run TestTabNavigation
```

### Manual Testing Scenarios

1. **Basic Tab Functionality**
   ```bash
   kiro-krew
   # In TUI: watch start
   # Verify: Main tab + new agent tabs appear
   # Test: Switch between tabs with Ctrl+1, Ctrl+2
   ```

2. **Multi-Agent Workflow**
   ```bash
   # Start multiple agents simultaneously
   # Verify: Each gets dedicated tab
   # Test: Output isolation between agent tabs
   ```

3. **Tab Lifecycle Management**
   ```bash
   # Close agent tab with 'x'
   # Verify: Agent continues running
   # Verify: Output no longer mixed in main view
   ```

4. **Edge Cases**
   ```bash
   # Test narrow terminal (< 80 chars)
   # Test many agents (> 9 tabs)
   # Test rapid agent start/stop cycles
   ```

### Performance Verification
- Memory usage stable with multiple agent tabs
- No output lag when switching between tabs
- Tab bar rendering responsive on terminal resize

## Success Metrics

- **Functional**: All acceptance criteria met for each task
- **Compatibility**: Existing F2 toggle functionality preserved  
- **Performance**: No noticeable lag with up to 10 concurrent agent tabs
- **Usability**: Clear visual separation between agent outputs
- **Stability**: No crashes or memory leaks during tab operations

## Future Considerations

- **Tab Persistence**: Save/restore tab states between TUI sessions
- **Tab Reordering**: Drag-and-drop or keyboard-based tab reordering
- **Tab Grouping**: Group related agent tabs by project or issue type
- **Enhanced Status**: Show agent progress indicators in tab titles