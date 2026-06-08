# Design Specification: Durable Incident Logging System

**Issue:** Closes #65  
**Title:** Implement durable incident logging system outside worktrees  
**Created:** 2026-06-07T21:17:55.771-04:00

## Solution Approach

The current incident logging system writes files within temporary worktrees, causing them to appear in PRs. This design implements a durable logging system that:

1. **Stores logs outside worktrees** in `~/.kiro-krew/incidents/<project>` organized by repository
2. **Preserves complete incident reports** without modification or summarization
3. **Provides REPL integration** with `logs` command for viewing workflow incidents
4. **Ensures persistence** after worktree deletion
5. **Prevents PR contamination** by using user home directory storage

### Architecture Changes

- **Storage Location:** `~/.kiro-krew/incidents/<repo-name>/` (existing implementation is already correct)
- **File Format:** `incident-<issue>-<attempt>-<timestamp>.md` (existing format maintained)
- **REPL Command:** New `logs` command with interactive incident browser
- **Project Association:** Repository-based organization using git remote origin URL

## Relevant Files

### Files to Modify
- `internal/tui/commands.go` - Add `logs` command handler and incident browser UI
- `cmd/kiro-krew/cmd/logs.go` - New CLI command for incident log management

### Files Already Implemented (No Changes Needed)
- `internal/incidents/logger.go` - Core logging functionality (already stores outside worktrees)
- `internal/incidents/storage.go` - Storage operations (already correct)
- `internal/incidents/logger_test.go` - Unit tests (covers expected behavior)
- `cmd/kiro-krew/cmd/log_incident.go` - CLI logging command (works correctly)

### Files Referenced but Not Modified
- `internal/tui/tui.go` - TUI main loop (integration point for logs command)
- Various `.kiro-krew/specs/incidents/*.md` - Example incident formats to preserve

## Team Orchestration

### Single Component Task
This is a UI integration task that requires:
1. **TUI Team:** Add REPL command handler for `logs` 
2. **CLI Team:** Add standalone `kiro-krew logs` command
3. **Testing Team:** Validate incident browsing functionality

No inter-team dependencies exist as the storage layer is already implemented correctly.

## Step-by-Step Task Breakdown

### Task 1: Implement REPL Logs Command
**Acceptance Criteria:**
- User can type `logs` in REPL to view incident list
- Interactive interface shows: Issue #, Attempt, Timestamp, Title (first line)
- User can select incident to view full content
- Graceful handling when no incidents exist
- Project-specific filtering (current repository only)

**Implementation Details:**
- Add `handleLogs()` function to `internal/tui/commands.go`
- Create overlay UI for incident list browsing
- Integrate incident selection with content viewer
- Use existing `IncidentLogger.ListIncidents()` and `GetIncident()` methods

### Task 2: Implement CLI Logs Command
**Acceptance Criteria:**
- `kiro-krew logs` lists all incidents for current repository
- `kiro-krew logs --all` lists incidents across all projects
- `kiro-krew logs <issue-number>` shows incidents for specific issue
- `kiro-krew logs --content <file-path>` displays full incident content
- Proper error handling for missing directories/files

**Implementation Details:**
- Create `cmd/kiro-krew/cmd/logs.go`
- Register command with cobra in `init()` function
- Implement filtering and formatting options
- Use existing incident logger infrastructure

### Task 3: Integration Testing
**Acceptance Criteria:**
- REPL `logs` command works in active kiro-krew session
- CLI `kiro-krew logs` works from any directory within git repository
- Incidents persist after worktree deletion
- No incident files appear in git status or PRs
- Cross-project incident isolation verified

## Validation Commands

```bash
# Test durable storage location
ls -la ~/.kiro-krew/incidents/

# Verify repository-based organization  
ls -la ~/.kiro-krew/incidents/kiro-krew/

# Test REPL integration
echo "logs" | kiro-krew

# Test CLI commands
kiro-krew logs
kiro-krew logs --all
kiro-krew logs 65

# Verify no PR contamination
git status --porcelain

# Test persistence after worktree cleanup
# (Run after worktree deletion)
kiro-krew logs

# Test cross-repository isolation
cd /different/repo && kiro-krew logs
```

## Technical Implementation Notes

### REPL Command Implementation
```go
func (m model) handleLogs() (model, tea.Cmd) {
    logger, err := incidents.NewIncidentLogger()
    if err != nil {
        m = m.appendActivity(m.styles.Error.Render(fmt.Sprintf("Failed to initialize incident logger: %v", err)))
        return m, nil
    }
    
    incidents, err := logger.ListIncidents()
    if err != nil {
        m = m.appendActivity(m.styles.Error.Render(fmt.Sprintf("Failed to list incidents: %v", err)))
        return m, nil
    }
    
    if len(incidents) == 0 {
        m = m.appendActivity(m.styles.Warning.Render("No incidents found for this project"))
        return m, nil
    }
    
    // Create overlay with incident list
    content := []string{m.styles.Prompt.Render("Workflow Incidents")}
    for _, incident := range incidents {
        line := fmt.Sprintf("Issue #%d Attempt %d - %s", 
            incident.IssueNumber, 
            incident.Attempt, 
            incident.Timestamp.Format("2006-01-02 15:04"))
        content = append(content, line)
    }
    
    m = m.activateOverlay(overlayLogs, "Incident Log Browser", content)
    return m, nil
}
```

### CLI Command Structure
```go
var logsCmd = &cobra.Command{
    Use:   "logs [issue-number]",
    Short: "View workflow incident logs",
    Long:  "Browse and view workflow incident logs stored outside worktrees",
    Args:  cobra.MaximumNArgs(1),
    RunE:  runLogs,
}

func runLogs(cmd *cobra.Command, args []string) error {
    logger, err := incidents.NewIncidentLogger()
    if err != nil {
        return fmt.Errorf("failed to initialize incident logger: %w", err)
    }
    
    incidents, err := logger.ListIncidents()
    // Implementation details...
}
```

### Storage Verification
The existing `IncidentLogger` already implements correct durable storage:
- Uses `~/.kiro-krew/incidents/<repo-name>/` directory
- Extracts repository name from git remote origin URL
- Creates timestamped incident files outside worktree scope
- Provides listing and retrieval methods

No changes needed to storage layer - only UI integration required.

## Risk Mitigation

**Risk:** User confusion about where incidents are stored  
**Mitigation:** Clear documentation and helpful error messages showing storage location

**Risk:** Cross-project incident leakage  
**Mitigation:** Repository-based isolation already implemented via git remote origin detection

**Risk:** Disk space consumption from accumulated incidents  
**Mitigation:** Document cleanup procedures; consider future retention policies

**Risk:** Performance impact from large incident lists  
**Mitigation:** Implement pagination or filtering in UI if needed during validation

## Success Metrics

1. **Zero incident files in PRs** - Verified by git status checks
2. **Incident persistence** - Logs accessible after worktree deletion  
3. **Project isolation** - Each repository only sees its own incidents
4. **REPL integration** - `logs` command works within kiro-krew session
5. **CLI accessibility** - `kiro-krew logs` works from any repository directory
6. **Complete preservation** - Full incident report format maintained without modification