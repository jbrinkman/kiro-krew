# Issue #211: Agent Tabs Missing Footer System

**Closes #211**

## Solution Approach

This specification addresses the inconsistency where agent tabs do not display the two-row footer system established in Issue #195, unlike the main console and planning tabs. The root cause is in the rendering flow: the `View()` method in `tui.go` currently handles agent tabs and other non-main tabs differently, and while planning tabs receive proper footer rendering through `renderTabContentWithFooter()`, agent tabs also follow the same code path but their content doesn't visually show the expected footer.

Upon deeper investigation, the actual issue is that **all non-main tabs (including agent tabs) already use the unified footer rendering system** via `renderTabContentWithFooter()` in line 797 of `tui.go`. However, agent tabs appear to be missing the footer due to how their content is rendered within the `AgentTab.View()` method through `OutputView`.

The solution involves:
1. **Verifying Footer Rendering Flow**: Confirming that `renderTabContentWithFooter()` is called for agent tabs (it already is)
2. **Investigating OutputView**: Examining why the footer may not be visible in the agent tab layout
3. **Ensuring Consistent Footer Display**: Making sure agent tabs show the same base footer information as other tabs
4. **Layout Adjustments**: Adjusting viewport heights and content area calculations to properly accommodate the footer

## Relevant Files

### Files to Investigate
- `internal/tui/output_view.go` - The OutputView component used by agent tabs
- `internal/tui/agent_tab.go` - Agent tab implementation (may need height calculations)

### Files to Modify
- `internal/tui/output_view.go` - Adjust viewport sizing to account for footer
- `internal/tui/footer.go` - Potentially add agent-specific footer rendering logic
- `internal/tui/tui.go` - Verify and document the unified rendering flow

### Files for Reference
- `internal/tui/planning_tab.go` - Example of proper footer integration
- `internal/tui/main_tab.go` - Example of main tab footer behavior
- `internal/tui/log_tab.go` - Another tab type for comparison
- `.kiro-krew/specs/issue-195-acp-planning-tab.md` - Original footer system specification

## Team Orchestration

The implementation involves coordination between three components:

1. **Footer System**: Already implemented and working for main and planning tabs, provides the two-row footer (input row + status row)
2. **Agent Tab Rendering**: Agent tabs use `OutputView` component which needs to properly size itself to leave room for the footer
3. **Unified Rendering**: The `renderTabContentWithFooter()` method in `tui.go` already handles footer composition for all tabs

**Key Finding**: The rendering path already calls `renderTabContentWithFooter()` for agent tabs (line 797 in `tui.go`), so the issue likely lies in how `OutputView` calculates its content height or how the viewport is sized.

Dependencies:
- OutputView Layout → Footer Height Calculation → Footer Rendering
- Agent Tab Height → OutputView Resize → Content Display

## Step-by-Step Task Breakdown

### Task 1: Investigate and Diagnose OutputView Layout
**Acceptance Criteria**:
- Read `internal/tui/output_view.go` to understand current viewport sizing logic
- Identify why footer may not be visible (viewport consuming full height, content overflow, etc.)
- Document the root cause with specific code references
- Determine if OutputView needs to be aware of footer height when calculating viewport dimensions
**Dependencies**: None (investigation task)

### Task 2: Adjust OutputView Height Calculations
**Acceptance Criteria**:
- Modify `OutputView.Resize()` to account for footer height when sizing the viewport
- Use `footerManager.GetFooterHeight()` to get accurate footer dimensions
- Ensure OutputView doesn't consume the full height allocated to the tab
- Update any related viewport height calculations
- Maintain backward compatibility with other OutputView usages if any exist
**Dependencies**: Task 1 (requires understanding of current implementation)

