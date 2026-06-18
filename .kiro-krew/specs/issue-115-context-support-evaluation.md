# Design Specification: Context Support for Evaluation Test Cases

**Issue:** #115 - feat: add context support to evaluation test cases  
**Author:** Architect Agent  
**Date:** 2026-06-18

Closes #115

## Solution Approach

Add optional context support to the evaluation framework to enable Stage 3 AI maturity testing with real-world scenarios. The implementation will:

1. **Extend TestCase struct** with optional `Context []string` field
2. **Update YAML parsing** to handle the new field transparently
3. **Enhance scoring functions** to utilize context when evaluating responses
4. **Maintain backward compatibility** - existing test cases continue to work unchanged

## Relevant Files

### Files to Modify
- `internal/eval/types.go` - Add `Context []string` field to TestCase struct
- `internal/eval/runner.go` - Update scoring functions to use context
- `.kiro-krew/evals/cases/planner/case-1.yaml` - Add context example

### Files to Create
- None (all changes are modifications to existing files)

### Files for Reference
- `internal/eval/runner_test.go` - Existing test patterns for validation
- `.kiro-krew/evals/rubrics/planner.yaml` - Current evaluation criteria

## Team Orchestration

This is a focused enhancement that requires:

1. **Data Structure Changes** (Builder) - Modify TestCase struct
2. **Scoring Logic Updates** (Builder) - Enhance deterministic and LLM scoring
3. **Example Updates** (Builder) - Add context to sample test case
4. **Testing** (Validator) - Ensure backward compatibility

No coordination with external systems required - all changes are within the evaluation framework.

## Step-by-Step Task Breakdown

### Task 1: Extend TestCase Data Structure
**Acceptance Criteria:**
- Add `Context []string` field to TestCase struct in `internal/eval/types.go`
- Field must have `yaml:"context,omitempty"` and `json:"context,omitempty"` tags
- Existing YAML parsing continues to work without the field
- New field is optional and defaults to empty/nil

### Task 2: Update Deterministic Scoring to Use Context
**Acceptance Criteria:**
- Modify `scoreDeterministic()` function to accept and utilize context when available
- Context should be included in scoring reasoning output when present
- All existing deterministic scoring patterns continue to work unchanged
- Context enhances evaluation but doesn't break existing logic

### Task 3: Update LLM-Judged Scoring to Include Context  
**Acceptance Criteria:**
- Modify `scoreLLMJudge()` function to include context in evaluation prompts
- Context appears in the "INPUT/CONTEXT" section of the LLM prompt
- LLM can use context to make more informed scoring decisions
- Existing behavior preserved when context is not provided

### Task 4: Add Context Support to Example Test Case
**Acceptance Criteria:**
- Update `.kiro-krew/evals/cases/planner/case-1.yaml` with context example
- Context should include relevant codebase information (file paths, constraints)
- Existing test case structure and output remain unchanged
- Context demonstrates realistic Stage 3 AI maturity scenario

### Task 5: Verify Context in Evaluation Output
**Acceptance Criteria:**
- Run evaluation with context-enabled test case
- Confirm context appears in evaluation reasoning output
- Verify context is properly included in LLM evaluation prompts
- Ensure scoring makes use of contextual information

### Task 6: Backward Compatibility Testing
**Acceptance Criteria:**
- All existing test cases pass without modification
- Evaluation runs successfully on test cases without context field
- No breaking changes to existing evaluation workflow
- Scoring results remain consistent for non-context test cases

## Validation Commands

```bash
# Verify structure changes compile
go build ./internal/eval/...

# Run existing evaluation tests
go test ./internal/eval/... -v

# Test YAML parsing with and without context
cd .kiro-krew/evals/cases/planner
# Should parse successfully with new context field
cat case-1.yaml

# Run evaluation to verify context integration
kiro-krew eval --agent planner

# Verify output includes context reasoning
cat .kiro-krew/evals/results/*/planner.json | grep -A5 -B5 "reasoning"
```

## Implementation Notes

### Context Field Design
- Use `[]string` type for flexibility in context representation
- Each string can represent a file path, constraint, or background info
- YAML `omitempty` ensures backward compatibility

### Scoring Integration Points
- **Deterministic scoring**: Include context in reasoning strings
- **LLM scoring**: Append context to existing INPUT/CONTEXT prompt section
- **Context formatting**: Present context as numbered list for clarity

### Backward Compatibility Strategy
- Optional field with `omitempty` tags prevents YAML/JSON serialization issues
- Nil checks in scoring functions handle missing context gracefully
- No changes to existing evaluation command line interface

### Example Context Usage
```yaml
context:
  - "Current router: internal/api/router.go"
  - "Existing user model: internal/models/user.go"  
  - "Config loading: internal/config/config.go"
  - "Constraint: Use golang-jwt/jwt/v5 library"
  - "Constraint: No database migration in this PR"
```

This design enables realistic Stage 3 AI evaluation while maintaining full backward compatibility with the existing evaluation framework.