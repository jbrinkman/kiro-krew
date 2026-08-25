# Design Specification: Add Placeholder Text to Planning Tab Message Input

**Issue**: #231  
**Title**: Add placeholder text to planning tab message input  
**Repository**: jbrinkman/kiro-krew  
**Closes**: #231

## Problem Summary

The planning tab's message input (`internal/tui/planning_tab.go`) currently shows no placeholder text when empty. A previous fix (PR #228, commit 3350e44) removed the placeholder (`"Type your message here..."`) to eliminate a visual artifact where the bubbles/v2 virtual cursor rendered the first character of the placeholder ("T") as the cursor glyph. Users now see a blank input with no guidance on what to type.

## Root Cause

The bubbles/v2 textinput component's `placeholderView()` method uses the first character of the placeholder text as the virtual cursor glyph when virtual cursor mode is enabled (default behavior). This creates a visual artifact where the placeholder's first character appears as the cursor instead of the actual text.

## Solution Approach

Re-introduce placeholder text while disabling the virtual cursor feature to prevent the artifact. The solution involves three coordinated changes:

1. **Disable virtual cursor** via `SetVirtualCursor(false)` to prevent the first-character glyph rendering
2. **Set meaningful placeholder text** that describes the planner agent's purpose
3. **Style the placeholder** using the textinput's built-in `Styles.Focused.Placeholder` and `Styles.Blurred.Placeholder` fields with muted/dimmed styling consistent with the theme system

This approach uses the textinput's native placeholder mechanism rather than implementing a custom overlay, ensuring compatibility with the existing custom prompt rendering system.

## Relevant Files

### Files to Modify

- **`internal/tui/planning_tab.go`** (lines 171-180)
  - Textinput initialization in `NewPlanningTabWithSession()`
  - Add `ti.SetVirtualCursor(false)` call
  - Set `ti.Placeholder` to descriptive text
  - Configure placeholder styling via `ti.Styles()`

### Files to Review (No Changes Required)

- **`internal/tui/planning_tab.go`** (line 628+)
  - `renderInputArea()` method — ensure placeholder works with custom prompt rendering
- **`internal/config/themes.go`**
  - Verify `TextMuted` color is available for placeholder styling
- **`internal/tui/integration_test.go`**
  - Update test that checks for empty placeholder (added in commit 3350e44)

## Architecture Context

### Textinput Styling Structure

The bubbles/v2 textinput uses a hierarchical style system:

```
Styles
├── Focused (StyleState)
│   ├── Text
│   ├── Placeholder       ← dimmed text when empty and focused
│   ├── Suggestion
│   └── Prompt
├── Blurred (StyleState)
│   ├── Text
│   ├── Placeholder       ← dimmed text when empty and not focused
│   ├── Suggestion
│   └── Prompt
└── Cursor (CursorStyle)
```

The `Placeholder` field in both `Focused` and `Blurred` states controls placeholder appearance. We'll style both states identically using the theme's `TextMuted` color for consistency.

### Custom Prompt Rendering

The planning tab uses custom prompt rendering via `renderInputArea()` which:
- Renders state-dependent prompts (`[planner] ●`, `[planner] >`, etc.)
- Concatenates the styled prompt with `pt.textinput.View()`
- Applies different prompt styles based on planning state (idle, active, completed, failed, read-only)

The placeholder text appears **inside** the textinput view, after the custom prompt, so there's no conflict with the existing rendering system.

### Virtual Cursor Behavior

The virtual cursor feature in bubbles/v2 textinput:
- **When enabled** (default): Uses first placeholder character as cursor glyph in `placeholderView()`
- **When disabled** via `SetVirtualCursor(false)`: Renders placeholder text normally as dimmed text
- **Purpose**: Virtual cursor provides visual cursor indication in environments where hardware cursor isn't visible

Since Kiro Krew's TUI environment has a working hardware cursor, disabling the virtual cursor is safe and eliminates the artifact.

## Team Orchestration

This is a single-task implementation with no dependencies. All changes are confined to textinput initialization in `planning_tab.go` and can be completed in one pass. The test update can be done in the same file change.

## Step-by-Step Task Breakdown

### Task 1: Implement Placeholder Text with Virtual Cursor Disabled

**Acceptance Criteria**:
1. Set placeholder text to `"ask a question or describe a task"` (lowercase to match terminal aesthetic)
2. Call `ti.SetVirtualCursor(false)` before `ti.Focus()`
3. Configure placeholder styling on both focused and blurred states using theme's `TextMuted` color
4. Placeholder appears when textinput is empty
5. Placeholder disappears when user begins typing
6. Placeholder reappears when input is cleared
7. No visual artifacts from virtual cursor glyph

