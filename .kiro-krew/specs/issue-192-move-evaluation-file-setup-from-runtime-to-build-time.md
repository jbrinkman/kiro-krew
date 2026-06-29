# Design Specification: Move Evaluation File Setup from Runtime to Build-time in Dockerfile

**Issue**: #192 - Move evaluation file setup from runtime to build-time in Dockerfile

Closes #192

## Problem Analysis

The evaluation framework currently performs file operations at runtime that should be done at build-time, causing permission errors and violating Docker best practices:

### Current Runtime Operations (Problematic)
1. **GitHub CLI Mock Files**: `SetupGitHubMocking()` copies mock files at runtime
2. **Agent Configurations**: Read from host `.kiro/` at runtime
3. **Evaluation Data**: Test cases and rubrics read from host `.kiro-krew/evals/` at runtime
4. **Binary Dependencies**: kiro-krew binary not included in container

### Permission Error Root Cause
```
mkdir: can't create directory '/workspace/.kiro/': Permission denied
```

This occurs because `SetupGitHubMocking()` tries to create directories in the container filesystem at runtime, but the non-root `sandbox` user lacks write permissions to create the required directory structure.

## Solution Approach

Transform the evaluation framework from runtime-dependent to build-time self-contained by embedding all required files during Docker image construction.

### Architecture Changes

1. **Build-time File Embedding**: Copy all necessary files into Docker images during build
2. **Runtime Function Elimination**: Remove `SetupGitHubMocking()` and `ConfigureMockGitHubPath()` entirely
3. **Self-contained Images**: Images include all dependencies and configurations
4. **Immutable Containers**: No runtime filesystem modifications required

## Relevant Files

### Files to Modify
- `internal/eval/sandbox/container.go` - Update `GenerateDockerfileWithPlatform()` to include build-time COPY operations
- `internal/eval/runner.go` - Remove calls to `SetupGitHubMocking()` and `ConfigureMockGitHubPath()` 
- `internal/eval/sandbox/mock_github.go` - Delete entire file (functions no longer needed)

### Files to Copy at Build-time
- `internal/eval/sandbox/testdata/github-cli-mock/` → `/workspace/.kiro/skills/github-cli/`
- `.kiro/agents/` → `/workspace/.kiro/agents/`
- `.kiro-krew/evals/` → `/workspace/.kiro-krew/evals/`
- `kiro-krew` binary → `/usr/local/bin/kiro-krew`

### Test Files to Update
- `internal/eval/sandbox/installation_test.go` - Remove `SetupGitHubMocking()` test calls
- All test files using these functions need runtime setup removal

## Team Orchestration

### Builder Tasks
1. **Phase 1**: Update Dockerfile generation to include build-time COPY operations
2. **Phase 2**: Remove runtime setup function calls from runner.go
3. **Phase 3**: Delete obsolete mock_github.go file
4. **Phase 4**: Update tests to expect pre-configured containers

### Validator Tasks
- Verify containers start without permission errors
- Confirm all evaluation tests pass
- Check that containers are self-contained (no host dependencies)
- Validate faster container startup (no runtime setup)

## Step-by-Step Task Breakdown

### Task 1: Update Dockerfile Generation for Build-time File Copying
**File**: `internal/eval/sandbox/container.go`
**Function**: `GenerateDockerfileWithPlatform()`

**Changes Needed**:
- Add COPY commands for GitHub CLI mock files
- Add COPY commands for agent configurations 
- Add COPY commands for evaluation data
- Set proper permissions and PATH configuration
- Include kiro-krew binary if available

**Acceptance Criteria**:
- Generated Dockerfiles contain appropriate COPY commands
- Mock GitHub CLI files copied to correct container path
- Agent configs copied to `/workspace/.kiro/agents/`
- Evaluation data copied to `/workspace/.kiro-krew/evals/`
- Container PATH includes `/workspace/.kiro/skills/github-cli`

### Task 2: Remove Runtime Setup Function Calls
**File**: `internal/eval/runner.go`
**Lines**: ~704-710 in `invokeAgentInContainer()`

**Changes Needed**:
- Remove `SetupGitHubMocking()` call and error handling
- Remove `ConfigureMockGitHubPath()` call and error handling
- Keep MockGitHub configuration flag for conditional logic
- Update timing logs to reflect removed setup

**Acceptance Criteria**:
- No calls to `SetupGitHubMocking()` or `ConfigureMockGitHubPath()`
- Container execution proceeds directly to agent invocation
- MockGitHub flag still controls build-time behavior
- Faster container startup due to eliminated runtime setup

### Task 3: Delete Obsolete Mock GitHub File
**File**: `internal/eval/sandbox/mock_github.go`

**Changes Needed**:
- Delete entire file
- Remove associated tests that specifically test these functions

**Acceptance Criteria**:
- File no longer exists in codebase
- No compilation errors from missing function references
- Related unit tests removed or updated

### Task 4: Update Tests for Pre-configured Containers
**Files**: 
- `internal/eval/sandbox/installation_test.go`
- Any other test files calling removed functions

**Changes Needed**:
- Remove test calls to `SetupGitHubMocking()`
- Update tests to expect pre-configured containers
- Modify assertions to check for build-time file presence
- Update test container creation to use new build process

**Acceptance Criteria**:
- All tests pass without calling removed functions
- Tests verify that required files exist in built images
- Test containers start successfully without runtime setup
- No permission error tests (problem eliminated)

## Validation Commands

### Build Verification
```bash
# Verify Dockerfile generation includes COPY commands
go run ./cmd/kiro-krew eval --debug architect --test-case sample 2>&1 | grep -E "COPY|ADD"

# Verify image builds successfully
docker build -f /tmp/generated-dockerfile -t test-eval .

# Check files exist in built image
docker run --rm test-eval ls -la /workspace/.kiro/skills/github-cli
docker run --rm test-eval ls -la /workspace/.kiro/agents
docker run --rm test-eval ls -la /workspace/.kiro-krew/evals
```

### Runtime Verification
```bash
# Verify no permission errors during evaluation
go run ./cmd/kiro-krew eval architect --test-case sample

# Verify faster startup (timing should be shorter)
go run ./cmd/kiro-krew eval --debug architect --test-case sample 2>&1 | grep "Container setup:"

# Run full evaluation test suite
go test ./internal/eval/... -v
```

### Integration Testing
```bash
# Verify end-to-end evaluation works
go run ./cmd/kiro-krew eval builder --test-case simple-implementation

# Verify mocked GitHub CLI is available
docker run --rm test-eval which gh
docker run --rm test-eval gh --version

# Verify agent configurations are accessible
docker run --rm test-eval ls /workspace/.kiro/agents/
```

## Constraints Compliance

- ✅ **Binary Distribution**: Build-time copying works whether kiro-krew is built from source or distributed as binary
- ✅ **Immutable Images**: All files embedded at build-time, no runtime modifications
- ✅ **Docker Best Practices**: Follows layer optimization and build-time dependency management
- ✅ **Reproducible**: Same inputs produce identical container images
- ✅ **Self-contained**: No runtime dependencies on host filesystem

## Expected Outcomes

1. **Eliminated Permission Errors**: No more runtime directory creation failures
2. **Faster Container Startup**: Reduced setup time by eliminating runtime operations
3. **Improved Reliability**: Self-contained images with predictable behavior
4. **Better Maintainability**: Simplified codebase without complex runtime setup logic
5. **Docker Best Practices**: Build-time file operations following container conventions