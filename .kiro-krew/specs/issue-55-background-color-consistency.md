# Design Specification: Fix Background Color Inconsistency in Status Dialog

**Issue**: #55 - Fix background color inconsistency in status dialog for high-contrast theme  
**Closes**: #55

## Problem Analysis

The status dialog in the TUI exhibits visual inconsistency in the high-contrast theme where the text content background differs from the dialog's surface background. This occurs because:

1. **OverlayBorder style** (in `internal/tui/styles.go:28-31`) correctly sets `Background(lipgloss.Color(theme.Colors.Surface))`
2. **OverlayContent style** (in `internal/tui/styles.go:34-35`) only sets `Foreground(lipgloss.Color(theme.Colors.TextPrimary))` but **does not set a background**

In the high-contrast theme:
- `surface` color is `#222222` (dark gray)
- `text_primary` color is `#FFFFFF` (white)
- Without an explicit background, the content area may inherit a different background than the border area

## Root Cause

In `internal/tui/tui.go:504-509`, the overlay rendering applies both styles:
```go
overlayContent := lipgloss.JoinVertical(lipgloss.Left, title, "", m.styles.OverlayContent.Render(contentStr))

return m.styles.OverlayBorder.
    Width(m.overlayWidth - 4).
    Height(m.overlayHeight - 2).
    Render(overlayContent)
```

The `OverlayContent.Render(contentStr)` creates styled content without a background, then `OverlayBorder.Render()` wraps it with a surface background, causing visual inconsistency.

## Solution Approach

**Minimal Fix Strategy**: Add background color to `OverlayContent` style to match `OverlayBorder` surface background.

### Design Principles
- **Consistency**: All overlay components should use the same surface background
- **Inheritance**: Content should inherit the same surface styling as the border
- **Theme Agnostic**: Fix should work across all themes (default, light, high-contrast, custom)
- **Minimal Impact**: Single line change to avoid affecting other overlay types

## Relevant Files

### Files to Modify
- `internal/tui/styles.go` - Add background color to OverlayContent style

### Files for Validation
- `.kiro-krew/themes/high-contrast.yaml` - Primary test theme
- `.kiro-krew/themes/default.yaml` - Validate no regression
- `.kiro-krew/themes/light.yaml` - Validate no regression
- `internal/tui/commands.go` - Status command that triggers the overlay
- `internal/tui/tui.go` - Overlay rendering logic (no changes needed)

## Team Orchestration

### Single Developer Task
This is a minimal UI consistency fix that can be completed by one developer:

1. **Analysis Phase**: Verify the issue across all themes
2. **Implementation Phase**: Add background color to OverlayContent style  
3. **Testing Phase**: Validate fix across all themes and overlay types

### No External Dependencies
- No coordination with other teams required
- No API changes or breaking modifications
- No new dependencies or infrastructure changes

## Step-by-Step Task Breakdown

### Task 1: Verify Current Issue
**Acceptance Criteria**:
- [ ] Reproduce background color inconsistency in high-contrast theme
- [ ] Confirm issue only affects high-contrast theme
- [ ] Document current visual behavior

**Commands**:
```bash
# Build and test current state
go build -o kiro-krew-test cmd/kiro-krew/main.go
# Test with high-contrast theme - manually observe status dialog
```

### Task 2: Implement Background Fix
**Acceptance Criteria**:
- [ ] Add `.Background(lipgloss.Color(theme.Colors.Surface))` to OverlayContent style
- [ ] Verify change is minimal and targeted
- [ ] Ensure no other style definitions are affected

**Implementation Details**:
In `internal/tui/styles.go`, modify the OverlayContent style from:
```go
OverlayContent: lipgloss.NewStyle().
    Foreground(lipgloss.Color(theme.Colors.TextPrimary)),
```

To:
```go
OverlayContent: lipgloss.NewStyle().
    Foreground(lipgloss.Color(theme.Colors.TextPrimary)).
    Background(lipgloss.Color(theme.Colors.Surface)),
```

### Task 3: Cross-Theme Validation
**Acceptance Criteria**:
- [ ] High-contrast theme shows consistent background colors
- [ ] Default theme maintains existing appearance
- [ ] Light theme maintains existing appearance
- [ ] All overlay types (status, help, about) work consistently

### Task 4: Integration Testing
**Acceptance Criteria**:
- [ ] Status command displays properly in all themes
- [ ] Help command overlay background is consistent
- [ ] About command overlay background is consistent
- [ ] No visual regressions in any theme

## Validation Commands

### Build and Test Setup
```bash
# Build the application
go build -o kiro-krew-test cmd/kiro-krew/main.go

# Start the TUI in test mode
./kiro-krew-test
```

### Theme-Specific Testing
```bash
# Test each theme with status command
# In kiro-krew REPL:

# Test high-contrast theme
theme high-contrast
status

# Test default theme  
theme default
status

# Test light theme
theme light
status

# Test help overlay consistency
help

# Test about overlay consistency  
about
```

### Visual Validation Checklist
For each theme, verify:
- [ ] Status dialog has consistent background color between border and content areas
- [ ] Text remains readable with proper contrast
- [ ] No visual artifacts or color bleeding
- [ ] Help and about overlays maintain consistency
- [ ] Overall UI polish and professional appearance

### Automated Validation
```bash
# Run existing tests to ensure no regressions
go test ./internal/tui/...
go test ./internal/config/...

# Run integration tests if available
./test_integration.sh
```

## Risk Assessment

### Low Risk Changes
- **Single line addition**: Minimal code change reduces chance of introducing bugs
- **Style-only modification**: No logic or data flow changes
- **Theme-agnostic**: Uses existing theme color definitions

### Potential Issues
- **Color contrast**: Ensure text remains readable on surface background in all themes
- **Performance**: Minimal impact as only adds one style property
- **Compatibility**: No breaking changes to existing theme definitions

### Rollback Plan
If issues arise, simply remove the `.Background()` call to revert to original behavior.

## Success Criteria

### Primary Goals
1. **Visual Consistency**: Status dialog background is uniform across border and content areas in high-contrast theme
2. **Cross-Theme Compatibility**: No visual regressions in default or light themes  
3. **Overlay Consistency**: All overlay types (status, help, about) maintain consistent styling

### Secondary Goals
1. **Code Quality**: Clean, minimal implementation following existing patterns
2. **Maintainability**: Solution is easy to understand and modify
3. **Documentation**: Changes are self-documenting through clear code structure

### Definition of Done
- [ ] Background color inconsistency resolved in high-contrast theme
- [ ] All themes tested and validated for visual consistency
- [ ] No functional or visual regressions introduced
- [ ] Code follows existing style and patterns
- [ ] Implementation is minimal and targeted