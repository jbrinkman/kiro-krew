---
name: builder-conventions
description: Project-specific conventions and patterns for the builder agent. Maintains synchronization between template and live project files.
---

# Builder Conventions

Project-specific conventions, patterns, and best practices for the builder agent to follow during implementation tasks.

## Mandatory Template Synchronization

**Critical for Self-Hosting**: Kiro Krew templates must stay synchronized with live project files. When modifying any template-synchronized content, BOTH versions must be updated using the exact commands below.

### Complete Sync Workflow Commands

```bash
# Agent files (all files sync - JSON configs and prompt files)
cp .kiro/agents/*.json cmd/kiro-krew/templates/kiro/agents/
cp .kiro/agents/*.md cmd/kiro-krew/templates/kiro/agents/

# Scripts (all files sync - shell scripts)
cp .kiro-krew/scripts/*.sh cmd/kiro-krew/templates/kiro-krew/scripts/

# Themes (all files sync - YAML theme configurations)
cp .kiro-krew/themes/*.yaml cmd/kiro-krew/templates/kiro-krew/themes/

# Evals (excluding results directory - test fixtures and rubrics)
cp .kiro-krew/evals/fixtures/* cmd/kiro-krew/templates/kiro-krew/evals/fixtures/
cp .kiro-krew/evals/rubrics/* cmd/kiro-krew/templates/kiro-krew/evals/rubrics/
cp -r .kiro-krew/evals/cases/* cmd/kiro-krew/templates/kiro-krew/evals/cases/
```

### Exclusion Patterns

**Never sync *-conventions skills**: Skills ending in `-conventions` (e.g., `builder-conventions`, `planner-conventions`) are project-specific and must NOT be synchronized to templates.

```bash
# WRONG - Do not sync conventions skills
# cp .kiro/skills/builder-conventions/* cmd/kiro-krew/templates/

# CORRECT - Conventions skills stay project-specific
echo "Skipping *-conventions skills - project-specific"
```

### Sync Verification Example

Complete workflow for agent file modifications:

```bash
# 1. Modify live agent file
vim .kiro/agents/builder.json

# 2. Sync to template
cp .kiro/agents/builder.json cmd/kiro-krew/templates/kiro/agents/

# 3. Verify synchronization
diff .kiro/agents/builder.json cmd/kiro-krew/templates/kiro/agents/builder.json
# Should show no differences

# 4. Test template functionality (if modifying core templates)
cd /tmp && kiro-krew init test-project && cd test-project
diff .kiro/agents/builder.json /path/to/source/cmd/kiro-krew/templates/kiro/agents/builder.json
```

## Template/Live Synchronization

When updating any project files that have corresponding templates, ensure both template and live versions are updated:

### Agent Files
- **Live**: `.kiro/agents/*.json`
- **Template**: `cmd/kiro-krew/templates/kiro/agents/*.json`
- **Sync Rule**: Both versions must have identical JSON structure and content

### Evaluation Cases
- **Live**: `.kiro/evals/**/*`
- **Template**: `cmd/kiro-krew/templates/kiro/evals/**/*`
- **Sync Rule**: Template contains representative test cases, live may have additional project-specific cases

### Rubrics
- **Live**: `.kiro/rubrics/**/*`
- **Template**: `cmd/kiro-krew/templates/kiro/rubrics/**/*`
- **Sync Rule**: Templates contain standard rubrics, live may have project-specific extensions

### Scripts
- **Live**: `.kiro-krew/scripts/**/*`
- **Template**: `cmd/kiro-krew/templates/kiro-krew/scripts/**/*`
- **Sync Rule**: Both versions must be functionally equivalent

### Themes
- **Live**: `.kiro/themes/**/*`
- **Template**: `cmd/kiro-krew/templates/kiro/themes/**/*`
- **Sync Rule**: Templates contain default themes, live may have project customizations

## Blocking Verification Commands

**MANDATORY**: Run verification before task completion. Script MUST exit with error if files are out of sync.

### Verification Script Logic

