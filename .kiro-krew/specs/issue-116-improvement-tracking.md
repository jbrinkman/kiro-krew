# Design Specification: Quantified Improvement Tracking for Evaluation Framework

**Issue:** #116 - feat: implement improvement tracking for quantified evaluation metrics  
**Closes:** #116

## Solution Approach

Extend the existing evaluation framework to track quantified improvements over time by introducing baseline comparison capabilities. The solution leverages git commit history for version tracking and adds improvement delta calculations to evaluation results.

Key architectural decisions:
1. **Backward Compatible Extensions**: Extend existing types rather than breaking changes
2. **Git-Based Baselines**: Use commit hashes to identify baseline evaluations for comparison
3. **Delta Calculation Layer**: Add improvement calculation logic as a separate concern
4. **Enhanced Diff Commands**: Extend existing diff functionality with improvement metrics

## Relevant Files

### Files to Modify
- `internal/eval/types.go` - Add improvement tracking fields to existing types
- `internal/eval/diff.go` - Enhance diff command with improvement calculations
- `internal/eval/runner.go` - Add baseline commit field to evaluation runs
- `cmd/kiro-krew/cmd/eval.go` - Add improvement report subcommands

### Files to Create
- `internal/eval/improvement.go` - Core improvement calculation logic
- `internal/eval/baseline.go` - Baseline management and lookup functions

### Related Files (Reference Only)
- `.kiro-krew/evals/results/` - Existing evaluation results directory
- `.kiro-krew/evals/rubrics/` - Evaluation criteria definitions

## Team Orchestration

### Builder Implementation Order
1. **Phase 1**: Extend data structures (`types.go`)
2. **Phase 2**: Add baseline management (`baseline.go`)
3. **Phase 3**: Implement improvement calculations (`improvement.go`)
4. **Phase 4**: Enhance diff command (`diff.go`)
5. **Phase 5**: Add CLI commands (`eval.go`)
6. **Phase 6**: Update runner for baseline tracking (`runner.go`)

### Validator Focus Areas
- Backward compatibility with existing result files
- Accuracy of improvement delta calculations
- Proper baseline commit resolution
- CLI command functionality

## Step-by-Step Task Breakdown

### Task 1: Extend Data Structures
**File**: `internal/eval/types.go`

Add improvement tracking fields to existing types:
```go
// Add to Summary struct
type Summary struct {
    GitHash         string             `json:"git_hash"`
    BaselineCommit  string             `json:"baseline_commit,omitempty"`
    TotalCost       CostInfo           `json:"total_cost"`
    AgentScores     map[string]float64 `json:"agent_scores"`
    ImprovementData *ImprovementData   `json:"improvement_data,omitempty"`
}

// New improvement tracking structure
type ImprovementData struct {
    AccuracyChange    float64            `json:"accuracy_change"`
    ErrorRateChange   int                `json:"error_rate_change"`
    AgentImprovements map[string]AgentImprovement `json:"agent_improvements"`
}

type AgentImprovement struct {
    ScoreDelta     float64 `json:"score_delta"`
    AccuracyGained float64 `json:"accuracy_gained"`
    ErrorsReduced  int     `json:"errors_reduced"`
}
```

**Acceptance Criteria**:
- Backward compatibility maintained (existing JSON files still parse)
- New fields are optional (omitempty tags used)
- Struct definitions compile without errors

### Task 2: Baseline Management Functions
**File**: `internal/eval/baseline.go`

Implement baseline lookup and management:
```go
func FindBaselineRun(targetCommit string) (string, error)
func LoadBaselineResults(baselineRun string) (*Summary, map[string]AgentResult, error)
func SetBaseline(currentCommit, baselineCommit string) error
```

**Acceptance Criteria**:
- Function resolves baseline runs by commit hash
- Handles missing baseline gracefully
- Loads complete baseline dataset (summary + agent results)

### Task 3: Improvement Calculation Engine
**File**: `internal/eval/improvement.go`

Core improvement calculation logic:
```go
func CalculateImprovements(current *Summary, currentResults map[string]AgentResult, 
                          baseline *Summary, baselineResults map[string]AgentResult) *ImprovementData
func calculateAgentImprovement(currentResult, baselineResult AgentResult) AgentImprovement
func calculateAccuracyDelta(currentScores, baselineScores map[string]float64) float64
```

**Acceptance Criteria**:
- Accuracy deltas calculated as percentage improvement
- Error rate changes counted as integer differences
- Handles missing agents/criteria gracefully
- Returns zero deltas when no baseline exists

### Task 4: Enhanced Diff Command
**File**: `internal/eval/diff.go`

Extend existing diff functionality:
- Add `--improvement` flag to show quantified improvements
- Add improvement metrics to standard diff output
- Enhance report formatting for Stage 3 evidence

**Acceptance Criteria**:
- Existing diff functionality unchanged
- New improvement metrics displayed clearly
- Percentage accuracy gains and error count reductions shown
- Backward compatible with existing diff usage

### Task 5: CLI Command Extensions
**File**: `cmd/kiro-krew/cmd/eval.go`

Add new subcommands:
```go
var improvementCmd = &cobra.Command{
    Use:   "improvement [baseline-commit]",
    Short: "Generate improvement report against baseline",
}

var baselineCmd = &cobra.Command{
    Use:   "baseline <commit>",
    Short: "Set baseline commit for improvement tracking",
}
```

**Acceptance Criteria**:
- `kiro-krew eval improvement` shows current vs baseline metrics
- `kiro-krew eval baseline <commit>` sets baseline reference
- Commands integrate with existing eval command structure
- Help text includes Stage 3 maturity context

### Task 6: Runner Integration
**File**: `internal/eval/runner.go`

Integrate baseline tracking into evaluation runs:
- Detect baseline commit automatically
- Calculate improvements during run execution
- Store improvement data in results

**Acceptance Criteria**:
- Baseline commit auto-detected from previous runs
- Improvement calculations performed when baseline exists
- Results include both raw scores and improvement deltas
- No impact on evaluation performance

## Validation Commands

### Unit Tests
```bash
go test ./internal/eval/... -v
```

### Integration Tests
```bash
# Test baseline detection
kiro-krew eval baseline HEAD~1
kiro-krew eval architect

# Test improvement calculation
kiro-krew eval improvement

# Test enhanced diff
kiro-krew eval diff <old-run> <new-run>
```

### Backward Compatibility
```bash
# Verify old results still parse
ls .kiro-krew/evals/results/
kiro-krew eval diff <old-format-run> <new-format-run>
```

### Stage 3 Maturity Evidence
```bash
# Generate improvement report showing:
# - Accuracy gained (%)
# - Error rate reduced (count)
kiro-krew eval improvement --format=evidence
```

## Implementation Notes

### Backward Compatibility Strategy
- All new fields use `omitempty` JSON tags
- Existing Summary/AgentResult structures extended, not replaced
- Legacy result files continue to parse without improvement data
- Graceful degradation when baseline data unavailable

### Git Integration
- Leverage existing git hash tracking in Summary struct
- Use `parseDirectoryName()` function for commit hash extraction
- Follow existing directory naming convention: `YYMMDD-HHMMSS-{hash}`

### Error Handling
- Missing baseline commits result in zero improvement deltas
- Malformed result files logged as warnings, not errors
- Partial improvements calculated when some agents missing

### Performance Considerations
- Improvement calculations performed post-evaluation to avoid impacting core runs
- Baseline data cached during diff operations
- Large result sets handled via streaming for memory efficiency

This implementation provides the quantified improvement evidence required for Stage 3 AI maturity while maintaining full backward compatibility with the existing evaluation framework.