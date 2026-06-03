package eval

// Rubric defines scoring criteria for an agent.
type Rubric struct {
	Agent    string      `yaml:"agent" json:"agent"`
	Criteria []Criterion `yaml:"criteria" json:"criteria"`
}

// Criterion is a single scoring dimension within a rubric.
type Criterion struct {
	Name          string `yaml:"name" json:"name"`
	Description   string `yaml:"description" json:"description"`
	Scoring       string `yaml:"scoring" json:"scoring"` // e.g. "1-5"
	Deterministic bool   `yaml:"deterministic,omitempty" json:"deterministic,omitempty"`
	Type          string `yaml:"type,omitempty" json:"type,omitempty"` // e.g. "cost"
}

// TestCase defines input and expected characteristics for an agent evaluation.
type TestCase struct {
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
	Input       string `yaml:"input" json:"input"`
	Output      string `yaml:"output,omitempty" json:"output,omitempty"` // pre-captured output
	Agent       string `yaml:"agent" json:"agent"`
}

// CostInfo tracks token usage and estimated cost.
type CostInfo struct {
	TokensIn     int     `json:"tokens_in"`
	TokensOut    int     `json:"tokens_out"`
	EstimatedUSD float64 `json:"estimated_usd"`
}

// CriterionScore is the score for a single criterion on a single test case.
type CriterionScore struct {
	Name          string `json:"name"`
	Score         int    `json:"score"`
	MaxScore      int    `json:"max_score"`
	Deterministic bool   `json:"deterministic"`
	Skipped       bool   `json:"skipped,omitempty"`
	Reasoning     string `json:"reasoning,omitempty"`
}

// CaseResult holds scores and cost for one test case.
type CaseResult struct {
	CaseName string           `json:"case_name"`
	Scores   []CriterionScore `json:"scores"`
	Cost     CostInfo         `json:"cost"`
}

// AgentResult holds all case results for one agent.
type AgentResult struct {
	Agent   string       `json:"agent"`
	GitHash string       `json:"git_hash"`
	Cases   []CaseResult `json:"cases"`
}

// Summary holds aggregate results for an eval run.
type Summary struct {
	GitHash     string             `json:"git_hash"`
	TotalCost   CostInfo           `json:"total_cost"`
	AgentScores map[string]float64 `json:"agent_scores"` // agent -> average score
}