```bash
#!/bin/bash
# Template sync verification - exits with error if any differences found
sync_error=0

echo "Verifying template synchronization..."

# Check agent files (JSON configs and prompt files)
for file in .kiro/agents/*.json; do
  if [[ -f "$file" ]]; then
    template_file="cmd/kiro-krew/templates/kiro/agents/$(basename "$file")"
    if ! diff -q "$file" "$template_file" >/dev/null 2>&1; then
      echo "ERROR: $file differs from $template_file"
      sync_error=1
    fi
  fi
done

for file in .kiro/agents/*.md; do
  if [[ -f "$file" ]]; then
    template_file="cmd/kiro-krew/templates/kiro/agents/$(basename "$file")"
    if ! diff -q "$file" "$template_file" >/dev/null 2>&1; then
      echo "ERROR: $file differs from $template_file"
      sync_error=1
    fi
  fi
done

# Check scripts
for file in .kiro-krew/scripts/*.sh; do
  if [[ -f "$file" ]]; then
    template_file="cmd/kiro-krew/templates/kiro-krew/scripts/$(basename "$file")"
    if ! diff -q "$file" "$template_file" >/dev/null 2>&1; then
      echo "ERROR: $file differs from $template_file"
      sync_error=1
    fi
  fi
done

# Check themes
for file in .kiro-krew/themes/*.yaml; do
  if [[ -f "$file" ]]; then
    template_file="cmd/kiro-krew/templates/kiro-krew/themes/$(basename "$file")"
    if ! diff -q "$file" "$template_file" >/dev/null 2>&1; then
      echo "ERROR: $file differs from $template_file"
      sync_error=1
    fi
  fi
done

# Check evals (excluding results directory)
for file in .kiro-krew/evals/fixtures/*; do
  if [[ -f "$file" ]]; then
    template_file="cmd/kiro-krew/templates/kiro-krew/evals/fixtures/$(basename "$file")"
    if ! diff -q "$file" "$template_file" >/dev/null 2>&1; then
      echo "ERROR: $file differs from $template_file"
      sync_error=1
    fi
  fi
done

for file in .kiro-krew/evals/rubrics/*; do
  if [[ -f "$file" ]]; then
    template_file="cmd/kiro-krew/templates/kiro-krew/evals/rubrics/$(basename "$file")"
    if ! diff -q "$file" "$template_file" >/dev/null 2>&1; then
      echo "ERROR: $file differs from $template_file"
      sync_error=1
    fi
  fi
done

# Check evals cases (recursive)
if [[ -d ".kiro-krew/evals/cases" ]]; then
  find .kiro-krew/evals/cases -type f | while read -r file; do
    relative_path="${file#.kiro-krew/evals/cases/}"
    template_file="cmd/kiro-krew/templates/kiro-krew/evals/cases/$relative_path"
    if ! diff -q "$file" "$template_file" >/dev/null 2>&1; then
      echo "ERROR: $file differs from $template_file"
      exit 1
    fi
  done
  # Check exit status from subshell
  if [[ $? -ne 0 ]]; then
    sync_error=1
  fi
fi

if [[ $sync_error -eq 0 ]]; then
  echo "✅ All template files are synchronized"
else
  echo "❌ Template synchronization verification failed"
  echo "Run sync commands to fix differences before task completion"
fi

exit $sync_error
```

### Required Verification Commands

Use these commands to verify synchronization before task completion:

```bash
# Option 1: Inline verification (recommended)
sync_error=0
for file in .kiro/agents/*.json .kiro/agents/*.md; do
  if [[ -f "$file" ]]; then
    template_file="cmd/kiro-krew/templates/kiro/agents/$(basename "$file")"
    if ! diff -q "$file" "$template_file" >/dev/null 2>&1; then
      echo "ERROR: $file differs from $template_file" && sync_error=1
    fi
  fi
done
# [Include similar loops for scripts, themes, evals...]
[[ $sync_error -eq 0 ]] && echo "✅ Verification passed" || { echo "❌ Verification failed"; exit 1; }

# Option 2: Quick individual checks
diff .kiro/agents/builder.json cmd/kiro-krew/templates/kiro/agents/builder.json || exit 1

# Option 3: All agent manifests check
for agent in architect builder documenter krew-lead planner validator; do
  diff .kiro/agents/$agent.json cmd/kiro-krew/templates/kiro/agents/$agent.json || { echo "SYNC ERROR: $agent"; exit 1; }
done
```

### Debugging Sync Failures

When verification fails:

