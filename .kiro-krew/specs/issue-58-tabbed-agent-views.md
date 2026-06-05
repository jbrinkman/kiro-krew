# Design Specification: Tabbed Agent Views

**Issue:** Closes #58  
**Title:** Implement tabbed views for individual agent output instead of mixed single pane  
**Created:** 2026-06-05  

## Problem Statement

The current agent output toggle feature switches between a single agent view and TUI view, but does not work well with multiple agents. Mixed agent output in one pane creates confusion and makes it difficult to follow individual agent progress.

## Solution Approach

### High-Level Strategy

Transform the existing binary view toggle (console/agent output) into a comprehensive tabbed interface where:

1. **Main TUI Tab**: Permanent, non-closable tab containing the existing console interface
2. **Individual Agent Tabs**: Dynamically created tabs for each active agent with dedicated output views
3. **Tab Navigation**: Intuitive keyboard shortcuts and visual indicators for tab switching
4. **Lifecycle Management**: Agent tabs can be closed without affecting agent execution

### Architectural Decisions

- **Extend Existing Tab Infrastructure**: Build upon the existing `TabManager`, `Tab` interface, and `MainTab` implementations
- **Agent Tab Factory Pattern**: Create `AgentTab` type that wraps individual agent output views
- **Preserve Existing Functionality**: Maintain F2 toggle behavior for backward compatibility
- **Separation of Concerns**: Agent execution remains independent of tab visibility

## Relevant Files

### Files to Create
- `internal/tui/agent_tab.go` - Individual agent tab implementation
- `internal/tui/tab_bar.go` - Visual tab bar rendering and navigation
- `internal/tui/tab_navigation.go` - Keyboard shortcuts and tab switching logic

### Files to Modify
- `internal/tui/tui.go` - Integrate tab bar rendering and agent lifecycle hooks
- `internal/tui/tab_manager.go` - Add agent tab lifecycle management methods
- `internal/tui/commands.go` - Add agent lifecycle event handling
- `internal/tui/styles.go` - Add tab bar styling definitions

### Key Dependencies
- `internal/agent/manager.go` - Agent lifecycle event integration
- `internal/tui/output_view.go` - Per-agent output view adaptation

## Team Orchestration

### Component Responsibilities

1. **Tab Bar Component**: Visual rendering of tab titles and indicators
2. **Agent Tab Component**: Individual agent output view management
3. **Tab Manager**: Central tab lifecycle and navigation coordination
4. **Main TUI**: Integration point for tab rendering and input routing

### Integration Points

- Agent Manager → Tab Manager: Agent creation/destruction events
- Tab Manager → Agent Tabs: Message routing and state updates  
- Tab Bar → Tab Manager: Navigation input handling
- Main TUI → Tab Manager: Unified rendering coordination

## Step-by-Step Task Breakdown

### Phase 1: Core Tab Infrastructure

#### Task 1.1: Create Agent Tab Implementation
**File:** `internal/tui/agent_tab.go`

**Requirements:**
- Implement `Tab` interface for individual agents
- Wrap existing `OutputView` functionality
- Handle agent-specific output filtering and display
- Support scroll state preservation when switching tabs

**Acceptance Criteria:**
- AgentTab implements all Tab interface methods
- Agent output is properly filtered to show only relevant lines
- Scroll position is maintained when switching away and back
- Tab is closable without affecting agent execution

#### Task 1.2: Create Tab Bar Component  
**File:** `internal/tui/tab_bar.go`

**Requirements:**
- Render horizontal tab bar showing all active tabs
- Highlight active tab with distinct styling
- Show closable indicator for agent tabs
- Handle tab title truncation for narrow terminals
- Display tab count when too many tabs to show

**Acceptance Criteria:**
- Tab bar renders correctly in various terminal widths
- Active tab is clearly distinguished visually
- Tab titles are readable and appropriately truncated
- Close indicators (×) are shown for closable tabs only

#### Task 1.3: Implement Tab Navigation Logic
**File:** `internal/tui/tab_navigation.go`  

**Requirements:**
- Handle keyboard shortcuts for tab switching (Ctrl+Tab, Ctrl+Shift+Tab)
- Support numeric shortcuts (Ctrl+1, Ctrl+2, etc.) for direct tab access
- Implement tab closing shortcuts (Ctrl+W)
- Mouse click support for tab selection and closing

**Acceptance Criteria:**
- Keyboard shortcuts work intuitively for tab navigation
- Numeric shortcuts work for tabs 1-9
- Tab closing works only for closable tabs
- Navigation wraps around (last tab → first tab)

### Phase 2: Integration and Lifecycle Management

#### Task 2.1: Extend Tab Manager for Agent Lifecycle
**File:** `internal/tui/tab_manager.go`

**Requirements:**
- Add `CreateAgentTab(agent *Agent)` method
- Add `CloseAgentTab(agentID string)` method  
- Add `GetAgentTab(agentID string)` method
- Handle tab reordering to keep main tab first
- Prevent closing main tab

**Acceptance Criteria:**
- Agent tabs are created automatically when agents start
- Agent tabs can be closed without affecting agent execution
- Main tab always remains first and cannot be closed
- Tab manager correctly tracks agent tab associations

#### Task 2.2: Integrate Tab System in Main TUI
**File:** `internal/tui/tui.go`

