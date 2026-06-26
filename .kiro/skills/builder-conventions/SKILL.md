---
name: builder-conventions
description: Project-specific conventions and patterns for the builder agent. Maintains synchronization between template and live project files.
---

# Builder Conventions

Project-specific conventions, patterns, and best practices for the builder agent to follow during implementation tasks.

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

## Sync Verification Commands

Before completing tasks that modify template-synchronized files:

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

## Implementation Patterns

### Quality Assurance
- Run ALL discovered QA commands before completion
- Use QA discovery results from `.kiro-krew/artifacts/qa-tools.md`
- Document specific QA command sources (CI vs build tool)

### File Modifications
- Preserve existing formatting and structure
- Maintain JSON validity for configuration files
- Document changes in sentinel files

### Error Recovery
- Address validator feedback from `.kiro-krew/artifacts/validator-<issue>.md`
- Focus on specific failing commands identified by validator
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