# Replace Alphanumeric Shortcut Keys - Issue #62

Closes #62

## Solution Approach

Replace the conflicting alphanumeric shortcut key `o` with a non-conflicting alternative that preserves the tab toggle functionality while allowing normal REPL typing. The solution uses `Tab` key as the replacement, which is intuitive for tab navigation and avoids conflicts with alphanumeric input.

### Key Design Decisions

1. **Primary Replacement**: Replace `o` with `Tab` key for main/agent tab toggle
2. **Preserve F2**: Keep existing F2 functionality as secondary option  
3. **Safety Review**: Audit all alphanumeric shortcuts in agent output view
4. **Documentation Update**: Update help text and status commands

## Relevant Files

### Files to Modify
- `internal/tui/tui.go` - Main shortcut key handling (primary change)
- `internal/tui/commands.go` - Help text and status display updates
- `internal/tui/output_view.go` - Review vim-style navigation keys for conflicts

### Files for Reference
- `internal/tui/tab_manager.go` - Understanding ToggleView() functionality
- `internal/hotkey/detector.go` - Existing modifier key patterns
- `internal/tui/styles.go` - Style definitions for consistency

## Team Orchestration

This is a focused UI/UX change that can be implemented independently:
- **Frontend Team**: Handle TUI shortcut modifications and help text updates
- **Testing Team**: Verify REPL input functionality and shortcut behavior
- **Documentation Team**: Review any external docs referencing shortcuts (if applicable)

No backend or API changes required. No coordination with external systems needed.

## Step-by-Step Task Breakdown

### Task 1: Update Main Shortcut Handler
**File**: `internal/tui/tui.go`
**Location**: Around line 355 in `Update()` method  
**Action**: Replace `"f2", "o"` with `"f2", "tab"`
```go
// Before
case "f2", "o":
    // Toggle between main and agent tabs
    m.tabManager.ToggleView()
    return m, nil

// After  
case "f2", "tab":
    // Toggle between main and agent tabs
    m.tabManager.ToggleView()
    return m, nil
```
**Acceptance Criteria**:
- Tab key toggles between main and agent tabs
- F2 functionality remains unchanged
- `o` key no longer triggers tab toggle

### Task 2: Update Help Documentation
**File**: `internal/tui/commands.go`
**Location**: `handleHelp()` function around line 135
**Action**: Update hotkey description
```go
// Before
"  F2, o          - Toggle between console and agent output views",

// After
"  F2, Tab        - Toggle between console and agent output views",
```
**Acceptance Criteria**:
- Help overlay shows correct shortcut keys
- No references to `o` key remain in help text

### Task 3: Update Status Command Documentation  
**File**: `internal/tui/commands.go`
**Location**: `handleStatus()` function around line 80
**Action**: Update navigation help text
```go
// Before
content = append(content, "  F2 - Toggle between main and first agent tab")

// After
content = append(content, "  F2/Tab - Toggle between main and first agent tab")
```
**Acceptance Criteria**:
- Status overlay shows updated shortcut information
- Both F2 and Tab are documented as alternatives

### Task 4: Review Agent Output View Shortcuts
**File**: `internal/tui/output_view.go` 
**Location**: `Update()` method around line 40
**Action**: Review existing vim-style shortcuts (`k`, `j`, `g`, `G`)
**Analysis Required**:
- Confirm these shortcuts only apply in agent tab context
- Verify they don't interfere with REPL input in main tab
- Assess if any conflict with common terminal usage

**Acceptance Criteria**:
- Document that vim shortcuts are tab-context-specific
- Confirm no alphanumeric conflicts in main console tab
- All navigation works as expected per tab context

### Task 5: Comprehensive Testing
**Action**: Create test scenarios for shortcut conflicts
**Test Cases**:
1. Type `o` in REPL - should input character, not toggle tabs
2. Type other alphanumeric characters - should work normally in REPL
3. Press Tab - should toggle between main/agent tabs  
4. Press F2 - should toggle between main/agent tabs
5. In agent tab: test vim navigation (`k`, `j`, `g`, `G`) works
6. Verify `[` and `]` tab navigation still works
7. Test all functionality with no agent tabs present

**Acceptance Criteria**:
- All alphanumeric keys work normally in REPL input
- Tab navigation shortcuts function correctly
- Agent tab vim shortcuts don't leak to main tab
- No regression in existing functionality

## Validation Commands

```bash
# Build the application
make build

# Start kiro-krew and test shortcuts
./kiro-krew

# In the TUI, test each scenario:
# 1. Type 'o' - should appear in input, not toggle tabs
# 2. Press Tab - should toggle to agent tab (if available)  
# 3. Press F2 - should toggle back to main tab
# 4. Type 'help' - verify documentation shows correct shortcuts
# 5. Type 'status' - verify status shows correct navigation help

# Test vim navigation in agent tab (if agents running):
# 1. Press Tab to switch to agent tab
# 2. Press 'k' and 'j' - should scroll content
# 3. Press 'g' and 'G' - should go to top/bottom
# 4. Switch back to main tab - vim keys should not affect REPL

# Integration test - verify REPL functionality
# 1. Type commands containing 'o': "stop 1", "about" 
# 2. Verify all commands execute normally
# 3. Confirm no accidental tab switching during typing
```

## Implementation Notes

### Terminal Compatibility
- `Tab` key is universally supported across terminals
- No conflicts with common terminal control sequences
- Maintains consistency with tab navigation paradigms

### User Experience Impact  
- **Positive**: Eliminates typing interference in REPL
- **Positive**: More intuitive tab navigation (Tab key for tabs)
- **Neutral**: Users must learn new shortcut, but F2 remains as fallback
- **Mitigation**: Clear documentation in help and status commands

### Edge Cases Handled
- No agent tabs present - Tab/F2 shortcuts should be no-op or graceful
- Multiple agent tabs - existing ToggleView() logic handles this correctly  
- Tab key in agent output view - should maintain scroll navigation, not toggle tabs
- Overlay active - Tab key should be blocked like other shortcuts

### Future Considerations
- This change creates precedent for non-alphanumeric shortcuts
- Consider standardizing all shortcuts to avoid similar conflicts
- Tab key could potentially be extended for other navigation features