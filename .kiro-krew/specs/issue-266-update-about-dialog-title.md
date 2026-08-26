# Design Specification: Update About Dialog Title

**Issue**: #266  
**Title**: Update about dialog title from 'Kiro-Krew' to 'Elasticache Krew'  
**Created**: 2026-08-26

Closes #266

## Problem Statement

The about dialog currently displays "Kiro-Krew Version Information" as its title. This needs to be updated to reflect "Elasticache Krew" branding to align with the project's rebranding initiative.

## Solution Approach

This is a straightforward string replacement task. The change affects only the title parameter passed to the `activateOverlay()` function within the `handleAbout()` method in `/internal/tui/commands.go`.

### Current Implementation

The about dialog is triggered by the `about` command in the REPL interface. The flow is:

1. User types `about` command
2. Command parser in `tui.go` (line ~1245) calls `handleAbout()`
3. `handleAbout()` in `commands.go` (line 419-424):
   - Builds dialog content using `AboutDialog.BuildContent()`
   - Updates status line with "Checking for updates..."
   - Calls `activateOverlay()` with title **"Kiro-Krew Version Information"**
   - Returns command to check for updates

The `activateOverlay()` function (defined in `tui.go` line 717) stores the title in the overlay content structure, which is then rendered in the UI.

### Change Required

Update line 423 in `/internal/tui/commands.go`:

**Current**:
```go
m = m.activateOverlay(overlayAbout, "Kiro-Krew Version Information", m.aboutDialog.GetFullContent())
```

**New**:
```go
m = m.activateOverlay(overlayAbout, "Elasticache Krew Version Information", m.aboutDialog.GetFullContent())
```

### Scope Limitations

Per acceptance criteria, **only** the about dialog title should be modified. Other occurrences of "Kiro-Krew" or related branding strings elsewhere in the codebase are explicitly out of scope for this issue.

The only other reference found was in `.kiro-krew/specs/issue-49-overlay-popups.md` which is a historical spec document and should not be modified.

## Relevant Files

### Files to Modify

- `/internal/tui/commands.go` (line 423) — Update title string parameter

### Files Referenced (No Changes)

- `/internal/tui/tui.go` — Contains `activateOverlay()` function definition
- `/internal/tui/about.go` — Contains `AboutDialog` implementation
- `/internal/tui/commands_test.go` — Contains existing test for `AboutDialog`

## Team Orchestration

This is a single-file, single-line change with no dependencies. The task can be completed atomically by one builder agent.

No coordination with other components is required since:
- The title is a display-only string with no functional impact
- No tests directly assert on the title text
- The change does not affect any interfaces or data structures

## Step-by-Step Task Breakdown

### Task 1: Update About Dialog Title String

**Description**: Change the title parameter in the `handleAbout()` function from "Kiro-Krew Version Information" to "Elasticache Krew Version Information"

**Acceptance Criteria**:
- Line 423 in `/internal/tui/commands.go` updated with new title string
- String change is exact: "Elasticache Krew Version Information"
- No other lines modified in the file
- Code compiles without errors

**Dependencies**: None

**Implementation Steps**:
1. Open `/internal/tui/commands.go`
2. Locate line 423 in the `handleAbout()` function
3. Replace `"Kiro-Krew Version Information"` with `"Elasticache Krew Version Information"`
4. Save the file
5. Verify compilation with `go build ./cmd/kiro-krew`

## Validation Commands

### Build Verification
```bash
go build ./cmd/kiro-krew
```
Expected: Clean build with no errors

### Manual Testing
```bash
./kiro-krew
# In the REPL, type:
about
```
Expected: Dialog appears with title "Elasticache Krew Version Information"

### Automated Testing
```bash
go test ./internal/tui/...
```
Expected: All existing tests pass (no test changes needed since tests don't assert on title text)

### Code Review Checklist
- [ ] Only line 423 in `/internal/tui/commands.go` modified
- [ ] Title string is exactly "Elasticache Krew Version Information"
- [ ] No other "Kiro-Krew" references modified
- [ ] Code builds successfully
- [ ] Manual verification shows correct title in about dialog

## Notes

### Why This is Safe

1. **Isolated Change**: The title parameter is purely for display purposes
2. **No Breaking Changes**: No API, interface, or data structure changes
3. **No Test Updates Needed**: Existing tests validate structure, not specific string content
4. **No Dependencies**: No other code depends on this specific string value

### Future Considerations

This change is part of a broader rebranding effort. Other occurrences of "Kiro-Krew" throughout the codebase may be addressed in separate issues to maintain atomic, reviewable changes.
