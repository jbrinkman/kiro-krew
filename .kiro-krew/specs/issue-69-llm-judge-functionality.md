# Design Specification: LLM-as-a-Judge Functionality

Closes #69

## Solution Approach

Replace the stub `scoreLLMJudge` function with a working implementation that routes LLM evaluation calls through the kiro CLI to maintain consistency with the system architecture. The solution follows Promptfoo's llm-rubric pattern with structured JSON responses and graceful error handling.

Key architectural decisions:
- **CLI Integration**: Route LLM calls through `kiro chat` command to leverage existing LLM infrastructure
- **Structured Evaluation**: Use templated prompts that return standardized JSON responses
- **Score Normalization**: Convert 1-5 scale responses to internal 0-1 normalized scores
- **Error Resilience**: Implement fallback behavior for LLM failures or malformed responses
- **Future-Proofing**: Include TODO placeholders for chain-of-thought evaluation enhancement

## Relevant Files

**Files to Modify:**
- `internal/eval/runner.go` — Replace scoreLLMJudge stub with working implementation
- `internal/eval/llm_judge.go` — New file for LLM judge logic and prompt templates

**Files for Reference:**
- `internal/eval/types.go` — Existing data structures (Criterion, TestCase, CriterionScore)
- `.kiro-krew/evals/rubrics/*.yaml` — Existing rubric definitions with LLM-judged criteria
- `cmd/kiro-krew/cmd/root.go` — CLI command structure for routing calls

## Team Orchestration

This is a single-component implementation within the eval framework:
- **No breaking changes** to existing deterministic evaluation logic
- **Maintains compatibility** with current rubric YAML structure
- **Preserves existing** test case and result data formats
- **Integrates cleanly** with current eval workflow

## Step-by-Step Task Breakdown

### Task 1: Create LLM Judge Infrastructure
**Acceptance Criteria:**
- Create `internal/eval/llm_judge.go` with prompt template system
- Implement `callKiroLLM()` function to execute `kiro chat` commands
- Define standard prompt template following Promptfoo's llm-rubric pattern
- Parse JSON responses with score, reasoning, and pass fields
- Handle malformed responses gracefully with fallback scoring

**Implementation Details:**
- Use `exec.Command("kiro", "chat", "--json")` for CLI integration
- Template: "Evaluate this output against the criterion: {criterion}. Score 1-5 where..."
- Expected JSON: `{"score": int, "reasoning": string, "pass": bool}`
- Fallback: Return score 1, reasoning explaining the failure, skipped=false

### Task 2: Replace scoreLLMJudge Stub
**Acceptance Criteria:**
- Remove the current stub that returns skipped=true
- Implement actual LLM evaluation logic
- Convert 1-5 scale to normalized internal scoring
- Preserve existing function signature: `scoreLLMJudge(criterion Criterion, tc TestCase) (int, string, bool)`
- Return appropriate error handling for edge cases

**Implementation Details:**
- Call LLM judge infrastructure from Task 1
- Handle empty/missing test case output
- Convert LLM score (1-5) to internal scale using parseMaxScore()
- Maintain backward compatibility with existing evaluation pipeline

### Task 3: Add Chain-of-Thought Placeholders
**Acceptance Criteria:**
- Include TODO comments for future CoT evaluation enhancement
- Document where advanced reasoning chains would be integrated
- Ensure current implementation can be extended without breaking changes

**Implementation Details:**
- Add TODO in prompt template area for CoT prompting
- Add TODO in response parsing for multi-step reasoning
- Document extension points in code comments

### Task 4: Error Handling and Resilience
**Acceptance Criteria:**
- Handle kiro CLI command failures gracefully
- Parse malformed JSON responses without crashing
- Provide meaningful error messages in reasoning field
- Ensure evaluation continues even with individual LLM failures
- Log appropriate warnings for debugging

**Implementation Details:**
- Wrap CLI calls with timeout and error checking
- Use json.Unmarshal with fallback for malformed responses
- Never return skipped=true unless absolutely necessary
- Provide diagnostic information in failure cases

## Validation Commands

**Build and Test:**
```bash
go build ./cmd/kiro-krew
go test ./internal/eval/...
```

**Functional Testing:**
```bash
# Run eval on single agent to test LLM judge
./kiro-krew eval architect

# Verify results contain non-skipped LLM scores
cat .kiro-krew/evals/results/*/architect.json | jq '.cases[0].scores[] | select(.deterministic == false)'

# Check that previously deterministic evaluations still work
./kiro-krew eval builder
```

**Integration Testing:**
```bash
# Full eval run should complete without errors
./kiro-krew eval

# Summary should show meaningful scores for LLM-judged criteria
cat .kiro-krew/evals/results/*/summary.json | jq '.agent_scores'
```

**Manual Verification:**
- Verify LLM-judged criteria return scores between 1-5 (internal scale)
- Confirm reasoning fields contain meaningful explanations
- Check that deterministic criteria continue working unchanged
- Validate error handling by temporarily renaming kiro CLI