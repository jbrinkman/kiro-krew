# Visual Tab Headers Design Specification

**Issue:** #63 - Add visual tab headers to TUI interface  
**Closes:** #63

## Solution Approach

Add visual tab headers to the existing TUI by integrating a tab header rendering system above the current content area. This follows the bubbletea tabs example pattern while preserving all existing functionality and keyboard shortcuts.

### Architecture Decision
- Extend `TabManager` with tab header rendering capability
- Add tab header styles to `Styles` struct  
- Integrate mouse click support for tab switching
- Modify main TUI view to include tab headers above content

## Relevant Files

### Files to Modify:
1. `internal/tui/tab_manager.go` - Add tab header rendering methods
2. `internal/tui/styles.go` - Add tab header styles (active, inactive, close button)
3. `internal/tui/tui.go` - Integrate tab headers into main view, add mouse click handling
4. `internal/tui/main_tab.go` - Update title to be more concise ("Main")

### Files for Reference:
- `internal/tui/tabs.go` - Tab interface (no changes needed)
- `internal/tui/agent_tab.go` - Agent tab implementation (no changes needed)

## Step-by-Step Task Breakdown

### Task 1: Add Tab Header Styles
**Acceptance Criteria:**
- Add `ActiveTab`, `InactiveTab`, and `CloseButton` styles to `Styles` struct
- Use theme colors with appropriate borders and highlighting
- Follow bubbletea tabs example pattern for border styling

### Task 2: Extend TabManager with Header Rendering  
**Acceptance Criteria:**
- Add `RenderTabHeaders(width int) string` method to `TabManager`
- Render tab titles with active/inactive styling
- Add close buttons (×) for closable tabs only
- Handle terminal width constraints gracefully
- Return empty string if no tabs or width too small

### Task 3: Update Main Tab Title
**Acceptance Criteria:** 
- Change `MainTab.Title()` from "Main TUI" to "Main" for space efficiency

### Task 4: Integrate Tab Headers into Main View
**Acceptance Criteria:**
- Modify `tui.go` `View()` method to include tab headers above content
- Reserve appropriate height for tab headers (1-2 lines)
- Adjust content area height accordingly
- Only show headers when multiple tabs exist or when at least one agent tab exists

### Task 5: Add Mouse Click Support
**Acceptance Criteria:**
- Handle `tea.MouseClickMsg` events for tab header area
- Map click positions to tab indices for switching
- Support close button clicks for closable tabs
- Preserve existing keyboard navigation ([/], F2, Ctrl+W)

### Task 6: Handle Width Constraints
**Acceptance Criteria:**
- Gracefully truncate or hide tabs when terminal is narrow
- Prioritize showing active tab and close tabs
- Show indicators (...) when tabs are truncated
- Minimum functional width of ~40 characters

## Team Orchestration

This is a single-component change focused on the TUI layer. No coordination with other systems required.

**Dependencies:** None - all tab management infrastructure already exists  
**Affected Systems:** TUI display layer only

## Validation Commands

```bash
# Build and test basic functionality
go build ./cmd/kiro-krew
./kiro-krew

# In TUI, verify:
# 1. Tab headers appear when agent tabs are created 
# 2. Active tab is highlighted
# 3. Mouse clicks work on headers
# 4. Close buttons work on agent tabs  
# 5. Keyboard shortcuts still work: [, ], F2, Ctrl+W
# 6. Narrow terminal handling (resize to ~40 chars)

# Run existing tests to ensure no regressions
go test ./internal/tui/...
```

## Implementation Notes

- **Minimal UI Changes:** Focus only on adding visual headers, no changes to core tab logic
- **Preserve UX:** All existing keyboard shortcuts must continue working
- **Mouse Support:** Optional enhancement - should not break if mouse is unavailable
- **Performance:** Tab header rendering should be lightweight (single pass)
- **Styling:** Consistent with existing lipgloss theme system