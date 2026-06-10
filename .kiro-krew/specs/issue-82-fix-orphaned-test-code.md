# Design Specification: Fix Orphaned Code in integration_test.go

**Issue:** #82 - Fix compile error in internal/hotkey/integration_test.go - orphaned code outside functions  
**Closes:** #82

## Problem Analysis

The file `internal/hotkey/integration_test.go` contains orphaned code blocks at lines 160-164 and 248-252 that exist outside of any function body. These blocks contain identical temporary directory setup code:

```go
tempDir := t.TempDir()
origDir, _ := os.Getwd()
defer os.Chdir(origDir)
_ = os.Chdir(tempDir)
```

This code appears to be duplicated setup logic that should be inside test functions where they have access to the `*testing.T` parameter.

## Solution Approach

**Minimal Surgical Fix**: Remove the orphaned code blocks entirely since they are duplicates of existing setup code already present within the test functions.

### Analysis of Orphaned Blocks

1. **Lines 160-164**: This block appears between `TestHotkeyIntegrationEndToEnd` and another test function. The same setup code already exists at line 50-53 within `TestHotkeyIntegrationEndToEnd`.

2. **Lines 248-252**: This block appears at the end of a test function or between functions. Similar setup code exists within test functions that need it.

The orphaned blocks serve no purpose and are syntax errors because:
- `t.TempDir()` requires a `*testing.T` parameter only available inside test functions
- The variables are not used outside the function context
- The setup is already handled properly within each test function

## Relevant Files

- `internal/hotkey/integration_test.go` (MODIFY) - Remove orphaned code blocks

## Team Orchestration

This is a single-file syntax fix requiring no coordination between teams or components.

## Step-by-Step Task Breakdown

### Task 1: Remove First Orphaned Block (Lines 160-164)
**Acceptance Criteria:**
- Remove the 5-line orphaned code block at lines 160-164
- Ensure no functional test code is affected
- Maintain proper spacing between functions

### Task 2: Remove Second Orphaned Block (Lines 248-252)  
**Acceptance Criteria:**
- Remove the 5-line orphaned code block at lines 248-252
- Ensure no functional test code is affected
- Maintain proper file structure

### Task 3: Verification
**Acceptance Criteria:**
- File compiles without syntax errors
- All existing test functions remain intact and functional
- No changes to test logic or functionality

## Validation Commands

After implementing the fix, run these commands to verify success:

```bash
# Verify syntax is correct
go build ./internal/hotkey/

# Run the integration tests
go test ./internal/hotkey/ -v

# Run specific integration tests
go test ./internal/hotkey/ -run TestHotkey -v
```

Expected outcome: All commands should execute without compilation errors.

## Implementation Notes

- This is purely a syntax fix - no functional changes
- The orphaned code blocks are exact duplicates of setup code already present in test functions
- No test functionality will be lost by removing these blocks
- The fix enables the package to compile and tests to run successfully