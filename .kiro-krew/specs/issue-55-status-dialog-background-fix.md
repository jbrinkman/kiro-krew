# Issue #55: Fix background color inconsistency in status dialog for high-contrast theme

**Closes #55**

## Problem Analysis

The status dialog in the TUI exhibits visual inconsistency in the high-contrast theme where the text content background differs from the dialog's surface background. This creates an unprofessional appearance with mismatched background colors within the same overlay.

### Root Cause

After analyzing the codebase, the issue stems from the `OverlayContent` style definition in `internal/tui/styles.go`:

```go
OverlayContent: lipgloss.NewStyle().
    Foreground(lipgloss.Color(theme.Colors.TextPrimary)),
```

The `OverlayContent` style sets the text color to `text_primary` but does not explicitly set a background color, while the `OverlayBorder` style correctly uses:

```go
OverlayBorder: lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(lipgloss.Color(theme.Colors.Primary)).
    Background(lipgloss.Color(theme.Colors.Surface)).
    Padding(1, 2),
```

In the high-contrast theme:
- `surface` color is `#222222` (dark gray)
- `text_primary` color is `#FFFFFF` (white)

Without an explicit background, the content area may inherit a different background than the border surface, creating the visual inconsistency.

## Solution Approach

The fix is minimal and surgical: ensure `OverlayContent` explicitly inherits the same background color as `OverlayBorder` by setting the background to `theme.Colors.Surface`.

This approach:
1. Maintains visual consistency across all overlay types
2. Preserves existing theming architecture
3. Requires minimal code changes
4. Does not affect other overlay dialogs negatively

## Architecture Impact

- **File Scope**: Single line change in `internal/tui/styles.go`
- **Theme System**: No changes to theme loading or validation
- **Overlay System**: Enhanced consistency without functional changes
- **Backward Compatibility**: Fully maintained across all themes

## Relevant Files

### Modified Files
- `internal/tui/styles.go` - Fix `OverlayContent` style background

### Test Files  
- All themes in `.kiro-krew/themes/` - Validation targets
- `internal/tui/commands.go` - Status command for testing

### Related Files (No Changes Required)
- `internal/tui/tui.go` - Overlay rendering logic
- `internal/config/themes.go` - Theme loading system

## Team Orchestration

This is a single-developer task requiring:
1. **Developer Role**: Make the minimal style fix
2. **QA Role**: Validate across all available themes
3. **Integration Testing**: Verify overlay consistency

No coordination with external teams or complex system interactions required.

## Step-by-Step Task Breakdown

### Task 1: Fix OverlayContent Background
**Acceptance Criteria:**
- [ ] Modify `OverlayContent` style in `internal/tui/styles.go` to include `Background(lipgloss.Color(theme.Colors.Surface))`
- [ ] Ensure change matches the pattern used in `OverlayBorder`
- [ ] Code compiles without errors

**Implementation:**
```go
OverlayContent: lipgloss.NewStyle().
    Foreground(lipgloss.Color(theme.Colors.TextPrimary)).
    Background(lipgloss.Color(theme.Colors.Surface)),
```

### Task 2: Validate High-Contrast Theme Fix
**Acceptance Criteria:**
- [ ] Status dialog background is visually consistent in high-contrast theme
- [ ] Content background matches border/surface background
- [ ] Text remains readable with proper contrast

### Task 3: Cross-Theme Validation  
**Acceptance Criteria:**
- [ ] Default theme status dialog remains visually consistent
- [ ] Light theme status dialog remains visually consistent  
- [ ] All overlay dialogs (status, help, about) maintain consistency
- [ ] No regression in text readability across themes

### Task 4: Integration Testing
**Acceptance Criteria:**
- [ ] Status command displays correctly in all themes
- [ ] Help overlay maintains consistency
- [ ] About overlay maintains consistency
- [ ] Theme switching preserves overlay appearance

## Validation Commands

### Theme Testing Sequence
```bash
# Build the application
go build -o kiro-krew ./cmd/kiro-krew

# Test each theme
./kiro-krew # Start TUI

# In TUI, test each theme:
theme high-contrast
status
# Verify: background consistency in status dialog

theme default  
status
# Verify: no regression in default theme

theme light
status  
# Verify: no regression in light theme

# Test other overlays for consistency
help
about
```

### Visual Validation Checklist
```bash
# For each theme (high-contrast, default, light):
# 1. Open status dialog - check background consistency
# 2. Open help dialog - verify no regression  
# 3. Open about dialog - verify no regression
# 4. Switch themes - verify consistent behavior
```

### Automated Testing
```bash
# Ensure no compilation errors
go build ./...

# Run existing tests
go test ./internal/tui/...
go test ./internal/config/...
```

## Risk Assessment

**Low Risk Changes:**
- Single line modification to existing style
- No changes to theme system architecture  
- No changes to overlay rendering logic

**Mitigation Strategies:**
- Validate all existing themes before and after change
- Verify no impact to other overlay types
- Test theme switching functionality

## Success Criteria

1. **Primary Fix**: High-contrast theme status dialog has consistent background colors
2. **No Regressions**: Default and light themes remain visually correct
3. **Consistency**: All overlay types maintain visual coherence
4. **Architecture Preservation**: Theme system and overlay logic unchanged

## Post-Implementation Verification

After implementation, verify using the validation commands that:
- Issue #55 reported problem is resolved in high-contrast theme
- No visual regressions occur in other themes
- All overlay dialogs maintain consistent theming
- Theme switching continues to work correctly

This minimal, targeted fix addresses the specific background color inconsistency while maintaining the robustness and flexibility of the existing theming system.