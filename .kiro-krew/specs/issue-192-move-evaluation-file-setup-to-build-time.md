# Design Specification: Move Evaluation File Setup from Runtime to Build-time in Dockerfile

**Issue**: #192 - Move evaluation file setup from runtime to build-time in Dockerfile  
**Closes**: #192

## Problem Analysis

The current evaluation framework performs file operations at runtime that should happen at Docker build time, causing permission errors and unnecessary complexity:

### Current Issues:
- **Permission Errors**: Runtime file operations fail with "Permission denied" errors in containers
- **Runtime Complexity**: `SetupGitHubMocking()` and `ConfigureMockGitHubPath()` functions perform file copying at runtime
- **Performance Impact**: Each container startup requires file setup operations
- **Reliability Issues**: Runtime operations can fail unpredictably

### Current Architecture:
```
Container Start → Runtime Setup Functions → File Operations → Permission Errors
    ↓                     ↓                      ↓                    ↓
Container.Create() → SetupGitHubMocking() → Copy mock files → mkdir: Permission denied
                  → ConfigureMockGitHubPath() → PATH setup
```

## Solution Approach

Transform the evaluation framework to embed all required files at Docker build time, creating self-contained, immutable containers.

### Target Architecture:
```
Dockerfile Build → Embed Files → Immutable Container → Direct Execution
       ↓               ↓              ↓                    ↓  
  COPY commands → All files baked in → No runtime setup → Fast startup
```

## Relevant Files

### Files to Modify:
1. **`internal/eval/sandbox/container.go`** - Dockerfile generation logic
2. **`internal/eval/sandbox/mock_github.go`** - Remove runtime functions
3. **`internal/eval/runner.go`** - Remove runtime setup calls
4. **`internal/eval/dockerfile/base.Dockerfile`** - Add COPY commands

### Files to Reference:
1. **`internal/eval/sandbox/testdata/github-cli-mock/`** - Mock files to embed
2. **`.kiro/agents/`** - Agent configurations to embed  
3. **`.kiro-krew/evals/`** - Evaluation rubrics and test cases to embed
4. **`internal/eval/sandbox/installation_test.go`** - Test updates needed

### New Files to Create:
1. **`internal/eval/dockerfile/evaluation.Dockerfile`** - New build-time template
2. **Build context preparation scripts** - For copying files to build context

## Team Orchestration

This is a focused architectural change that can be implemented by a single developer with the following coordination points:

### Dependencies:
- **Dockerfile Templates**: Update base templates to include COPY operations
- **Build Context**: Ensure all required files are available in Docker build context
- **Testing**: Verify all evaluation tests pass without runtime setup

### Integration Points:
- **Image Manager**: Update to handle new build-time file embedding
- **Debug Mode**: Ensure generated Dockerfiles include proper file paths
- **CI/CD**: Verify binary distribution compatibility

## Step-by-Step Task Breakdown

### Task 1: Update Dockerfile Generation Logic
**File**: `internal/eval/sandbox/container.go`  
**Acceptance Criteria**:
- [ ] `GenerateDockerfileWithPlatform()` includes COPY commands for all required files
- [ ] Files are copied from embedded filesystem or build context
- [ ] Generated Dockerfile creates all necessary directories with proper permissions
- [ ] kiro-krew binary, agent configs, mock files, and evaluation files are embedded

**Implementation Details**:
```dockerfile
# Add to generated Dockerfile:
COPY --chown=sandbox:sandbox kiro-krew-binary /usr/local/bin/kiro-krew
COPY --chown=sandbox:sandbox .kiro/ /workspace/.kiro/
COPY --chown=sandbox:sandbox github-cli-mock/ /workspace/.kiro/skills/github-cli/
COPY --chown=sandbox:sandbox .kiro-krew/evals/ /workspace/.kiro-krew/evals/
RUN mkdir -p /workspace/.kiro/skills && chown -R sandbox:sandbox /workspace/.kiro
ENV PATH="/workspace/.kiro/skills/github-cli:$PATH"
```

### Task 2: Create Build Context Preparation
**File**: `internal/eval/sandbox/build_context.go` (new)  
**Acceptance Criteria**:
- [ ] Function to prepare build context with all required files
- [ ] Handle embedded filesystems (`//go:embed` directives)
- [ ] Support binary distribution (files from installed locations)
- [ ] Create temporary build context directory structure

**Implementation Details**:
```go
type BuildContext struct {
    TempDir string
    Files   map[string][]byte
}

func PrepareBuildContext(krewBinary string) (*BuildContext, error)
func (bc *BuildContext) AddAgentConfigs() error
func (bc *BuildContext) AddMockFiles() error  
func (bc *BuildContext) AddEvaluationFiles() error
func (bc *BuildContext) Cleanup() error
```

