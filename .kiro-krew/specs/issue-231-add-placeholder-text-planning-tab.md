# Design Specification: Add Placeholder Text to Planning Tab Message Input

**Issue**: #231  
**Closes**: #231  
**Created**: 2026-08-25

## Problem Statement

The planning tab's message input (`internal/tui/planning_tab.go`) currently displays no placeholder text when empty, leaving users without guidance on the input's purpose. A previous placeholder ("Type your message here...") was removed in commit 3350e44 (PR #228) because bubbles/v2's virtual cursor mechanism rendered the first character of the placeholder text as the cursor glyph, causing a "T" artifact.

The root cause is the `placeholderView()` function in bubbles/v2, which uses the first placeholder character as the virtual cursor representation when virtual cursor mode is enabled. The solution is to disable the virtual cursor while retaining placeholder functionality.

## User Story

As a user opening a new planning tab, I want to see helpful placeholder text in the message input that hints at the planner's purpose — asking questions or describing tasks to plan — so that I understand what the input is for without needing to read documentation.

## Solution Approach

The bubbles/v2 textinput library provides three key mechanisms:

1. **`Placeholder` field**: Text displayed when the input is empty
2. **`SetVirtualCursor(bool)`**: Controls whether the virtual cursor uses placeholder characters
3. **`Styles.Focused.Placeholder` and `Styles.Blurred.Placeholder`**: lipgloss.Style for placeholder appearance

By setting `ti.SetVirtualCursor(false)`, we disable the virtual cursor mechanism that caused the artifact, allowing us to safely use placeholder text with a real cursor. The placeholder styling can be configured via the textinput's `Styles.Focused.Placeholder` and `Styles.Blurred.Placeholder` fields.

## Relevant Files

### Files to Modify

1. **`internal/tui/planning_tab.go`** (lines 96-105)
   - `NewPlanningTabWithSession()`: Textinput initialization
   - Add placeholder text configuration
   - Disable virtual cursor with `ti.SetVirtualCursor(false)`
   - Configure placeholder styling using `Styles.Focused.Placeholder` and `Styles.Blurred.Placeholder`

2. **`internal/tui/integration_test.go`**
   - Update regression test that currently expects empty placeholder
   - Verify placeholder text is set correctly
   - Ensure virtual cursor is disabled

### Files Referenced (No Changes)

- **`internal/tui/styles.go`**: Theme color definitions (use `TextMuted` color for placeholder)
- **`internal/config/themes.go`**: Theme structure with color fields
- **`go.mod`**: Confirms bubbles/v2 v2.1.0 dependency

## Implementation Details

### Textinput Configuration (planning_tab.go:96-105)

Current implementation (lines 96-105):
```go
// Create simple textinput for message input with terminal prompt style
ti := textinput.New()
ti.Placeholder = "" // No placeholder — avoids virtual cursor rendering first char as cursor glyph
ti.Prompt = ""      // We'll render the prompt ourselves for consistent styling
ti.CharLimit = 4000 // Reasonable message limit

// Configure solid cursor (non-blinking)
currentStyles := ti.Styles()
currentStyles.Cursor.Blink = false
ti.SetStyles(currentStyles)

ti.Focus() // Start focused since focusTarget defaults to FocusTargetMessage
```

Required changes:
1. Set `ti.Placeholder` to helpful text (e.g., "ask a question or describe a task")
2. Call `ti.SetVirtualCursor(false)` to disable virtual cursor
3. Configure `currentStyles.Focused.Placeholder` and `currentStyles.Blurred.Placeholder` with muted styling

### Placeholder Text Content

The placeholder should reflect the planner agent's purpose: collaborating with users to refine ideas into well-structured GitHub issues through interactive Q&A.

**Recommended text**: `"ask a question or describe a task"`

This is:
- Concise and actionable
- Hints at both question-asking and task description use cases
- Matches the terminal-style aesthetic of the planning tab
- Lowercase to match the minimal, clean style

Alternative options:
- `"describe your idea or ask a question"`
- `"what would you like to plan?"`
- `"type a message to start planning"`

### Placeholder Styling

The placeholder should use the theme's `TextMuted` color to create a subtle, dimmed appearance:

```go
currentStyles := ti.Styles()

// Configure cursor
currentStyles.Cursor.Blink = false

// Configure placeholder with muted styling
placeholderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(styles.TextMuted))
currentStyles.Focused.Placeholder = placeholderStyle
currentStyles.Blurred.Placeholder = placeholderStyle

ti.SetStyles(currentStyles)
```

Note: The `styles` parameter is available in `NewPlanningTabWithSession()` and contains theme color references. Access `TextMuted` via `pt.styles` after initialization, or pass it during styles configuration.

### Theme Color Reference

From `internal/config/themes.go`, the Theme structure includes:
- `TextMuted`: For dimmed/secondary text (ideal for placeholders)
- `TextPrimary`: For primary text
- `TextSecondary`: For secondary text

The Styles type (`internal/tui/styles.go`) doesn't currently expose a direct placeholder style, but `TextMuted` is available through the theme configuration and should be used to create the placeholder lipgloss.Style.

### Virtual Cursor Behavior

The bubbles/v2 textinput library has two cursor modes:

1. **Virtual Cursor (default)**: Uses the first character of placeholder text as the cursor representation
   - Problem: This caused the "T" artifact from "Type your message here..."
   
2. **Real Cursor**: Normal terminal cursor rendering
   - Solution: Call `ti.SetVirtualCursor(false)` to use real cursor

With virtual cursor disabled, the placeholder text displays normally (styled with muted color) and the real terminal cursor blinks at position 0 when the input is focused and empty.

## Team Orchestration

This is a single-file change with test update. Implementation is straightforward:

1. **Task 1**: Update textinput initialization in `planning_tab.go`
2. **Task 2**: Update integration test to verify new behavior

Tasks are sequential (test depends on implementation).

## Step-by-Step Task Breakdown

### Task 1: Update Textinput Initialization in planning_tab.go

**File**: `internal/tui/planning_tab.go`  
**Location**: Lines 96-105 in `NewPlanningTabWithSession()`

**Changes**:
1. Set `ti.Placeholder = "ask a question or describe a task"`
2. Add `ti.SetVirtualCursor(false)` after textinput creation
3. Configure placeholder styling in `currentStyles.Focused.Placeholder` and `currentStyles.Blurred.Placeholder`
   - Use `lipgloss.NewStyle().Foreground(lipgloss.Color(styles.TextMuted))` for muted appearance
   - Apply to both focused and blurred states for consistency
4. Update inline comment to reflect the new approach

**Acceptance Criteria**:
- Placeholder text is set to "ask a question or describe a task"
- Virtual cursor is disabled via `SetVirtualCursor(false)`
- Placeholder style uses theme's TextMuted color
- Both focused and blurred placeholder styles are configured
- Code compiles without errors
- Comment accurately describes the configuration

**Implementation Notes**:
- The `styles` parameter is available in the function signature but needs to be accessed for the TextMuted color
- Theme colors are accessed via `styles` parameter, but direct field access to `TextMuted` requires checking the Theme structure
- Use `pt.styles` to access theme after initialization, or construct the style during initialization by accessing theme colors

**Code Location**:
```go
// Line 96-105 in NewPlanningTabWithSession()
func NewPlanningTabWithSession(id, title string, styles *Styles, contextTracker *ContextTracker, sessionManager *session.SessionManager, acpClient acp.Client) *PlanningTab {
    // ... existing code ...
    
    // Create simple textinput for message input with terminal prompt style
    ti := textinput.New()
    ti.Placeholder = ""  // <-- UPDATE THIS
    ti.Prompt = ""
    ti.CharLimit = 4000
    
    // <-- ADD ti.SetVirtualCursor(false) HERE
    
    // Configure solid cursor (non-blinking)
    currentStyles := ti.Styles()
    currentStyles.Cursor.Blink = false
    ti.SetStyles(currentStyles)
    
    // <-- ADD placeholder styling to currentStyles before SetStyles call
```

**Dependencies**: None

### Task 2: Update Integration Test

**File**: `internal/tui/integration_test.go`

**Changes**:
1. Locate the test that checks for empty placeholder (currently expects `Placeholder != ""`)
2. Update assertion to verify:
   - `planningTab.textinput.Placeholder == "ask a question or describe a task"`
   - Virtual cursor is disabled (verify via `planningTab.textinput.VirtualCursor() == false`)
3. Update test comment to reflect the new behavior

**Acceptance Criteria**:
- Test verifies placeholder text is set correctly
- Test verifies virtual cursor is disabled
- Test passes with new implementation
- Test comment describes the purpose (preventing virtual cursor artifact while providing placeholder)

**Current Test Code** (approximate location):
```go
if planningTab.textinput.Placeholder != "" {
    t.Errorf("Expected empty placeholder to avoid virtual cursor artifact, got %q", planningTab.textinput.Placeholder)
}
```

**Updated Test Logic**:
```go
// Verify placeholder is set with virtual cursor disabled to avoid artifact
expectedPlaceholder := "ask a question or describe a task"
if planningTab.textinput.Placeholder != expectedPlaceholder {
    t.Errorf("Expected placeholder %q, got %q", expectedPlaceholder, planningTab.textinput.Placeholder)
}

// Verify virtual cursor is disabled
if planningTab.textinput.VirtualCursor() {
    t.Error("Expected virtual cursor to be disabled to prevent placeholder artifact")
}
```

**Dependencies**: Task 1 (implementation must be complete)

## Validation Commands

After implementation, verify the changes with:

### 1. Build Verification
```bash
go build ./cmd/kiro-krew
```
**Expected**: Clean build with no compilation errors

### 2. Test Verification
```bash
go test ./internal/tui -v -run TestPlanningTab
```
**Expected**: All planning tab tests pass, including updated regression test

### 3. Manual Testing
```bash
./kiro-krew
# In the REPL:
plan
# Observe the message input - should display placeholder text when empty
# Type any character - placeholder should disappear
# Delete all text - placeholder should reappear
# Verify no "T" artifact appears
```

**Expected Behavior**:
- Placeholder displays "ask a question or describe a task" in muted color when input is empty
- Placeholder disappears when typing begins
- Placeholder reappears when input is cleared
- Real cursor blinks at position 0 (no virtual cursor artifacts)
- Custom prompt rendering (`[planner] >`) works correctly alongside placeholder

### 4. Theme Consistency Check
```bash
./kiro-krew
# Switch between themes in the REPL:
theme solarized-dark
plan
# Verify placeholder color matches theme's muted text
theme dracula
plan
# Verify placeholder color updates with theme
```

**Expected**: Placeholder color adapts to each theme's TextMuted color

## Constraints and Considerations

### Must Not Break
1. **Custom Prompt Rendering**: The external prompt system (`renderInputArea()`) that displays `[planner] > ` with state-dependent icons must continue working
2. **Focus Behavior**: Focus transitions between message input and footer must remain functional
3. **State-Based Input**: Read-only, active, completed, and failed states must display correctly
4. **Theme System**: Placeholder must respect the current theme's color palette

### Technical Requirements
1. Use built-in textinput placeholder mechanism (not custom overlay)
2. Disable virtual cursor to prevent character substitution artifacts
3. Style placeholder with lipgloss using theme's TextMuted color
4. Apply consistent styling to both focused and blurred states

### Accessibility
- Placeholder text should be descriptive enough to guide users without overwhelming the minimal interface
- Muted styling provides clear visual distinction between placeholder and user input
- Real cursor provides clear focus indication

## Risk Assessment

**Low Risk**: This is a localized change to textinput configuration that addresses a known issue with a documented solution.

**Potential Issues**:
1. **Theme compatibility**: Some themes might not define TextMuted color
   - Mitigation: Use fallback color if TextMuted is empty (e.g., TextSecondary or TextPrimary with reduced opacity)
   
2. **bubbles/v2 API changes**: Future library updates might change SetVirtualCursor behavior
   - Mitigation: Current implementation is documented and tested; regression test catches breakage

3. **Visual consistency**: Placeholder might not match the minimal aesthetic
   - Mitigation: Use lowercase text and TextMuted color for subtle appearance

## Success Criteria

Implementation is complete when:

1. ✅ Placeholder text "ask a question or describe a task" displays in empty input
2. ✅ Virtual cursor is disabled (no character substitution artifacts)
3. ✅ Placeholder uses theme's TextMuted color for subtle appearance
4. ✅ Placeholder disappears when typing, reappears when cleared
5. ✅ Custom prompt rendering continues to work correctly
6. ✅ All existing tests pass
7. ✅ Integration test verifies placeholder and virtual cursor state
8. ✅ No visual artifacts or regressions in planning tab behavior

## References

- **Issue**: #231 - Add placeholder text to planning tab message input
- **Previous Fix**: PR #228, Commit 3350e44 - Removed placeholder to fix "T" artifact
- **Bubbles Library**: charm.land/bubbles/v2/textinput v2.1.0
- **Related Code**: 
  - `internal/tui/planning_tab.go:96-105` - Textinput initialization
  - `internal/tui/planning_tab.go:506` - renderInputArea() prompt rendering
  - `internal/tui/styles.go` - Theme system and styling
  - `internal/config/themes.go` - Theme color definitions
