# Evaluation Framework

The evaluation framework measures agent quality and cost, enabling data-driven prompt improvements.

## Directory Structure

```
.kiro-krew/evals/
  rubrics/           # Scoring criteria per agent
    architect.yaml
    builder.yaml
    validator.yaml
    planner.yaml
  cases/             # Test cases per agent
    architect/
      case-1.yaml
    builder/
      case-1.yaml
    ...
  results/           # Results tagged by git hash
    <git-short-hash>/
      architect.json
      builder.json
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
- **LLM-judged criteria** — scored by an LLM evaluator using the rubric description (requires output)
- **Cost criteria** — tracked automatically from token usage

Results are written to `.kiro-krew/evals/results/<git-hash>/` enabling before/after comparison when prompts change.

## Comparing Runs

The `eval diff` command shows:
- Per-criterion score deltas per agent
- Token and cost deltas
- Quality-per-dollar assessment