### Task 3: Remove Runtime Setup Functions
**File**: `internal/eval/sandbox/mock_github.go`  
**Acceptance Criteria**:
- [ ] Remove `SetupGitHubMocking()` function entirely
- [ ] Remove `ConfigureMockGitHubPath()` function entirely
- [ ] Keep embedded filesystem for build-time use
- [ ] Update function documentation to reflect build-time approach

### Task 4: Update Runner to Remove Runtime Calls
**File**: `internal/eval/runner.go`  
**Acceptance Criteria**:
- [ ] Remove calls to `SetupGitHubMocking()` in `invokeAgent()`
- [ ] Remove calls to `ConfigureMockGitHubPath()` in `invokeAgent()`  
- [ ] Remove MockGitHub configuration checks
- [ ] Update error handling to remove mocking-related error paths

**Location**: Around line 704 in `invokeAgent()` function

### Task 5: Update Base Dockerfile Template
**File**: `internal/eval/dockerfile/base.Dockerfile`  
**Acceptance Criteria**:
- [ ] Include directory creation for `.kiro` and `.kiro-krew` paths
- [ ] Set proper ownership and permissions for sandbox user
- [ ] Add environment variables for PATH configuration
- [ ] Maintain Alpine Linux base and security practices

### Task 6: Embed File Resources  
**Files**: Multiple locations  
**Acceptance Criteria**:
- [ ] Add `//go:embed` directive for agent configurations
- [ ] Add `//go:embed` directive for evaluation files  
- [ ] Update existing GitHub mock embed to work with build context
- [ ] Handle kiro-krew binary embedding or build context inclusion

### Task 7: Update Integration Tests
**File**: `internal/eval/sandbox/installation_test.go`  
**Acceptance Criteria**:
- [ ] Remove tests for `SetupGitHubMocking()` 
- [ ] Remove tests for `ConfigureMockGitHubPath()`
- [ ] Add tests for build-time file embedding
- [ ] Verify containers start with all files present
- [ ] Test PATH configuration works correctly

### Task 8: Update Container Build Process
**File**: `internal/eval/sandbox/container.go`  
**Acceptance Criteria**:
- [ ] `BuildImageFromDockerfile()` prepares build context with all files
- [ ] Build context includes kiro-krew binary from current installation
- [ ] Temporary build context is cleaned up after image build
- [ ] Image Manager compatibility maintained

## Validation Commands

### Build and Test Commands:
```bash
# Build evaluation images
go run ./cmd/kiro-krew eval --agent architect --case basic-spec-generation

# Verify container contents
docker run --rm <image-name> ls -la /workspace/.kiro/agents/
docker run --rm <image-name> ls -la /workspace/.kiro/skills/github-cli/
docker run --rm <image-name> ls -la /workspace/.kiro-krew/evals/

# Test GitHub mocking
docker run --rm <image-name> /workspace/.kiro/skills/github-cli/gh --help

# Test kiro-krew binary
docker run --rm <image-name> kiro-krew --version

# Run full evaluation suite  
go test ./internal/eval/... -v

# Test binary distribution compatibility
# 1. Build binary: go build ./cmd/kiro-krew
# 2. Move to temp location outside source tree
# 3. Run evaluations from that location
```

### Verification Checklist:
- [ ] All evaluation tests pass without permission errors
- [ ] Containers start faster (no runtime file setup)  
- [ ] GitHub CLI mock works correctly from embedded files
- [ ] Agent configurations loaded from container filesystem
- [ ] Evaluation rubrics and cases accessible from container
- [ ] Works with binary distribution (no source code access)
- [ ] Docker images follow best practices (proper layering, caching)

## Success Metrics

1. **Zero Permission Errors**: No "Permission denied" errors during evaluation runs
2. **Faster Startup**: Container startup time reduced by eliminating runtime file operations  
3. **Immutable Containers**: All required files baked into image at build time
4. **Binary Distribution Compatible**: Works when kiro-krew distributed as standalone binary
5. **Test Suite Passes**: All existing evaluation tests continue to pass

## Constraints Addressed

- **Binary Distribution**: Solution works without source code access by detecting kiro-krew binary location
- **Immutable Containers**: All files embedded at build time, no runtime modifications
- **Docker Best Practices**: Proper layer optimization, security, and caching
- **Backward Compatibility**: Existing evaluation test cases continue to work

## Risk Mitigation

1. **Image Size**: Use multi-stage builds and .dockerignore to optimize image size
2. **Build Context Size**: Only copy necessary files, exclude large directories like `.git`
3. **File Permissions**: Set proper ownership and permissions in Dockerfile
4. **Cross-platform**: Ensure embedded files work across different host architectures