```bash
# Show exact differences
diff -u .kiro/agents/builder.json cmd/kiro-krew/templates/kiro/agents/builder.json

# Check file existence
ls -la .kiro/agents/*.json
ls -la cmd/kiro-krew/templates/kiro/agents/*.json

# Re-run sync commands if files differ
cp .kiro/agents/*.json cmd/kiro-krew/templates/kiro/agents/
```

## Legacy Verification Commands (for reference)

Basic verification commands that don't block on failures:

```bash
# Verify JSON manifests match
diff .kiro/agents/builder.json cmd/kiro-krew/templates/kiro/agents/builder.json

# Check all agent manifests
for agent in architect builder documenter krew-lead planner validator; do
  diff .kiro/agents/$agent.json cmd/kiro-krew/templates/kiro/agents/$agent.json || echo "DIFF: $agent"
done

# Verify script functionality (when modifying scripts)
.kiro-krew/scripts/worktree-create.sh test-worktree
.kiro-krew/scripts/worktree-merge.sh test-worktree
```

## Workflow Integration Requirements

**MANDATORY COMPLETION SEQUENCE**: All builders must follow this exact workflow when completing tasks that modify template-synchronized files:

### 1. Standard Implementation
- Complete the assigned task implementation
- Write/modify required files following project conventions

### 2. Template Synchronization (if applicable)
- Identify if any modified files are template-synchronized
- Run appropriate sync commands from the "Complete Sync Workflow Commands" section
- **Required for**: agent configs, scripts, themes, evaluation files

### 3. Sync Verification (BLOCKING)
- Run blocking verification commands before task completion
- Task CANNOT be marked complete if verification fails
- Must show ✅ verification passed status

### 4. Quality Assurance
- Run ALL discovered QA commands 
- Ensure 100% test pass rate
- Document QA results in sentinel file

### 5. Task Completion
- Create sentinel file only after ALL verifications pass
- Document sync verification results in completion report

## Complete Workflow Examples

### Example 1: Agent Configuration Update

```bash
# 1. Implement changes
vim .kiro/agents/builder.json

# 2. Sync to template
cp .kiro/agents/builder.json cmd/kiro-krew/templates/kiro/agents/

# 3. VERIFY SYNC (BLOCKING - task cannot complete if this fails)
diff .kiro/agents/builder.json cmd/kiro-krew/templates/kiro/agents/builder.json || { 
  echo "❌ SYNC VERIFICATION FAILED - cannot complete task"; exit 1; 
}
echo "✅ Sync verification passed"

# 4. Run QA
task lint
task test

# 5. Complete task (create sentinel file)
```

### Example 2: Script Modification with Full Verification

```bash
# 1. Implement changes  
vim .kiro-krew/scripts/worktree-create.sh

# 2. Sync to template
cp .kiro-krew/scripts/worktree-create.sh cmd/kiro-krew/templates/kiro-krew/scripts/

# 3. COMPREHENSIVE SYNC VERIFICATION (BLOCKING)
sync_error=0
for file in .kiro-krew/scripts/*.sh; do
  if [[ -f "$file" ]]; then
    template_file="cmd/kiro-krew/templates/kiro-krew/scripts/$(basename "$file")"
    if ! diff -q "$file" "$template_file" >/dev/null 2>&1; then
      echo "ERROR: $file differs from $template_file" && sync_error=1
    fi
  fi
done
[[ $sync_error -eq 0 ]] && echo "✅ All script templates synchronized" || { 
  echo "❌ Script sync verification failed - cannot complete task"; exit 1; 
}

# 4. Test functionality
.kiro-krew/scripts/worktree-create.sh test-verification
.kiro-krew/scripts/worktree-merge.sh test-verification

# 5. Run standard QA
task lint
task test

# 6. Complete task
```

### Example 3: Multiple File Types (Comprehensive)