**Requirements:**
- Render tab bar above content area
- Route keyboard input to tab navigation when appropriate
- Hook agent lifecycle events to create/remove tabs
- Preserve F2 toggle functionality for backward compatibility
- Update view rendering to use tab manager

**Acceptance Criteria:**
- Tab bar appears above all content
- Agent start/stop events create/remove appropriate tabs
- F2 still toggles between main tab and first agent tab
- All existing functionality continues to work

#### Task 2.3: Add Agent Lifecycle Event Handling
**File:** `internal/tui/commands.go`

**Requirements:**
- Listen for agent start events to create tabs
- Listen for agent completion/failure events to update tab titles
- Handle agent removal to clean up tabs
- Preserve agent tabs when agents complete (until manually closed)

**Acceptance Criteria:**
- Agent tabs appear immediately when agents start
- Tab titles reflect agent status (running/completed/failed)
- Completed agent tabs remain visible until manually closed
- Tab creation/removal doesn't interfere with existing commands

### Phase 3: Visual Design and Polish

#### Task 3.1: Add Tab Bar Styling
**File:** `internal/tui/styles.go`

**Requirements:**
- Define styles for active/inactive tabs
- Add hover styles for mouse interaction
- Create status indicator styles (running/completed/failed)
- Ensure theme compatibility across all themes

**Acceptance Criteria:**
- Tab bar styling is consistent with existing theme system
- Active/inactive states are clearly distinguishable
- Status indicators use appropriate colors
- Styling works across all supported themes

#### Task 3.2: Responsive Tab Bar Layout
**File:** `internal/tui/tab_bar.go` (enhancement)

**Requirements:**
- Handle overflow when too many tabs to display
- Show scroll indicators for hidden tabs
- Implement tab scrolling for navigation
- Graceful degradation for very narrow terminals

**Acceptance Criteria:**
- Tab bar works correctly with 10+ active agents
- Hidden tabs are indicated with scroll arrows
- Tab scrolling allows access to all tabs
- Minimum viable display works in 80-column terminals

### Phase 4: Advanced Features

#### Task 4.1: Tab State Persistence 
**File:** `internal/tui/agent_tab.go` (enhancement)

**Requirements:**
- Preserve scroll position when switching tabs
- Save agent output view state per tab
- Remember which tab was last active
- Handle tab restoration after agent restart

**Acceptance Criteria:**
- Scroll position is maintained when switching between tabs
- Agent output view preferences persist per tab
- Last active tab is restored on restart
- Reopened agent tabs show full output history

#### Task 4.2: Enhanced Tab Indicators
**File:** `internal/tui/tab_bar.go` (enhancement)

**Requirements:**
- Show unread output indicator on background tabs
- Display agent status icons (●, ✓, ✗)
- Show activity indicators for running agents
- Add tab tooltips with full agent information

**Acceptance Criteria:**
- Background tabs show when new output arrives
- Status icons clearly indicate agent state
- Activity animations indicate active agents
- Tooltips provide full context without switching tabs

## Validation Commands

### Unit Testing
```bash
# Run tab manager tests
go test ./internal/tui -run TestTabManager -v

# Run agent tab tests  
go test ./internal/tui -run TestAgentTab -v

# Run tab navigation tests
go test ./internal/tui -run TestTabNavigation -v
```

### Integration Testing
```bash
# Test multi-agent scenario
cd test-project
kiro-krew &
# In TUI: watch start
# Verify multiple agent tabs appear
# Test tab navigation with Ctrl+Tab
# Test closing agent tabs with Ctrl+W

# Test F2 backward compatibility
# Verify F2 still toggles between main and first agent tab
```

### Manual Validation Scenarios

1. **Basic Tab Creation**: Start multiple agents, verify individual tabs appear
2. **Tab Navigation**: Use keyboard shortcuts to switch between tabs
3. **Tab Closing**: Close agent tabs without affecting agent execution  
4. **Main Tab Protection**: Verify main tab cannot be closed
5. **Status Updates**: Verify tab titles update when agents complete/fail
6. **Output Isolation**: Verify each tab shows only relevant agent output
7. **Scroll State**: Verify scroll position preserved when switching tabs
8. **F2 Compatibility**: Verify F2 toggle still works as expected

### Performance Validation
```bash
# Test with many agents
for i in {1..10}; do
  # Create test issues and start agents
  # Verify UI remains responsive
  # Check memory usage stays reasonable
done
```

## Implementation Notes

### Backward Compatibility
- Existing F2 toggle behavior must be preserved
- ViewManager should remain functional during transition
- All existing keyboard shortcuts must continue working

### Performance Considerations
- Tab rendering should be lazy-loaded for better performance
- Agent output should be buffered per-tab to avoid memory bloat
- Tab switching should be instantaneous regardless of agent count

### Error Handling
- Gracefully handle agent crashes/disconnections
- Provide feedback when tab operations fail
- Maintain UI stability when agents start/stop rapidly

### Accessibility
- Tab navigation must work without mouse
- Screen reader compatibility for tab titles and status
- High contrast theme support for tab indicators

## Future Enhancements

The design supports these future improvements:

1. **Tab Persistence**: Save/restore tab layout across sessions
2. **Tab Grouping**: Group related agent tabs together  
3. **Split Views**: Show multiple agent tabs simultaneously
4. **Tab Customization**: User-configurable tab titles and colors
5. **Tab Search**: Quick switching to agent tabs by issue number/title