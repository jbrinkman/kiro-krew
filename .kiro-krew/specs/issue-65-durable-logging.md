# Design Specification: Durable Incident Logging System

Closes #65

## Solution Approach

Implement a durable logging system that stores incident reports outside temporary worktrees in a persistent location. The system will:

1. Store logs in `~/.kiro-krew/logs/<project-name>/` directory structure
2. Create incident logs with standardized naming: `incident-<issue>-<timestamp>.md`
3. Add REPL command `logs` to list and view incident reports
4. Modify incident generation in krew-lead agent to use persistent storage

## Relevant Files

### Files to Create:
- `internal/logging/incident.go` - Core incident logging functionality
- `internal/logging/storage.go` - File storage operations for logs
- `internal/logging/viewer.go` - Log viewing and listing functionality

### Files to Modify:
- `internal/tui/commands.go` - Add `logs` command handler
- `internal/config/config.go` - Add logging configuration options
- `.kiro/agents/krew-lead-prompt.md` - Update incident report creation path
- `cmd/kiro-krew/templates/kiro/agents/krew-lead-prompt.md` - Update template

### Files Relevant for Integration:
- `internal/agent/manager.go` - Integration point for incident logging
- `internal/config/config.go` - Configuration management
- `.kiro-krew/config.yaml` - Configuration schema

## Team Orchestration

### Development Phases:
1. **Storage Layer** - Create logging infrastructure and file operations
2. **Agent Integration** - Modify incident reporting to use persistent storage
3. **REPL Commands** - Add user interface for viewing logs
4. **Configuration** - Add configuration options and validation

### Dependencies:
- Storage layer must be completed before agent integration
- REPL commands can be developed in parallel with agent integration
- Configuration changes should be backward compatible

## Step-by-Step Task Breakdown

### Task 1: Create Logging Infrastructure
**File**: `internal/logging/incident.go`
**Acceptance Criteria**:
- Define `IncidentReport` struct matching current markdown format
- Implement `WriteIncident(projectName, issueNumber, report)` function
- Store logs in `~/.kiro-krew/logs/<project-name>/` directory
- Use naming pattern: `incident-<issue>-<timestamp>.md`
- Ensure directory creation if not exists

### Task 2: Implement Storage Operations
**File**: `internal/logging/storage.go`
**Acceptance Criteria**:
- Function `ListIncidents(projectName)` returns chronological list
- Function `ReadIncident(projectName, filename)` returns report content
- Handle missing directories gracefully
- Support project-based organization

### Task 3: Create Log Viewer Component
**File**: `internal/logging/viewer.go`
**Acceptance Criteria**:
- Function `FormatIncidentList(incidents)` for REPL display
- Function `FormatIncidentDetails(report)` for full view
- Include metadata: issue number, timestamp, attempt count
- Truncate summaries for list view

### Task 4: Add REPL Commands
**File**: `internal/tui/commands.go`
**Acceptance Criteria**:
- Add `logs` command to list incidents for current project
- Add `logs <number>` to view specific incident by list index
- Add `logs clear` to clear all logs for current project (with confirmation)
- Update help text with new commands
- Handle empty log directories gracefully

### Task 5: Update Configuration
**File**: `internal/config/config.go`
**Acceptance Criteria**:
- Add `LogsDir` field with default `~/.kiro-krew/logs`
- Add `MaxIncidentAge` field with default `30d`
- Maintain backward compatibility
- Validate log directory permissions on startup

### Task 6: Modify Incident Report Generation
**File**: `.kiro/agents/krew-lead-prompt.md`
**Acceptance Criteria**:
- Change incident path from `specs/incidents/` to use logging API
- Preserve exact incident report format
- Include project context in logging calls
- Update both main template and cmd template

### Task 7: Integration with Agent Manager
**File**: `internal/agent/manager.go`
**Acceptance Criteria**:
- Add logging dependency to Manager struct
- Call incident logging when agents fail
- Extract project name from repository context
- Maintain existing error handling flow

## Validation Commands

```bash
# Test basic logging functionality
go test ./internal/logging/... -v

# Test REPL command integration
go test ./internal/tui/... -v -run TestCommands

# Test configuration loading
go test ./internal/config/... -v

# Integration test - create test incident
mkdir -p ~/.kiro-krew/logs/test-project
echo "# Test Incident" > ~/.kiro-krew/logs/test-project/incident-99-$(date +%s).md

# Test REPL commands in development build
go run cmd/kiro-krew/main.go
# In REPL: 
# > logs
# > logs 1
# > logs clear

# Verify logs don't appear in git status
git status --porcelain | grep -v ".kiro-krew/logs" || echo "✓ Logs excluded from git"
```

## Implementation Notes

### Data Flow:
1. Agent encounters failure → Manager detects → Calls logging API
2. Incident stored in `~/.kiro-krew/logs/<project>/incident-<issue>-<timestamp>.md`
3. User runs `logs` command → Lists available incidents
4. User selects incident → Full report displayed

### Error Handling:
- Log storage failures should not block agent execution
- Missing log directories created automatically  
- Graceful handling of permission issues
- Fallback to temporary files if persistent storage fails

### Security Considerations:
- Logs stored in user home directory (not system-wide)
- No sensitive data validation needed (existing incident format is safe)
- Standard file permissions (user read/write only)

### Performance:
- Lazy loading of incident lists
- File-based storage suitable for expected volume
- No database required for this scope