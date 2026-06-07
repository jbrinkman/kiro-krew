# Issue #62 - Replace alphanumeric shortcut keys with non-conflicting alternatives

Closes #62

## Solution Approach

Replace the conflicting 'o' key shortcut with a non-alphanumeric alternative that doesn't interfere with REPL input while preserving the tab toggle functionality. The F2 key will remain as the primary shortcut, and 'o' will be removed entirely.

## Relevant Files

- `internal/tui/tui.go` - Contains the key handling logic that needs modification

## Team Orchestration

Single file change requiring minimal coordination. No external dependencies or API changes.

## Step-by-Step Task Breakdown

### Task 1: Remove 'o' key from shortcut handling
- **Description**: Remove "o" from the case statement in the key handling switch
- **File**: `internal/tui/tui.go`
- **Line**: ~354
- **Change**: Modify `case "f2", "o":` to `case "f2":`
- **Acceptance Criteria**: 
  - The 'o' key no longer triggers tab toggling
  - F2 key continues to work for tab toggling
  - Typing 'o' in REPL input works normally

### Task 2: Verify REPL input functionality
- **Description**: Ensure all alphanumeric characters can be typed normally in REPL
- **Acceptance Criteria**:
  - Can type commands containing 'o' without triggering shortcuts
  - All other alphanumeric keys work as expected in input

## Validation Commands

```bash
# Build and test the application
go build -o kiro-krew ./cmd/kiro-krew

# Manual testing steps:
# 1. Start kiro-krew in TUI mode
./kiro-krew

# 2. Test that 'o' can be typed normally in REPL
# Type: "echo hello world" - should work without triggering tab toggle

# 3. Test that F2 still works for tab toggling
# Press F2 - should toggle between main and agent tabs

# 4. Test other navigation keys still work
# Press [ and ] - should navigate between tabs
```

## Notes

- This is a minimal change removing only the conflicting 'o' shortcut
- F2 remains as the primary tab toggle mechanism  
- No new shortcuts are introduced to avoid additional conflicts
- Change preserves existing functionality while fixing the REPL input issue