**Implementation Details**:

Location: `internal/tui/planning_tab.go`, lines 171-180 in `NewPlanningTabWithSession()`

Changes required:
```go
ti := textinput.New()
ti.Placeholder = "ask a question or describe a task" // Changed from "" to descriptive text
ti.Prompt = ""      // We'll render the prompt ourselves for consistent styling
ti.CharLimit = 4000 // Reasonable message limit

// Configure solid cursor (non-blinking) with placeholder styling
currentStyles := ti.Styles()
currentStyles.Cursor.Blink = false

// Style placeholder with muted color from theme (same for focused and blurred states)
placeholderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(styles.styles.TextMuted))
currentStyles.Focused.Placeholder = placeholderStyle
currentStyles.Blurred.Placeholder = placeholderStyle

ti.SetStyles(currentStyles)

// Disable virtual cursor to prevent first-character glyph artifact
ti.SetVirtualCursor(false)

ti.Focus() // Start focused since focusTarget defaults to FocusTargetMessage
```

**Key Points**:
- Placeholder text is lowercase to match terminal aesthetic established in custom prompts
- Call `SetVirtualCursor(false)` **after** setting styles but **before** `Focus()`
- Access theme's `TextMuted` color via `styles` parameter (the `*Styles` struct from `internal/tui/styles.go`)
- Note: The function receives `styles *Styles`, but we need the theme colors. The `NewStyles()` function in `styles.go` receives the theme, but we don't have direct access here. We need to use the colors already in the `Styles` struct.

**Correction**: Looking at the code structure, the `styles` parameter is `*Styles` from `internal/tui/styles.go`, not the theme directly. We need to create the placeholder style using lipgloss directly with the muted color. Since `Styles` doesn't expose the theme colors directly, we should use a dimmed/faded variant of the text color or use a standard muted color.

**Revised approach**: Use lipgloss's built-in color fading or reference ANSI color 240 (gray) as a universal muted color for placeholder text:

```go
// Style placeholder with universally readable muted gray
placeholderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // ANSI gray
```

However, checking the theme system more carefully, themes define `TextMuted` but `Styles` struct doesn't expose it. We should look at how other muted text is styled in the codebase. Looking at `styles.go` line 26, the `TabInactive` style uses `theme.Colors.TextMuted`. So the theme system **does** have TextMuted available during `NewStyles()` construction.

**Final approach**: Since we can't access the theme directly in `NewPlanningTabWithSession`, we should either:
1. Add a field to `Styles` struct that exposes the `TextMuted` color
2. Use an existing style that's already muted (like examining what `TabInactive` foreground is)
3. Use ANSI color 240 as a safe default gray

For simplicity and consistency, **Option 3** is best: use ANSI 240 (gray) which is universally supported and matches the typical muted text appearance across all themes.

**Dependencies**: None

**Testing**:
- Manual verification: Run the TUI, open planning tab, observe placeholder text without "T" artifact
- Automated: Update integration test to verify placeholder is set and virtual cursor is disabled

### Task 2: Update Integration Test

**Acceptance Criteria**:
1. Test verifies `ti.Placeholder` equals `"ask a question or describe a task"`
2. Test verifies `ti.VirtualCursor()` returns `false`
3. Test verifies `ti.Value()` is empty on initialization
4. Test passes without breaking existing assertions

**Implementation Details**:

Location: `internal/tui/integration_test.go`, around line 306 (in `TestTask8Integration`)

Replace the existing test block:
```go
// Test textinput starts empty (no placeholder artifact like "T")
if planningTab.textinput.Value() != "" {
    t.Errorf("Expected empty textinput value, got %q", planningTab.textinput.Value())
}
if planningTab.textinput.Placeholder != "" {
    t.Errorf("Expected empty placeholder to avoid virtual cursor artifact, got %q", planningTab.textinput.Placeholder)
}
```

With:
```go
// Test textinput starts empty with placeholder (virtual cursor disabled to avoid "T" artifact)
if planningTab.textinput.Value() != "" {
    t.Errorf("Expected empty textinput value, got %q", planningTab.textinput.Value())
}
expectedPlaceholder := "ask a question or describe a task"
if planningTab.textinput.Placeholder != expectedPlaceholder {
    t.Errorf("Expected placeholder %q, got %q", expectedPlaceholder, planningTab.textinput.Placeholder)
}
if planningTab.textinput.VirtualCursor() != false {
    t.Error("Expected virtual cursor to be disabled to prevent placeholder artifact")
}
```

