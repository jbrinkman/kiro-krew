# Fix Planning Mode UI Layout - Tab Headers Misaligned When Planning Tab Active

**Issue**: #204  
**Closes**: #204

## Problem Summary

The planning mode has broken UI layout when a planning tab is active and terminal width > 100 characters. The root cause is conditional status bar rendering in the planning tab's View() method that disrupts the overall TUI layout structure.

### Current Problematic Flow

1. Main TUI View() method calls `tabHeaders := m.tabManager.RenderTabHeaders(m.width, m.styles)`
2. For planning tabs, it calls `content = activeTab.View()` 
3. Planning tab's View() method conditionally adds a status bar at the top when `pt.width > 100`
4. Main TUI combines them with `content = tabHeaders + "\n" + content`
5. Result: Tab headers on line 1, planning status bar on line 2, planning content below

### Expected Layout vs Actual Layout

**Expected (Wide Terminal)**:
```
[Main] [Planning Tab ×] [Agent Tab]    Context: 450/500 | Model: claude-sonnet-4 | State: Processing
Message area content...
────────────────────────
> Input area
```

**Actual (Wide Terminal)**:
```
[Main] [Planning Tab ×] [Agent Tab]
Context: 450/500 | Model: claude-sonnet-4 | State: Processing  
Message area content...
────────────────────────
> Input area
```

## Root Cause Analysis

The issue lies in the architectural mismatch between:

1. **Tab Header System**: Assumes tab content starts immediately after headers
2. **Planning Tab Status Bar**: Conditionally adds content at the top of its view that should be integrated with the header line

The planning tab's conditional status bar disrupts the layout contract established by the main TUI's tab system.

## Solution Approach

### Strategy: Header Integration Pattern

Instead of having planning tabs render status information within their content area, integrate status information into the tab header line. This maintains consistent layout while preserving responsive design.

### Key Design Decisions

1. **Single Header Line Principle**: All header-level information (tabs + status) must render on the same line
2. **Responsive Layout Preservation**: Maintain width-based status bar behavior (only show on wide terminals)
3. **Tab System Integration**: Use existing tab system architecture rather than creating special cases
4. **Clean Separation**: Planning tab content focuses purely on message/input areas

## Relevant Files

### Primary Implementation Files
- `internal/tui/planning_tab.go` - Remove conditional status bar from View() method
- `internal/tui/tui.go` - Modify View() method to handle planning status integration
- `internal/tui/tab_manager.go` - Enhance RenderTabHeaders to support status integration

### Supporting Files
- `internal/tui/styles.go` - May need style adjustments for integrated status display

## Team Orchestration

This is a UI layout fix that can be implemented as a single coordinated change:

### Task Dependencies
- **Task 1** and **Task 2** have no dependencies and can run in parallel
- **Task 3** depends on both Task 1 and Task 2 completion
- **Task 4** depends on Task 3 for validation

## Step-by-Step Task Breakdown

### Task 1: Refactor Planning Tab View Method
**Acceptance Criteria**:
- Remove conditional status bar rendering from planning_tab.go View() method
- Extract status information into a separate method `GetStatusInfo()`
- Ensure View() method returns only content area (message + separator + input)
- Preserve responsive design for content area dimensions
- Maintain all existing functionality for message/input handling

**Dependencies**: None

### Task 2: Enhance Tab Header System for Status Integration  
**Acceptance Criteria**:
- Modify TabManager.RenderTabHeaders() to accept optional status information
- Add logic to append status info to header line when terminal width > 100
- Ensure status info is right-aligned and doesn't conflict with tab headers
- Preserve existing tab header functionality and styling
- Handle overflow gracefully when status + headers exceed terminal width

**Dependencies**: None

### Task 3: Integrate Planning Status into Main TUI Layout
**Acceptance Criteria**:
- Modify main TUI View() method to collect status info from active planning tabs
- Pass status information to RenderTabHeaders() when appropriate
- Ensure status only appears when planning tab is active and width > 100
- Maintain existing behavior for all other tab types
- Preserve overlay system compatibility

**Dependencies**: Task 1, Task 2

### Task 4: Validation and Testing
**Acceptance Criteria**:
- Tab headers remain on first line across all terminal widths
- Close button appears inline with headers for all tab types
- Status information displays correctly for wide terminals (>100 chars)
- Status information is hidden for narrow terminals (≤100 chars)  
- All tab switching functionality remains intact
- Planning tab functionality (message input, ACP integration) unchanged
- No regressions in other tab types (main, agent tabs)

**Dependencies**: Task 3

## Implementation Details

### Status Information Flow
1. Planning tab exposes status via `GetStatusInfo() string` method
2. Main TUI detects active planning tab and extracts status
3. TabManager receives status info and integrates with header rendering
4. Single header line contains both tabs and status information

### Layout Calculations
- Tab headers render from left side  
- Status information renders from right side
- Minimum spacing maintained between tabs and status
- Graceful degradation when combined content exceeds terminal width

### Responsive Behavior
- **Wide terminals (>100 chars)**: Show tabs + status on same line
- **Narrow terminals (≤100 chars)**: Show only tabs, hide status  
- **Edge cases**: Handle gracefully when tab names + status exceed width

## Validation Commands

### Manual Testing Sequence
```bash
# 1. Build the application
go build ./cmd/kiro-krew

# 2. Start kiro-krew and create planning tab
./kiro-krew
plan test planning layout

# 3. Switch terminal sizes and verify layout
# Test narrow terminal (< 100 chars)
resize -s 20 80  
# Verify: Only tab headers visible

# Test wide terminal (> 100 chars) 
resize -s 20 120
# Verify: Tab headers + status on same line

# 4. Test tab switching with status
# Create multiple tabs and switch between them
# Verify status appears only for active planning tabs

# 5. Test close button positioning
# Verify close button (×) appears inline with tab name
```

### Automated Tests
```bash
# Run existing TUI tests to ensure no regressions
go test ./internal/tui/... -v

# Specific layout tests should pass:
# - Tab header rendering consistency
# - Planning tab status integration
# - Responsive layout behavior
```

### Expected Behavior Verification

**Terminal Width ≤ 100**:
- Tab headers appear on line 1 
- Close buttons inline with tab names
- No status information visible
- Planning tab content starts immediately after headers

**Terminal Width > 100**:  
- Tab headers appear on line 1 (left side)
- Status information appears on line 1 (right side)  
- Close buttons inline with tab names (left side)
- Planning tab content starts immediately after combined header line

**All Terminal Widths**:
- Tab switching maintains consistent header positioning
- No content displacement or layout shifts
- Overlay system continues to work correctly