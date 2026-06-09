# Design Specification: Workaround for Empty Subagent Responses with Artifact Checking

**Issue:** #77  
**Closes:** #77  
**Created:** 2026-06-09  
**Author:** Architect Agent  

## Problem Statement

Due to a bug in Kiro CLI, subagents occasionally return empty responses even when they have successfully completed their work and created the expected artifacts. This causes the krew-lead agent to incorrectly treat successful work as failures and trigger unnecessary retry escalation through the four-stage retry process.

## Solution Approach

### High-Level Strategy
1. **Agent Configuration Enhancement**: Add `expectedArtifacts` field to each agent JSON configuration file
2. **Artifact Detection Logic**: Implement artifact checking in krew-lead retry logic to detect successful completion despite empty responses  
3. **Temporary Workaround Implementation**: Clearly mark all workaround code for future removal when the Kiro CLI bug is fixed
4. **Preserve Existing Logic**: Maintain all current retry escalation behavior for legitimate failures

### Architectural Decisions
- **Artifact Knowledge Distribution**: Each agent specifies its own expected artifacts rather than centralizing in krew-lead
- **Non-Breaking Implementation**: Workaround activates only for empty responses, preserving normal failure handling
- **Clear Separation**: All workaround code is isolated and marked with clear comments for easy removal

## Relevant Files

### Files to Modify
- `.kiro/agents/architect.json` - Add expectedArtifacts for design specifications
- `.kiro/agents/builder.json` - Add expectedArtifacts for implementation artifacts  
- `.kiro/agents/validator.json` - Add expectedArtifacts for validation reports
- `.kiro/agents/documenter.json` - Add expectedArtifacts for documentation files
- `.kiro/agents/krew-lead-prompt.md` - Implement artifact checking logic in retry stages

### Files Not Modified
- `.kiro/agents/planner.json` - Planner is interactive and not part of automated krew workflow
- `.kiro/agents/krew-lead.json` - No configuration changes needed

## Agent Configuration Updates

### Expected Artifacts by Agent

**Architect Agent**
```json
{
  "name": "architect",
  "expectedArtifacts": [".kiro-krew/specs/issue-*-*.md"],
  "prompt": "file://./architect-prompt.md",
  "tools": ["read", "write", "shell"],
  "allowedTools": ["read", "write", "shell"],
  "model": "claude-sonnet-4"
}
```

**Builder Agent**
```json
{
  "name": "builder", 
  "expectedArtifacts": ["**/*.go", "**/*.js", "**/*.ts", "**/*.py", "**/*.md", "go.mod", "package.json", "Makefile"],
  "description": "Focused engineering agent that executes ONE task at a time. Builds, implements, creates.",
  "prompt": "file://./builder-prompt.md",
  "tools": ["read", "write", "shell"],
  "allowedTools": ["read", "write", "shell"],
  "model": "claude-sonnet-4",
  "welcomeMessage": "Builder ready. What task should I execute?"
}
```

**Validator Agent**
```json
{
  "name": "validator",
  "expectedArtifacts": [".kiro-krew/validation-*.md", "test-results-*.txt"],
  "description": "Read-only validation agent that verifies task completion against acceptance criteria.",
  "prompt": "file://./validator-prompt.md", 
  "tools": ["read", "shell"],
  "allowedTools": ["read", "shell"],
  "toolsSettings": {
    "shell": {
      "autoAllowReadonly": true
    }
  },
  "model": "claude-sonnet-4",
  "welcomeMessage": "Validator ready. What should I verify?"
}
```

**Documenter Agent**
```json
{
  "name": "documenter",
  "expectedArtifacts": ["README.md", "docs/**/*.md", "**/*_README.md"],
  "description": "Generates documentation for completed and validated features.",
  "prompt": "file://./documenter-prompt.md",
  "tools": ["read", "write"],
  "allowedTools": ["read", "write"],
  "model": "claude-sonnet-4",
  "welcomeMessage": "Documenter ready. What should I document?"
}
```

## Krew-Lead Logic Enhancement

### Artifact Checking Implementation

Add the following logic to krew-lead-prompt.md after the existing retry policy section:

```markdown
### TEMPORARY WORKAROUND: Empty Response Artifact Detection
<!-- TODO: Remove this section when Kiro CLI empty response bug is fixed -->

When a subagent returns an empty response, before escalating to the next retry stage:

1. **Load Agent Configuration**: Read the agent's JSON file to get `expectedArtifacts` patterns
2. **Check for Artifacts**: Use shell commands to check if any expected artifacts exist:
   ```bash
   # Check if any files match the patterns
   for pattern in "${expectedArtifacts[@]}"; do
     if ls $pattern 1> /dev/null 2>&1; then
       echo "Found artifacts matching: $pattern"
       ARTIFACTS_FOUND=true
       break
     fi
   done
   ```
3. **Handle Detection Results**:
   - **If artifacts found**: Log incident but continue workflow normally
   - **If no artifacts found**: Proceed with normal retry escalation

### Incident Logging for Workaround
When artifacts are detected despite empty response:
- Log to incident system: "Empty response detected but artifacts found for agent {name}"
- Include artifact patterns matched and file list
- Tag incident with "empty-response-workaround" label
- Continue workflow without retry escalation
```