```bash
# 1. Implement changes affecting multiple template types
vim .kiro/agents/validator.json
vim .kiro-krew/themes/dark.yaml
vim .kiro-krew/evals/fixtures/sample.yaml

# 2. Sync ALL affected template types
cp .kiro/agents/validator.json cmd/kiro-krew/templates/kiro/agents/
cp .kiro-krew/themes/dark.yaml cmd/kiro-krew/templates/kiro-krew/themes/
cp .kiro-krew/evals/fixtures/sample.yaml cmd/kiro-krew/templates/kiro-krew/evals/fixtures/

# 3. COMPREHENSIVE VERIFICATION (BLOCKING - ALL must pass)
sync_error=0

# Check agents
diff .kiro/agents/validator.json cmd/kiro-krew/templates/kiro/agents/validator.json || sync_error=1

# Check themes  
diff .kiro-krew/themes/dark.yaml cmd/kiro-krew/templates/kiro-krew/themes/dark.yaml || sync_error=1

# Check eval fixtures
diff .kiro-krew/evals/fixtures/sample.yaml cmd/kiro-krew/templates/kiro-krew/evals/fixtures/sample.yaml || sync_error=1

# BLOCKING CHECK - task fails if any sync errors
[[ $sync_error -eq 0 ]] && echo "✅ All synchronizations verified" || { 
  echo "❌ Multi-file sync verification failed - cannot complete task"
  echo "Re-run sync commands and fix differences before task completion"
  exit 1; 
}

# 4. Quality assurance continues only after sync verification passes
task lint
task test

# 5. Document in sentinel file
echo "- Template sync verification: ✅ PASSED" >> .kiro-krew/artifacts/builder-201.md
```

## Mandatory Sync Integration

### When Sync is Required

Sync verification is **REQUIRED** when modifying these file patterns:

```bash
# Agent configurations (JSON and MD)
.kiro/agents/*.json          → cmd/kiro-krew/templates/kiro/agents/
.kiro/agents/*.md            → cmd/kiro-krew/templates/kiro/agents/

# Scripts
.kiro-krew/scripts/*.sh      → cmd/kiro-krew/templates/kiro-krew/scripts/

# Themes  
.kiro-krew/themes/*.yaml     → cmd/kiro-krew/templates/kiro-krew/themes/

# Evaluation files (excluding results/)
.kiro-krew/evals/fixtures/*  → cmd/kiro-krew/templates/kiro-krew/evals/fixtures/
.kiro-krew/evals/rubrics/*   → cmd/kiro-krew/templates/kiro-krew/evals/rubrics/
.kiro-krew/evals/cases/**/*  → cmd/kiro-krew/templates/kiro-krew/evals/cases/
```

### When Sync is NOT Required

Skip sync verification for:
- Project-specific files (non-template)
- Skills ending in `-conventions` 
- Build artifacts and generated files
- Documentation in `docs/` or `app_docs/`
- Source code in `cmd/`, `pkg/`, etc.

## Implementation Patterns

### Quality Assurance Integration
- **Step 1**: Complete implementation
- **Step 2**: Sync templates (if applicable)
- **Step 3**: **BLOCKING** sync verification - task fails if verification fails
- **Step 4**: Run ALL discovered QA commands
- **Step 5**: Document verification results
- Use QA discovery results from `.kiro-krew/artifacts/qa-tools.md`
- Document specific QA command sources (CI vs build tool)

### Sentinel File Requirements

Include sync verification status in sentinel files:

```markdown
## Task Complete

**Template Sync**: ✅ VERIFIED (or "N/A - no template files modified")
**QA Results**: 
- Linting: ✅ PASS
- Tests: ✅ PASS (15/15 passed)
- Sync Verification: ✅ PASS

**Sync Commands Used**:
- `cp .kiro/agents/builder.json cmd/kiro-krew/templates/kiro/agents/`
- `diff .kiro/agents/builder.json cmd/kiro-krew/templates/kiro/agents/builder.json`
```

### File Modifications
- Preserve existing formatting and structure
- Maintain JSON validity for configuration files
- **NEW**: Verify template sync before task completion
- Document changes in sentinel files including sync status

### Error Recovery
- Address validator feedback from `.kiro-krew/artifacts/validator-<issue>.md`
- Focus on specific failing commands identified by validator
- **NEW**: Include sync verification in error recovery process
- Document how feedback was incorporated

## Project Standards

### Code Quality
- Follow existing code style and conventions
- Use project's configured linting and formatting tools
- Ensure all tests pass (100% pass rate required)

### Documentation
- Update relevant docs when adding features
- Follow project's documentation format
- Include usage examples for new functionality