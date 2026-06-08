# Test Plan for Issue #59: Agent View Reopening from Status Dialog

## Test Implementation Validation

The following changes have been implemented:

### 1. Enhanced Status Dialog (commands.go)
- ✅ Modified `handleStatus()` to display numbered running agents (1-9)
- ✅ Added "Running Agents (press number to open view):" section
- ✅ Added instructional text for keyboard navigation
- ✅ Separated running agents from stopped agents visually

### 2. Agent Selection Logic (tui.go)
- ✅ Added number key handling (1-9) when status overlay is active
- ✅ Validates agent selection bounds (0-8 for 1-9 keys)
- ✅ Filters only running agents for selection
- ✅ Validates agent is still running before tab creation
- ✅ Provides user feedback for invalid selections

### 3. Tab Management Enhancement (tab_manager.go)
- ✅ Added `HasAgentTab()` method to check for existing tabs
- ✅ Enhanced `FindTabByAgentID()` for better agent tab identification

### 4. Integration Features
- ✅ Status dialog closes immediately upon agent selection
- ✅ Existing agent tabs are focused if already open
- ✅ New agent tabs are created if none exist
- ✅ Current log state is preserved in restored views
- ✅ Agent state validation prevents errors for stopped agents

## Manual Testing Instructions

To test this implementation:

1. **Start kiro-krew with agents:**
   ```bash
   ./kiro-krew
   # In TUI, run: watch start
   # Wait for some agents to start
   ```

2. **Test status dialog interactivity:**
   ```bash
   # In TUI, run: status
   # Verify numbered list appears: "1. Issue #X: Title (running, Xs)"
   # Press number keys 1-9 to select agents
   # Verify agent tab opens/focuses and status dialog closes
   ```

3. **Test edge cases:**
   - Press numbers for non-existent agents (should be ignored)
   - Select agent that has stopped between status display and selection
   - Select agent with existing tab (should focus existing tab)
   - Test with 0 running agents (should show stopped agents only)

## Implementation Notes

- Number key selection only works when status overlay is active
- Supports up to 9 running agents (limitation by design for usability)
- Agent validation ensures robustness against race conditions
- Preserves all existing TUI functionality and keyboard shortcuts
- Uses existing tab management infrastructure for consistency