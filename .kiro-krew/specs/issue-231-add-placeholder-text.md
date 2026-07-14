# Design Specification: Add Placeholder Text to Planning Tab Message Input

**Issue**: #231  
**Title**: Add placeholder text to planning tab message input  
**Closes**: #231

## Problem Statement

The planning tab's message input (`internal/tui/planning_tab.go`) currently shows no placeholder text when empty. Previously, a placeholder was set (`"Type your message here..."`) but was removed because the bubbles/v2 virtual cursor rendered the first character of the placeholder as the cursor glyph (appearing as a "T" artifact). The fix removed the placeholder entirely, but users now see a blank input with no guidance on what to type.

## Solution Approach

The solution involves:

1. **Adding appropriate placeholder text** that guides users on the planner's purpose
2. **Disabling the virtual cursor** to prevent character artifacts 
3. **Adding proper placeholder styling** using the textinput's built-in placeholder mechanism
4. **Extending the theme system** to support placeholder colors
5. **Ensuring compatibility** with existing custom prompt rendering

### Technical Strategy

- Use textinput's built-in placeholder mechanism rather than custom overlay
- Disable virtual cursor with `ti.SetVirtualCursor(false)` to prevent artifacts
- Add `Placeholder` style to the theme system for consistent styling
- Configure placeholder text that reflects the planner agent's purpose
- Maintain compatibility with the existing `renderInputArea()` custom prompt system

## Relevant Files

### Files to Modify

1. **`internal/tui/planning_tab.go`** (lines 96-105, ~506)
   - Textinput initialization in `NewPlanningTabWithSession()`
   - `renderInputArea()` method (ensure compatibility)

2. **`internal/tui/styles.go`**
   - Add `Placeholder` style field to `Styles` struct
   - Update `NewStyles()` to initialize placeholder style
   - Use `TextMuted` color for placeholder styling

3. **`internal/config/themes.go`**
   - No changes needed - will reuse existing `TextMuted` color

### Dependencies

- `charm.land/bubbles/v2/textinput` - for placeholder functionality
- `charm.land/lipgloss/v2` - for styling

## Team Orchestration

This is a single-component enhancement with no external dependencies. All changes are contained within the TUI layer and can be implemented by a single builder agent.

### Parallel Execution Opportunity

Since this involves only styling and input configuration changes, all modifications can be made simultaneously in one development cycle.

## Step-by-Step Task Breakdown

### Task 1: Add Placeholder Style to Theme System
**Acceptance Criteria**:
- Add `Placeholder` field to `Styles` struct in `internal/tui/styles.go`
- Initialize `Placeholder` style in `NewStyles()` function using `TextMuted` color
- Style should be consistent with existing muted text styling
**Dependencies**: None

### Task 2: Configure Textinput with Placeholder Text and Disabled Virtual Cursor
**Acceptance Criteria**:
- Set placeholder text to "ask a question or describe a task" in `NewPlanningTabWithSession()`
- Call `ti.SetVirtualCursor(false)` to disable virtual cursor
- Configure textinput styles to use the new `Placeholder` style
- Placeholder appears when input is empty and disappears when user types
**Dependencies**: Task 1 (requires Placeholder style)

### Task 3: Verify Compatibility with Custom Prompt Rendering
**Acceptance Criteria**:
- Confirm `renderInputArea()` continues to work correctly with placeholder text
- Verify no visual artifacts when combining custom prompt with placeholder
- Test that placeholder styling doesn't interfere with state-dependent prompts
- Placeholder works across all planning tab states (idle, active, completed, etc.)
**Dependencies**: Task 2

## Implementation Details

### Placeholder Text Choice

The placeholder text "ask a question or describe a task" was chosen because:
- It reflects the planner agent's collaborative purpose
- It's concise and actionable
- It guides users without being prescriptive
- It matches the planning session workflow

### Virtual Cursor Solution

The root cause of the original "T" artifact was the bubbles/v2 virtual cursor feature:
- `placeholderView()` uses the first placeholder character as the virtual cursor glyph
- Setting `ti.SetVirtualCursor(false)` disables this behavior
- The real cursor will still function normally

