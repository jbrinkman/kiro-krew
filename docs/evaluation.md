# Evaluation Framework

The evaluation framework measures agent quality and cost, enabling data-driven prompt improvements.

## Directory Structure

```
.kiro-krew/evals/
  rubrics/           # Scoring criteria per agent
    architect.yaml
    builder.yaml
    documenter.yaml
    krew-lead.yaml
    planner.yaml
    validator.yaml
  cases/             # Test cases per agent
    architect/
      case-1.yaml
    builder/
      case-1.yaml
    documenter/
      case-1.yaml
    krew-lead/
      case-1.yaml
    ...
  results/           # Results tagged by git hash
    <git-short-hash>/
      architect.json
      builder.json
      documenter.json
      krew-lead.json
      summary.json
```

## Rubric Format

Each agent has a rubric YAML file defining scoring criteria:

```yaml
agent: architect
criteria:
  - name: task_decomposition
    description: "Spec breaks work into discrete, independently implementable tasks"
    scoring: 1-5
  - name: file_reference_accuracy
    description: "Referenced files exist and are relevant"
    scoring: 1-5
    deterministic: true    # Scored by code, not LLM
  - name: cost_efficiency
    description: "Token usage relative to output quality"
    type: cost             # Tracked as cost metric
```

Fields:
- `agent` — which agent this rubric evaluates
- `criteria[].name` — unique identifier for the criterion
- `criteria[].description` — what is being measured
- `criteria[].scoring` — score range (e.g. "1-5")
- `criteria[].deterministic` — if true, scored by code checks rather than LLM
- `criteria[].type` — set to "cost" for cost-tracking criteria

## Test Case Format

```yaml
name: simple-feature-issue
description: "Evaluate architect output for a simple feature request"
input: |
  The issue body or spec that the agent receives as input.
output: |
  Optional: pre-captured agent output for offline evaluation.
```

Fields:
- `name` — unique identifier
- `description` — what this case tests
- `input` — the input the agent would receive
- `output` — (optional) pre-captured output for offline scoring

## Running Evaluations

```bash
# Evaluate all agents
kiro-krew eval

# Evaluate a specific agent
kiro-krew eval architect

# Compare two runs
kiro-krew eval diff abc1234 def5678
```

## Adding Test Cases

1. Create a YAML file in `.kiro-krew/evals/cases/<agent>/`
2. Provide an `input` field with representative agent input
3. Optionally capture real agent output in the `output` field for offline evaluation

## How Scoring Works

- **Deterministic criteria** — scored by code checks (file existence, structural completeness)
- **LLM-judged criteria** — scored by an LLM evaluator using the rubric description (requires output and a configured judge)
- **Cost criteria** — tracked automatically from token usage

### Skipped Criteria

Non-deterministic criteria require an LLM judge to score. When no LLM judge is configured, these criteria are marked as `skipped` in the results and excluded from aggregate score calculations. This prevents false signal — scores only reflect what was actually measured.

Skipped criteria appear in results as:
```json
{
  "name": "task_decomposition",
  "score": 0,
  "max_score": 5,
  "skipped": true,
  "reasoning": "LLM judge not configured — criterion skipped"
}
```

To get full scoring coverage, configure an LLM judge (future feature). Until then, aggregate scores reflect only deterministic criteria.

Results are written to `.kiro-krew/evals/results/<git-hash>/` enabling before/after comparison when prompts change.

## Agent Coverage

All six shipped agents have rubrics and test cases:

| Agent | Rubric | Key Criteria |
|-------|--------|--------------|
| `architect` | `rubrics/architect.yaml` | task_decomposition, acceptance_criteria_testability, file_reference_accuracy, completeness |
| `builder` | `rubrics/builder.yaml` | code_correctness, spec_adherence, code_quality, test_coverage |
| `documenter` | `rubrics/documenter.yaml` | documentation_completeness, accuracy, file_naming_convention, practical_usage_guidance |
| `krew-lead` | `rubrics/krew-lead.yaml` | workflow_adherence, delegation_quality, retry_policy_compliance, error_handling |
| `planner` | `rubrics/planner.yaml` | requirement_clarity, scope_appropriateness, acceptance_criteria_quality, constraint_identification |
| `validator` | `rubrics/validator.yaml` | issue_coverage, test_execution, defect_detection, actionable_feedback |

## Evaluation Workflow

The evaluation framework serves as unit testing for prompt engineering. Follow this workflow when modifying agent prompts or configurations:

### Before Making Changes (Baseline)

**Required**: Run `kiro-krew eval` before making any prompt changes to establish a baseline:

```bash
# Capture current performance
kiro-krew eval
```

This creates a results snapshot at `.kiro-krew/evals/results/<git-hash>/` for comparison.

### After Making Changes (Verification)

**Required**: Run `kiro-krew eval` after prompt changes to verify improvements:

```bash
# Test modified behavior
kiro-krew eval

# Compare with baseline
kiro-krew eval diff <baseline-hash> <current-hash>
```

### Creating Test Cases for Behavioral Changes

When making specific behavioral changes, create targeted test cases:

1. **Identify the behavior** — What specific agent behavior are you changing?
2. **Create test case** — Add a case in `.kiro-krew/evals/cases/<agent>/` that exercises this behavior
3. **Verify coverage** — Ensure existing rubric criteria measure the desired change
4. **Test iteratively** — Run evaluations as you refine the prompt

Example workflow for improving architect task decomposition:
```bash
# 1. Baseline
kiro-krew eval architect

# 2. Add test case for complex decomposition scenario
# Edit .kiro-krew/evals/cases/architect/complex-decomposition.yaml

# 3. Modify architect prompt
# Edit .kiro/agents/architect-prompt.md

# 4. Verify improvement
kiro-krew eval architect
kiro-krew eval diff <baseline> <current>
```

### Evaluation as Unit Testing

Treat evaluations like unit tests:
- **Red-Green-Refactor**: Baseline (red) → Change (green) → Optimize (refactor)
- **Regression prevention**: Catch unintended behavior changes
- **Performance tracking**: Monitor cost and quality over time
- **Documentation**: Results serve as behavioral specifications

## Comparing Runs

The `eval diff` command shows:
- Per-criterion score deltas per agent
- Token and cost deltas
- Quality-per-dollar assessment
