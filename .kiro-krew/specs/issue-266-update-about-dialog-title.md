# Design Specification: Update About Dialog Title from 'Kiro-Krew' to 'Elasticache Krew'

**Issue**: #266  
**Closes**: #266

## Overview

This specification addresses a simple branding update to change the about dialog title from "Kiro-Krew Version Information" to "Elasticache Krew Version Information" in the TUI application.

## Problem Statement

The about command in the REPL currently displays "Kiro-Krew Version Information" as the dialog title. The branding needs to be updated to "Elasticache Krew" to reflect the correct product name.

## Solution Approach

This is a straightforward string replacement in a single location. The change requires:
- Modifying one string literal in the `handleAbout()` function
- No architectural changes or refactoring needed
- No impact on other functionality or references

## Current Implementation Analysis

### File: `internal/tui/commands.go`

**Location**: Line 422 in the `handleAbout()` function

**Current Code**:
```go
func (m model) handleAbout() (model, tea.Cmd) {
	m.aboutDialog.BuildContent()
	m.aboutDialog.UpdateStatusLine([]string{"Checking for updates..."})

	m = m.activateOverlay(overlayAbout, "Kiro-Krew Version Information", m.aboutDialog.GetFullContent())
	return m, checkForUpdateCmd()
}
```

**Context Analysis**:
- The `activateOverlay()` function accepts three parameters: overlay type, title string, and content
- The title parameter is directly passed to the overlay's content structure
- The overlay system renders this title at the top of the dialog
- No other code references this specific string

### Related Components

**File**: `internal/tui/tui.go`
- Contains the `activateOverlay()` function that receives the title parameter
- Stores the title in the `overlayContent` struct
- No logic depends on the specific title value

**File**: `internal/tui/about.go`
- Manages about dialog content generation
- Does NOT contain the title string (title is passed by the caller)
- Generates version information content only

### Verification of Scope

A comprehensive search of the codebase reveals:
- **Only one occurrence** of "Kiro-Krew Version Information" in the entire Go codebase
- Multiple references to "Kiro-Krew" and "Kiro Krew" exist in documentation and other contexts
- **Per acceptance criteria**: Only the about dialog title should be modified
- No test files currently validate the exact title string

## Relevant Files

### Files to Modify
- `internal/tui/commands.go` — Line 422, `handleAbout()` function

### Files for Reference (No Changes Required)
- `internal/tui/tui.go` — Understanding overlay activation mechanism
- `internal/tui/about.go` — Understanding about dialog content structure

## Team Orchestration

This is a single-file, single-line change with no dependencies. No parallel work or coordination required.

## Step-by-Step Task Breakdown

### Task 1: Update About Dialog Title String
**Acceptance Criteria**:
- Replace `"Kiro-Krew Version Information"` with `"Elasticache Krew Version Information"` at line 422 in `internal/tui/commands.go`
- Verify the exact location in the `handleAbout()` function
- Ensure no other strings are modified
- Change is limited to the string literal passed to `activateOverlay()`

**Implementation Details**:
```go
// Change this line:
m = m.activateOverlay(overlayAbout, "Kiro-Krew Version Information", m.aboutDialog.GetFullContent())

// To:
m = m.activateOverlay(overlayAbout, "Elasticache Krew Version Information", m.aboutDialog.GetFullContent())
```

**Dependencies**: None

### Task 2: Verify Build Integrity
**Acceptance Criteria**:
- Project compiles without errors
- No syntax errors introduced
- Application starts successfully

**Implementation Details**:
- Run `go build ./cmd/kiro-krew`
- Verify zero exit status
- Confirm binary is created

**Dependencies**: Task 1

### Task 3: Manual Verification
**Acceptance Criteria**:
- Launch the TUI application
- Execute the `about` command in the REPL
- Confirm the dialog title displays "Elasticache Krew Version Information"
- Verify the rest of the about dialog content remains unchanged

**Implementation Details**:
- Run the built binary: `./kiro-krew`
- Type `about` at the prompt
- Visually inspect the overlay title
- Press ESC to close the overlay

**Dependencies**: Task 2

## Validation Commands

### Build Verification
```bash
# Build the project
go build ./cmd/kiro-krew

# Verify build success
echo $?  # Should output: 0
```

### Manual Testing
```bash
# Run the application
./kiro-krew

# In the REPL, type:
about

# Expected: Dialog title shows "Elasticache Krew Version Information"
# Press ESC to close
```

### Regression Check
```bash
# Verify no unintended changes to other "Kiro-Krew" references
grep -r "Kiro-Krew" internal/tui/*.go

# Should show references in other contexts, but NOT the old about dialog title
```

## Risk Assessment

**Risk Level**: Minimal

**Risks**:
- **Typo risk**: Misspelling "Elasticache" in the replacement
  - Mitigation: Carefully review the exact string before committing
- **Scope creep**: Accidentally modifying other "Kiro-Krew" references
  - Mitigation: Use precise string replacement, verify with `git diff`

**No Breaking Changes**: This change is purely cosmetic and does not affect:
- API interfaces
- Configuration files
- User workflows
- Data structures
- External integrations

## Testing Strategy

**Manual Testing Required**:
1. Visual confirmation of the new title in the about overlay
2. Verification that overlay functionality remains unchanged
3. Confirmation that ESC key still closes the overlay

**Automated Testing**:
- No unit tests currently exist for the exact about dialog title
- Future consideration: Add integration test for about command output
- Not required for this change (per acceptance criteria scope)

## Deployment Considerations

**Build Requirements**:
- Standard Go build process unchanged
- No new dependencies
- No configuration changes

**User Impact**:
- Users will see the updated branding in the about dialog
- No behavioral changes to the about command
- No migration or upgrade steps required

## Success Criteria Summary

✅ String replacement completed at line 422 in `internal/tui/commands.go`  
✅ Project builds successfully  
✅ About command displays "Elasticache Krew Version Information"  
✅ No other "Kiro-Krew" references modified  
✅ All existing functionality preserved  

## Notes

This is a minimal-scope change focused exclusively on the about dialog title. While the codebase contains many other references to "Kiro-Krew" in documentation, comments, and other contexts, the acceptance criteria explicitly limit this change to the about dialog title only.

Future branding updates may address other references, but they are outside the scope of issue #266.