### Task 3: Verify Footer Content for Agent Tabs
**Acceptance Criteria**:
- Confirm `footer.go`'s `renderStatusRow()` method handles `TabTypeAgent` appropriately
- Ensure agent tabs display base footer info (theme at minimum)
- Verify the footer's "Row 2" (status row) renders for agent tabs
- Test that footer displays consistently with main and planning tab footers
- No agent-specific context information needed (just base theme info like Issue #195)
**Dependencies**: None (can run in parallel with Task 2)

### Task 4: Test and Validate Layout Consistency
**Acceptance Criteria**:
- Agent tabs display the two-row footer system (input row + status row)
- Footer shows base information (theme) consistent with Issue #195 specification
- Agent output viewport properly scrolls within available space
- No content cutoff or layout issues when footer is displayed
- All existing agent tab functionality remains intact
- Footer remains visible when switching between agent tabs
- Verify behavior across different terminal sizes
**Dependencies**: Task 2 (layout adjustments), Task 3 (footer content)

### Task 5: Update Documentation and Comments
**Acceptance Criteria**:
- Add code comments in `output_view.go` explaining footer height calculation
- Document the unified rendering flow in `tui.go` for future reference
- Update any relevant inline documentation about tab layouts
- Ensure the fix is clear for future maintainers
**Dependencies**: Task 4 (after implementation is complete)

## Validation Commands

```bash
# Build the application
task build

# Run the TUI and verify footer appears on agent tabs
./kiro-krew

# Within the TUI:
# 1. Start watcher: watch start
# 2. Wait for agent to spawn (or spawn one manually)
# 3. Switch to agent tab using mouse click or ] key
# 4. Verify two-row footer is visible:
#    - Row 1: Command entry area (kiro-krew> prompt)
#    - Row 2: Status row showing "theme: <theme-name>"

# Test with different terminal sizes
# Resize terminal window and verify footer remains visible and properly positioned

# Test with multiple agent tabs
# Create multiple agents and switch between tabs
# Verify footer persists across all agent tabs

# Compare with planning tab
# Create a planning session: plan test
# Switch between planning tab and agent tab
# Verify footer consistency between tab types

# Test with different themes
echo "theme light" | ./kiro-krew
echo "theme high-contrast" | ./kiro-krew
# Verify footer displays correctly with different themes

# Run existing tests to ensure no regressions
task test
```

## Technical Implementation Notes

### Current Rendering Flow (Issue #195 Implementation)

```go
// From tui.go View() method (line 771):
func (m model) View() tea.View {
    // ... initialization ...
    
    // Render tab headers
    tabHeaders := m.tabManager.RenderTabHeaders(m.width, m.styles)
    
    // Render active tab content
    var content string
    activeTab := m.tabManager.GetActiveTab()
    if activeTab != nil && activeTab.Type() != TabTypeMain {
        // For non-main tabs (INCLUDING AGENT TABS), get their content 
        // and apply unified rendering with footer
        tabContent := activeTab.View()
        content = m.renderTabContentWithFooter(tabContent, activeTab.Type())
    } else {
        // For main tab
        content = m.renderBaseView()
    }
    
    // Combine headers with content
    content = tabHeaders + "\n" + content
    
    // ... overlay rendering ...
}
```

**Key Discovery**: Agent tabs ARE already being passed through `renderTabContentWithFooter()`, which means the footer system is attempting to render for them. The issue is that either:
1. The OutputView is consuming too much vertical space, obscuring the footer
2. The OutputView needs to be resized to leave room for the footer
3. There's a viewport height calculation issue

### Footer System Architecture

From `footer.go`:
- `FooterManager` manages two-row footer display
- `RenderWithSeparator()` produces: separator line + input row + status row
- `GetFooterHeight()` returns 3 (separator + input + status)
- Base info (theme) shown on ALL tabs
- Enhanced info (context, model) only for planning tabs

### Expected Layout for Agent Tabs

```text
┌─────────────────────────────────────────────────────────────────────────────┐
│ [Main TUI] [Planning 1] [Issue 123*]                                        │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Agent output content goes here...                                         │
│  [Scrollable viewport area]                                                │
│  ...                                                                        │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│ kiro-krew> Type your command here...                                        │
│ theme: dark                                                                 │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Implementation Strategy

1. **Height Calculation**: OutputView must subtract footer height from available space
2. **Viewport Sizing**: When `OutputView.Resize()` is called, it should:
   ```go
   // Current (incorrect):
   viewport.Height = height
   
   // Corrected:
   footerHeight := 3 // or get from footerManager
   viewport.Height = height - footerHeight
   ```
3. **Footer Rendering**: Already handled by `renderTabContentWithFooter()` in `tui.go`

### Consistency Check

- **Main Tab**: Uses `renderBaseView()` which internally calls `renderTabContentWithFooter()`
- **Planning Tab**: Returns content from `PlanningTab.View()`, then wrapped by `renderTabContentWithFooter()`
- **Agent Tab**: Returns content from `AgentTab.View()` → `OutputView.View()`, then wrapped by `renderTabContentWithFooter()`
- **Log Tab**: Similar to agent tab, returns viewport content

All tabs should show at minimum:
- Row 1: Command input (kiro-krew> prompt)
- Row 2: Base status (theme: <name>)

Planning tabs additionally show:
- Context usage (ctx: X/Y)
- Model (model: claude-sonnet-4)
- Directory (📁 /path)

### Testing Approach

1. **Visual Verification**: Manual testing with actual TUI to see footer presence
2. **Height Calculation**: Add debug logging to verify viewport heights
3. **Cross-Tab Comparison**: Switch between tab types to ensure consistency
4. **Responsive Testing**: Test with various terminal sizes
5. **Regression Testing**: Run existing test suite to ensure no breakage

## Related Issues and Context

- **Issue #195**: Implemented the two-row footer system with enhanced context information display
- **Footer System**: Introduced `FooterManager`, `ContextTracker`, and unified rendering
- **Planning Tab**: First tab type to fully implement enhanced footer display
- **Current State**: Main and Planning tabs show footer correctly, Agent tabs do not

This issue completes the footer system implementation by ensuring consistent footer display across all tab types as originally intended in Issue #195.