**Dependencies**: Task 1 (requires the implementation changes to be in place)

**Testing**:
- Run `go test ./internal/tui -v -run TestTask8Integration`
- Verify test passes with new assertions

## Validation Commands

### Build and Compile Check
```bash
go build ./...
```

### Run Integration Tests
```bash
go test ./internal/tui -v -run TestTask8Integration
```

### Run All Tests
```bash
go test ./...
```

### Manual Verification
```bash
# Build and run the application
go build ./cmd/kiro-krew

# Launch the TUI and switch to planning mode
./kiro-krew

# In the REPL:
# 1. Press Ctrl+Alt+P (or Ctrl+Option+P on macOS) to switch to planning mode
# 2. Verify placeholder text "ask a question or describe a task" appears in empty input
# 3. Type any character and verify placeholder disappears
# 4. Clear the input and verify placeholder reappears
# 5. Verify no "T" artifact or cursor glyph appears
```

### Visual Validation Checklist
- [ ] Placeholder text appears when input is empty
- [ ] Placeholder has muted/dimmed appearance (gray color)
- [ ] Placeholder disappears when typing begins
- [ ] Placeholder reappears when input is cleared (backspace to empty)
- [ ] No "T" character or cursor glyph artifacts
- [ ] Custom prompt (`[planner] >`) renders correctly before placeholder
- [ ] Cursor is visible and positioned correctly in empty input
- [ ] Theme switching doesn't break placeholder styling

## Acceptance Criteria Verification

| Criterion | Task | Validation Method |
|-----------|------|-------------------|
| 1. Textinput displays placeholder text | Task 1 | Manual TUI verification, integration test |
| 2. Virtual cursor is disabled | Task 1 | Integration test checks `VirtualCursor() == false` |
| 3. Placeholder has muted/dimmed appearance | Task 1 | Manual verification of gray color (ANSI 240) |
| 4. Placeholder disappears when typing | Task 1 | Manual TUI verification |
| 5. Placeholder reappears when cleared | Task 1 | Manual TUI verification |
| 6. Custom prompt rendering still works | Task 1 | Manual verification of `[planner] >` prefix |
| 7. No cursor/placeholder artifacts | Task 1 | Manual verification, integration test |

## Risk Assessment

**Low Risk**: Changes are minimal and localized to textinput initialization. The virtual cursor disable is a safe operation that only affects placeholder rendering, not cursor functionality.

**Rollback Strategy**: If issues arise, revert to empty placeholder by setting `ti.Placeholder = ""` and removing the `SetVirtualCursor(false)` call.

## Design Rationale

### Why "ask a question or describe a task"?

This placeholder text clearly communicates the planner agent's purpose: collaborative issue planning through Q&A or task description. It's:
- **Concise**: Fits comfortably in typical terminal widths
- **Lowercase**: Matches the minimal terminal aesthetic of the custom prompts
- **Actionable**: Suggests specific user actions (asking or describing)
- **Aligned with agent purpose**: Reflects the planner's role in refining ideas into structured GitHub issues

### Why ANSI Color 240 (Gray)?

- **Universal compatibility**: ANSI 240 is a standard gray supported across all terminal emulators
- **Theme-independent**: Works visually with both light and dark themes
- **Consistent with conventions**: Gray is the standard placeholder color in most text inputs
- **No theme coupling**: Avoids needing direct theme access in the planning tab constructor

### Why Disable Virtual Cursor Instead of Alternative Solutions?

Alternative approaches considered:
1. **Custom placeholder overlay**: More complex, risks layout conflicts with custom prompt
2. **Change placeholder first character**: Hack that doesn't address root cause
3. **Use empty placeholder with helper text elsewhere**: Loses in-input context

Disabling virtual cursor is the **cleanest solution** because:
- Addresses root cause directly (virtual cursor glyph rendering)
- Uses native textinput placeholder mechanism
- No custom rendering logic needed
- Virtual cursor feature isn't needed in Kiro Krew's TUI (hardware cursor works)
- Single function call with no side effects

## Notes

- The `styles` parameter in `NewPlanningTabWithSession` is `*Styles` from `internal/tui/styles.go`, not the raw theme config
- The theme's `TextMuted` color is used during `NewStyles()` construction but not directly accessible in planning tab initialization
- Using ANSI 240 (gray) provides consistent muted appearance across all themes without coupling to theme system
- The test update is straightforward: replace assertion checking for empty placeholder with assertion checking for expected text
- No changes needed to `renderInputArea()` — placeholder appears inside `pt.textinput.View()` automatically
