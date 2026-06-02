# Design Specification: Migrate CLI from Switch-Based Implementation to Cobra Framework

**Issue:** #36  
**Closes:** #36

## Solution Approach

Replace the current brittle switch-based CLI implementation in `cmd/kiro-krew/main.go` with the Cobra framework to leverage built-in CLI features and reduce maintenance overhead. The migration will preserve all existing functionality while eliminating manual help handling, case sensitivity management, and custom command routing.

### High-Level Strategy

1. **Preserve Default Behavior**: Maintain TUI launch when no commands are provided
2. **Command Migration**: Convert each switch case to a Cobra command with identical functionality  
3. **Leverage Cobra Features**: Use built-in help system, case-insensitive commands, and structured command hierarchy
4. **Clean Architecture**: Remove manual help structures and switch-based routing

### Architectural Decisions

- Use Cobra's `rootCmd.Run` to handle the no-arguments case (TUI launch)
- Create separate command files for better organization and maintainability
- Maintain the same command interface for backward compatibility
- Leverage Cobra's built-in help system instead of custom help structures

## Relevant Files

### Files to Modify
- `cmd/kiro-krew/main.go` - Complete refactoring to use Cobra
- `go.mod` - Add Cobra dependency

### Files to Create
- `cmd/kiro-krew/cmd/root.go` - Root command setup
- `cmd/kiro-krew/cmd/init.go` - Init command implementation
- `cmd/kiro-krew/cmd/update.go` - Update command implementation  
- `cmd/kiro-krew/cmd/eval.go` - Eval command with diff subcommand

### Dependencies Referenced
- `internal/agent` - Agent manager for TUI
- `internal/config` - Configuration loading
- `internal/eval` - Evaluation runner and diff functionality
- `internal/tui` - Terminal UI interface
- `internal/watcher` - File watcher

## Team Orchestration

This is a single-component refactoring that affects only the CLI entry point. No coordination with other teams required, but the migration must:

1. **Maintain API Compatibility**: All existing command invocations must work identically
2. **Preserve Behavior**: TUI launch, command functionality, and output formatting must remain unchanged
3. **Testing Strategy**: Verify each command works before and after migration

## Step-by-Step Task Breakdown

### Task 1: Add Cobra Dependency
**Acceptance Criteria:**
- [ ] Add `github.com/spf13/cobra` to go.mod
- [ ] Run `go mod tidy` to resolve dependencies

### Task 2: Create Root Command Structure
**Acceptance Criteria:**
- [ ] Create `cmd/kiro-krew/cmd/root.go` with root command setup
- [ ] Configure root command to launch TUI when no subcommands provided
- [ ] Set up case-insensitive command matching
- [ ] Add version information and basic description

### Task 3: Migrate Init Command
**Acceptance Criteria:**
- [ ] Create `cmd/kiro-krew/cmd/init.go` 
- [ ] Move `extractTemplates()` function to init command
- [ ] Preserve exact behavior: extract templates without overwrite
- [ ] Add proper Cobra command description and usage
- [ ] Test: `kiro-krew init` works identically to current implementation

### Task 4: Migrate Update Command  
**Acceptance Criteria:**
- [ ] Create `cmd/kiro-krew/cmd/update.go`
- [ ] Reuse `extractTemplates()` with force=true parameter
- [ ] Preserve exact behavior: force overwrite templates except config.yaml
- [ ] Add proper Cobra command description and usage
- [ ] Test: `kiro-krew update` works identically to current implementation

### Task 5: Migrate Eval Command with Diff Subcommand
**Acceptance Criteria:**
- [ ] Create `cmd/kiro-krew/cmd/eval.go` with main eval command
- [ ] Add `diff` subcommand under eval command
- [ ] Preserve exact behavior: `eval.Run(agent)` for main command
- [ ] Preserve exact behavior: `eval.Diff(runA, runB)` for diff subcommand
- [ ] Support optional agent parameter for eval command
- [ ] Test: Both `kiro-krew eval [agent]` and `kiro-krew eval diff <run-a> <run-b>` work identically

### Task 6: Refactor Main Function
**Acceptance Criteria:**
- [ ] Replace entire switch-based logic with `rootCmd.Execute()`
- [ ] Remove all manual help handling code
- [ ] Remove `helpData` map and `CommandHelp` struct
- [ ] Remove `showGeneralHelp()` and `showCommandHelp()` functions
- [ ] Preserve embedded templates functionality
- [ ] Test: All commands work with new structure

### Task 7: Cleanup and Validation
**Acceptance Criteria:**
- [ ] Remove all unused helper functions and structures
- [ ] Ensure no breaking changes to command interface
- [ ] Verify case-insensitive commands work
- [ ] Verify built-in help system works for all commands
- [ ] Test default behavior (TUI launch) when no arguments provided

## Validation Commands

Run these commands to verify the implementation works correctly:

### Basic Functionality Tests
```bash
# Test default behavior (should launch TUI)
kiro-krew

# Test help system
kiro-krew --help
kiro-krew -h
kiro-krew help

# Test command help
kiro-krew init --help
kiro-krew update --help  
kiro-krew eval --help
kiro-krew eval diff --help
```

### Command Functionality Tests
```bash
# Test init command
kiro-krew init

# Test update command  
kiro-krew update

# Test eval command
kiro-krew eval
kiro-krew eval some-agent

# Test eval diff command
kiro-krew eval diff run1 run2
```

### Case Insensitivity Tests
```bash
# Test case variations
kiro-krew INIT
kiro-krew Init
kiro-krew UPDATE
kiro-krew Update
kiro-krew EVAL
kiro-krew Eval
```

### Error Handling Tests
```bash
# Test invalid commands
kiro-krew invalid-command

# Test invalid eval diff usage
kiro-krew eval diff
kiro-krew eval diff run1
```

### Build and Module Tests
```bash
# Verify module integrity
go mod verify
go mod tidy

# Build verification
go build -o kiro-krew-test cmd/kiro-krew/main.go
./kiro-krew-test --help
rm kiro-krew-test
```

## Implementation Notes

### Cobra Command Structure
- Root command handles TUI launch (no arguments case)
- Each command maintains identical behavior to current switch cases
- Built-in help replaces custom help system
- Subcommands properly nested (eval diff under eval)

### Backward Compatibility
- All existing command invocations must work identically
- Command output and behavior preserved exactly
- Error messages should be equivalent or improved

### Code Organization
- Separate command files improve maintainability
- Shared template extraction logic can be refactored into helper
- Clean separation between CLI structure and business logic

This migration eliminates the brittle switch-based approach while preserving all existing functionality and improving maintainability through Cobra's built-in features.