### Styling Integration

The placeholder styling approach:
- Reuses existing `TextMuted` color from the theme system
- No new theme colors needed - maintains backward compatibility
- Consistent with other muted UI elements (tab inactive states, autocomplete ghost text)
- Uses textinput's built-in `Styles().Placeholder` configuration

### Code Changes Preview

**In `internal/tui/styles.go`**:
```go
type Styles struct {
    // ... existing fields ...
    Placeholder lipgloss.Style  // Add this field
}

func NewStyles(theme *config.Theme) *Styles {
    return &Styles{
        // ... existing styles ...
        Placeholder: lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.TextMuted)),
    }
}
```

**In `internal/tui/planning_tab.go`**:
```go
// Create simple textinput for message input with terminal prompt style
ti := textinput.New()
ti.Placeholder = "ask a question or describe a task"  // Set helpful placeholder
ti.Prompt = ""      // We'll render the prompt ourselves for consistent styling
ti.CharLimit = 4000 // Reasonable message limit

// Configure solid cursor (non-blinking) and disable virtual cursor
currentStyles := ti.Styles()
currentStyles.Cursor.Blink = false
currentStyles.Placeholder = styles.Placeholder  // Apply placeholder styling
ti.SetStyles(currentStyles)
ti.SetVirtualCursor(false)  // Prevent placeholder character artifacts

ti.Focus() // Start focused since focusInput defaults to true
```

## Validation Commands

### Manual Testing
1. **Start application**: `go run ./cmd/kiro-krew`
2. **Create planning tab**: Press `Ctrl+Alt+P` or use planning command
3. **Verify placeholder appears**: Check that "ask a question or describe a task" is visible in empty input
4. **Test placeholder behavior**: Type text and verify placeholder disappears
5. **Test placeholder reappearance**: Clear input and verify placeholder returns
6. **Test visual styling**: Confirm placeholder is styled with muted colors
7. **Test no artifacts**: Verify no "T" or other cursor artifacts appear

### Automated Validation
1. **Unit tests**: Verify textinput configuration is correct
2. **Integration tests**: Test placeholder behavior in different tab states
3. **Theme tests**: Confirm placeholder styling works with different themes

### Cross-State Testing
- Test placeholder in all planning tab states: idle, active, completed, failed, read-only
- Verify custom prompt rendering compatibility
- Check placeholder doesn't interfere with focus states

## Constraints & Compatibility

### Must Maintain
- Existing custom prompt rendering system (`renderInputArea()`)
- All planning tab states and their visual indicators
- Theme system compatibility
- Focus management and keyboard navigation
- Session persistence functionality

### Must Not Break
- Custom prompt with state-dependent icons and colors
- Tab switching and focus restoration
- Keyboard shortcuts and input handling
- Message sending and streaming response display

## Success Criteria

1. ✅ Placeholder text "ask a question or describe a task" appears in empty input
2. ✅ Virtual cursor is disabled (`ti.SetVirtualCursor(false)`)
3. ✅ Placeholder styled with `TextMuted` color (consistent with theme)
4. ✅ Placeholder disappears when user types
5. ✅ Placeholder reappears when input is cleared
6. ✅ Custom prompt rendering continues to work
7. ✅ No visual artifacts or cursor overlapping issues
8. ✅ Works across all planning tab states
9. ✅ Compatible with existing theme system
10. ✅ No regression in keyboard navigation or focus management

## Risk Mitigation

**Risk**: Breaking existing custom prompt rendering  
**Mitigation**: Preserve all existing `renderInputArea()` logic and test thoroughly

**Risk**: Theme compatibility issues  
**Mitigation**: Use existing `TextMuted` color, no new theme fields required

**Risk**: Introducing new cursor artifacts  
**Mitigation**: Explicitly disable virtual cursor and test with different terminal environments

**Risk**: Focus management regressions  
**Mitigation**: Maintain existing focus logic, only modify placeholder configuration