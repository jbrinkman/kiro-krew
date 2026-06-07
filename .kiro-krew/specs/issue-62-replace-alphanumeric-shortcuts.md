# Design Specification: Replace Alphanumeric Shortcut Keys

**Issue:** #62 - Replace alphanumeric shortcut keys with non-conflicting alternatives  
**Status:** Design Phase  
**Closes #62**

## Problem Statement

The TUI currently uses the `o` key as a shortcut to toggle between main and agent tabs, which conflicts with normal REPL usage where users need to type alphanumeric characters including `o` for commands and input. This prevents users from typing naturally in the console input field.

## Current State Analysis

### Existing Keyboard Shortcuts

From analysis of `internal/tui/tui.go` and `internal/tui/commands.go`:

**Conflicting Alphanumeric Shortcuts:**
- `"o"` - Toggle between main and agent tabs (conflicts with typing)

**Non-Conflicting Shortcuts Currently Used:**
- `"f2"` - Also toggles between main and agent tabs (good)
- `"["` - Previous tab navigation
- `"]"` - Next tab navigation  
- `"ctrl+c"` - Exit confirmation
- `"ctrl+w"` - Close current tab
- `"ctrl+alt+p"` - Toggle planning/console modes (via hotkey system)
- `"esc"` - Dismiss overlays
- `"up"`, `"down"`, `"pgup"`, `"pgdown"`, `"home"`, `"end"` - Console scrolling
- `"enter"` - Execute command

### Impact Analysis

The `o` key conflict specifically affects:
- Users typing commands containing 'o' (common: "stop", "focus", "option", etc.)
- Natural text input in REPL environments
- User experience when the interface unexpectedly switches views during typing

## Solution Approach

### Strategy: Remove Conflicting Alphanumeric Keys

1. **Remove `"o"` shortcut** - Keep only the non-conflicting `"f2"` for tab toggling
2. **Verify no other alphanumeric conflicts exist** in the codebase
3. **Update help documentation** to reflect the change
4. **Test REPL functionality** to ensure all alphanumeric keys work normally

### Alternative Shortcuts Considered

**Option 1: Function Keys Only (RECOMMENDED)**
- Keep `"f2"` for tab toggle (already implemented)
- Remove `"o"` entirely
- Most intuitive, zero conflict with typing

**Option 2: Ctrl+Key Combinations** 
- Replace `"o"` with `"ctrl+o"`
- Pros: Still mnemonic
- Cons: More complex, potential conflicts with terminal/editor shortcuts

**Option 3: Symbol Keys**
- Replace `"o"` with `"~"` or `"`" 
- Pros: No typing conflict
- Cons: Less discoverable, not mnemonic

**Decision: Use Option 1** - Function keys provide the cleanest solution with zero conflicts.

## Relevant Files

### Files to Modify

1. **`internal/tui/tui.go`** (PRIMARY)
   - Remove `"o"` from the keyboard shortcut case statement (line ~391)
   - Keep `"f2"` functionality unchanged

2. **`internal/tui/commands.go`** 
   - Update help text in `handleHelp()` function
   - Remove reference to `"o"` shortcut in help content

### Files for Reference/Testing

3. **`docs/hotkey-toggle.md`** 
   - Update documentation if it mentions the `"o"` shortcut
   - Ensure documentation reflects correct shortcuts

4. **`internal/tui/integration_test.go`**
   - Add test cases for alphanumeric input functionality
   - Verify REPL typing works with all letters/numbers

## Team Orchestration

This is a single-developer task with no cross-team dependencies. The change is isolated to the TUI keyboard handling logic.

## Step-by-Step Task Breakdown

### Task 1: Remove Conflicting Shortcut
**Acceptance Criteria:**
- [ ] Remove `"o"` from the keyboard shortcut case statement in `tui.go`
- [ ] Verify `"f2"` functionality remains intact
- [ ] Code compiles without errors

### Task 2: Update Help Documentation
**Acceptance Criteria:**
- [ ] Update help text in `handleHelp()` to remove `"o"` reference
- [ ] Help text shows only `"F2"` for tab toggling
- [ ] Help command displays updated information correctly

### Task 3: Update External Documentation
**Acceptance Criteria:**
- [ ] Update `docs/hotkey-toggle.md` if it references the removed shortcut
- [ ] Ensure all documentation is consistent

### Task 4: Add Verification Tests
**Acceptance Criteria:**
- [ ] Add test to verify alphanumeric keys don't trigger shortcuts
- [ ] Test that typing commands with `"o"` works normally
- [ ] Verify `"f2"` still toggles tabs correctly

### Task 5: Manual Testing
**Acceptance Criteria:**
- [ ] Start kiro-krew TUI
- [ ] Type commands containing 'o' (e.g., "stop", "focus")  
- [ ] Verify typing works normally without unexpected tab switches
- [ ] Verify F2 still toggles between tabs
- [ ] Test other alphanumeric characters work normally in input

## Validation Commands

```bash
# Build and test the application
go build -o kiro-krew-test cmd/kiro-krew/main.go

# Start the TUI for manual testing
./kiro-krew-test

# Run the test suite to ensure no regressions
go test ./internal/tui/...

# Integration test for keyboard functionality
go test -run TestAlphanumericInput ./internal/tui/...
```

## Manual Test Scenarios

1. **Basic Typing Test:**
   ```
   # In kiro-krew TUI, type each command and verify no tab switching:
   kiro-krew> stop
   kiro-krew> focus
   kiro-krew> option
   kiro-krew> hello world
   ```

2. **Tab Toggle Test:**
   ```
   # Press F2 - should toggle tabs
   # Press 'o' - should only type 'o', no tab change
   ```

3. **Full Alphabet Test:**
   ```
   # Type the full alphabet to ensure no shortcuts interfere:
   kiro-krew> abcdefghijklmnopqrstuvwxyz
   ```

## Risk Assessment

**Low Risk Change:**
- Minimal code modification (removing one line)
- No architectural changes
- Existing F2 functionality provides same capability
- Easy to revert if issues arise

**Potential Issues:**
- User muscle memory for 'o' shortcut (mitigation: F2 provides same function)
- None of the existing functionality is lost

## Success Metrics

- [ ] Users can type all alphanumeric characters without triggering shortcuts
- [ ] F2 key continues to work for tab toggling  
- [ ] Help documentation accurately reflects available shortcuts
- [ ] No regressions in existing TUI functionality