## Step-by-Step Task Breakdown

### Task 1: Update Agent Configuration Files
**Acceptance Criteria:**
- All agent JSON files contain `expectedArtifacts` field with appropriate patterns
- Patterns cover the primary artifacts each agent produces
- JSON syntax is valid and parseable

**Validation Commands:**
```bash
# Validate JSON syntax
jq . .kiro/agents/architect.json
jq . .kiro/agents/builder.json  
jq . .kiro/agents/validator.json
jq . .kiro/agents/documenter.json

# Verify expectedArtifacts field exists
jq '.expectedArtifacts' .kiro/agents/architect.json
```

### Task 2: Implement Artifact Checking Logic
**Acceptance Criteria:**
- Krew-lead prompt contains artifact checking logic
- Workaround code is clearly marked as temporary
- Logic only activates for empty responses
- Existing retry escalation preserved for legitimate failures

**Validation Commands:**
```bash
# Verify krew-lead prompt contains workaround section
grep -A 10 "TEMPORARY WORKAROUND" .kiro/agents/krew-lead-prompt.md

# Test pattern matching works
ls .kiro-krew/specs/issue-*-*.md 2>/dev/null && echo "Architect patterns work"
```

### Task 3: Test Workaround Implementation  
**Acceptance Criteria:**
- Workaround correctly identifies when artifacts exist
- Normal retry behavior preserved for actual failures
- Incident logging captures empty response events
- No breaking changes to existing workflow

**Validation Commands:**
```bash
# Create test artifacts and verify detection
mkdir -p .kiro-krew/specs
touch .kiro-krew/specs/issue-77-test.md
ls .kiro-krew/specs/issue-*-*.md && echo "Test artifact detected"

# Clean up test
rm .kiro-krew/specs/issue-77-test.md
```

## Team Orchestration

### Implementation Sequence
1. **Builder Agent**: Update all agent JSON configuration files with expectedArtifacts fields
2. **Builder Agent**: Implement artifact checking logic in krew-lead-prompt.md  
3. **Validator Agent**: Verify all configurations are valid and logic is correctly implemented
4. **Builder Agent**: Create unit tests or validation scripts to verify workaround behavior

### Communication Protocol
- All workaround code must include comments referencing issue #77
- Mark all sections with "TEMPORARY WORKAROUND" for easy identification
- Document artifact patterns clearly in agent configurations

## Validation Commands

### Pre-Implementation Validation
```bash
# Verify current agent configurations are valid
for agent in architect builder validator documenter; do
  jq . ".kiro/agents/$agent.json" > /dev/null && echo "$agent.json is valid"
done

# Check krew-lead prompt exists
test -f .kiro/agents/krew-lead-prompt.md && echo "krew-lead-prompt.md exists"
```

### Post-Implementation Validation  
```bash
# Verify expectedArtifacts fields are present
for agent in architect builder validator documenter; do
  jq -e '.expectedArtifacts' ".kiro/agents/$agent.json" && echo "$agent has expectedArtifacts"
done

# Test artifact detection patterns
mkdir -p .kiro-krew/specs test-dir
touch .kiro-krew/specs/issue-test-example.md
touch test-dir/example.go

# Test architect pattern
ls .kiro-krew/specs/issue-*-*.md && echo "✓ Architect pattern works"

# Test builder patterns  
ls **/*.go 2>/dev/null && echo "✓ Builder pattern works"

# Cleanup
rm -rf test-dir .kiro-krew/specs/issue-test-example.md
```

### Runtime Validation
```bash
# Verify workaround logic is present in krew-lead
grep -q "TEMPORARY WORKAROUND" .kiro/agents/krew-lead-prompt.md && echo "✓ Workaround implemented"

# Check for proper marking of temporary code
grep -q "TODO: Remove.*Kiro CLI.*bug" .kiro/agents/krew-lead-prompt.md && echo "✓ Removal instructions present"
```

## Risk Assessment

### Low Risk
- Agent JSON configuration changes (easily reversible)
- Adding artifact patterns (non-breaking addition)

### Medium Risk  
- Modifying krew-lead retry logic (could affect failure handling)
- File pattern matching (could have false positives/negatives)

### Mitigation Strategies
- Preserve all existing retry logic as fallback
- Use conservative artifact patterns to avoid false positives
- Clear marking of workaround code for easy removal
- Comprehensive validation before deployment

## Future Removal Plan

When the Kiro CLI empty response bug is fixed:

1. **Remove Workaround Code**: Delete all sections marked "TEMPORARY WORKAROUND" from krew-lead-prompt.md
2. **Clean Agent Configs**: Remove `expectedArtifacts` fields from all agent JSON files  
3. **Update Documentation**: Remove references to empty response handling from workflow documentation
4. **Test Cleanup**: Verify normal retry escalation still works correctly

Search pattern for removal:
```bash
grep -r "TEMPORARY WORKAROUND\|TODO.*Kiro CLI.*bug\|expectedArtifacts" .kiro/
```