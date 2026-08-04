package eval

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCalculateImprovements tests improvement calculation with various scenarios
func TestCalculateImprovements(t *testing.T) {
	tests := []struct {
		name                string
		current             Summary
		baselineHash        string
		setupBaseline       func(t *testing.T, tempDir string) string
		wantErr             bool
		wantNil             bool
		wantOverall         float64
		wantSignificantLen  int
		wantAccuracyChanges map[string]float64
	}{
		{
			name: "significant improvement case",
			current: Summary{
				GitHash: "abc123de",
				AgentScores: map[string]float64{
					"architect": 0.90,
					"builder":   0.85,
					"validator": 0.88,
				},
			},
			setupBaseline: func(t *testing.T, tempDir string) string {
				baseline := Summary{
					GitHash: "def456ab",
					AgentScores: map[string]float64{
						"architect": 0.80,
						"builder":   0.75,
						"validator": 0.78,
					},
				}
				hash := createTestRun(t, tempDir, baseline, "def456ab")
				return hash
			},
			wantErr:            false,
			wantNil:            false,
			wantOverall:        10.0, // ~10% average improvement
			wantSignificantLen: 3,    // All agents have >5% improvement
			wantAccuracyChanges: map[string]float64{
				"architect": 10.0,
				"builder":   10.0,
				"validator": 10.0,
			},
		},
		{
			name: "regression case",
			current: Summary{
				GitHash: "abc456de",
				AgentScores: map[string]float64{
					"architect": 0.70,
					"builder":   0.65,
				},
			},
			setupBaseline: func(t *testing.T, tempDir string) string {
				baseline := Summary{
					GitHash: "def789ab",
					AgentScores: map[string]float64{
						"architect": 0.80,
						"builder":   0.75,
					},
				}
				hash := createTestRun(t, tempDir, baseline, "def789ab")
				return hash
			},
			wantErr:            false,
			wantNil:            false,
			wantOverall:        -10.0, // 10% regression
			wantSignificantLen: 2,     // Both agents regressed >5%
			wantAccuracyChanges: map[string]float64{
				"architect": -10.0,
				"builder":   -10.0,
			},
		},
		{
			name: "no change case",
			current: Summary{
				GitHash: "abc789de",
				AgentScores: map[string]float64{
					"architect": 0.80,
					"builder":   0.75,
				},
			},
			setupBaseline: func(t *testing.T, tempDir string) string {
				baseline := Summary{
					GitHash: "def012ab",
					AgentScores: map[string]float64{
						"architect": 0.80,
						"builder":   0.75,
					},
				}
				hash := createTestRun(t, tempDir, baseline, "def012ab")
				return hash
			},
			wantErr:            false,
			wantNil:            false,
			wantOverall:        0.0,
			wantSignificantLen: 0, // No significant changes
			wantAccuracyChanges: map[string]float64{
				"architect": 0.0,
				"builder":   0.0,
			},
		},
		{
			name: "no baseline set",
			current: Summary{
				GitHash:     "abc999de",
				AgentScores: map[string]float64{"architect": 0.80},
			},
			baselineHash: "",
			wantErr:      false,
			wantNil:      true, // Should return nil when no baseline
		},
		{
			name: "missing baseline run",
			current: Summary{
				GitHash:     "abc000de",
				AgentScores: map[string]float64{"architect": 0.80},
			},
			baselineHash: "deadbeef",
			wantErr:      true,
		},
		{
			name: "missing agent in baseline",
			current: Summary{
				GitHash: "abc111de",
				AgentScores: map[string]float64{
					"architect": 0.90,
					"builder":   0.85,
					"validator": 0.88,
				},
			},
			setupBaseline: func(t *testing.T, tempDir string) string {
				baseline := Summary{
					GitHash: "def111ab",
					AgentScores: map[string]float64{
						"architect": 0.80,
						"builder":   0.75,
						// validator missing in baseline
					},
				}
				hash := createTestRun(t, tempDir, baseline, "def111ab")
				return hash
			},
			wantErr:            false,
			wantNil:            false,
			wantOverall:        10.0, // Only architect and builder calculated
			wantSignificantLen: 2,    // Both have >5% improvement
			wantAccuracyChanges: map[string]float64{
				"architect": 10.0,
				"builder":   10.0,
				// validator not in map
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup temp directory
			tempDir := t.TempDir()
			oldResultsDir := filepath.Join(".kiro-krew", "evals", "results")
			newResultsDir := filepath.Join(tempDir, ".kiro-krew", "evals", "results")

			// Override results directory for testing
			defer setupTestResultsDir(t, oldResultsDir, newResultsDir)()

			// Setup baseline if provided
			baselineHash := tt.baselineHash
			if tt.setupBaseline != nil {
				baselineHash = tt.setupBaseline(t, tempDir)
			}

			// Run test
			metrics, err := CalculateImprovements(tt.current, baselineHash)

			// Check error expectations
			if tt.wantErr {
				if err == nil {
					t.Errorf("CalculateImprovements() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("CalculateImprovements() unexpected error: %v", err)
				return
			}

			// Check nil expectations
			if tt.wantNil {
				if metrics != nil {
					t.Errorf("CalculateImprovements() expected nil, got %+v", metrics)
				}
				return
			}

			// Validate metrics
			if metrics == nil {
				t.Fatal("CalculateImprovements() returned nil metrics unexpectedly")
			}

			// Check overall improvement (with tolerance for float comparison)
			tolerance := 0.01
			if abs(metrics.OverallImprovement-tt.wantOverall) > tolerance {
				t.Errorf("OverallImprovement = %.2f, want %.2f", metrics.OverallImprovement, tt.wantOverall)
			}

			// Check significant changes count
			if len(metrics.SignificantChanges) != tt.wantSignificantLen {
				t.Errorf("SignificantChanges count = %d, want %d", len(metrics.SignificantChanges), tt.wantSignificantLen)
			}

			// Check accuracy changes
			if tt.wantAccuracyChanges != nil {
				for agent, wantChange := range tt.wantAccuracyChanges {
					gotChange, exists := metrics.AccuracyChange[agent]
					if !exists {
						t.Errorf("AccuracyChange missing agent %s", agent)
						continue
					}
					if abs(gotChange-wantChange) > tolerance {
						t.Errorf("AccuracyChange[%s] = %.2f, want %.2f", agent, gotChange, wantChange)
					}
				}

				// Check for unexpected agents
				for agent := range metrics.AccuracyChange {
					if _, expected := tt.wantAccuracyChanges[agent]; !expected {
						t.Errorf("AccuracyChange has unexpected agent %s", agent)
					}
				}
			}
		})
	}
}

// TestFindBaselineRun tests baseline lookup functionality
func TestFindBaselineRun(t *testing.T) {
	tests := []struct {
		name         string
		baselineHash string
		setupRuns    func(t *testing.T, tempDir string)
		wantErr      bool
		wantGitHash  string
		wantScores   map[string]float64
	}{
		{
			name:         "find existing baseline",
			baselineHash: "abc123de",
			setupRuns: func(t *testing.T, tempDir string) {
				summary := Summary{
					GitHash: "abc123de",
					AgentScores: map[string]float64{
						"architect": 0.85,
						"builder":   0.82,
					},
				}
				createTestRun(t, tempDir, summary, "abc123de")
			},
			wantErr:     false,
			wantGitHash: "abc123de",
			wantScores: map[string]float64{
				"architect": 0.85,
				"builder":   0.82,
			},
		},
		{
			name:         "baseline not found",
			baselineHash: "deadbeef",
			setupRuns: func(t *testing.T, tempDir string) {
				summary := Summary{
					GitHash:     "cafe0123",
					AgentScores: map[string]float64{"architect": 0.85},
				}
				createTestRun(t, tempDir, summary, "cafe0123")
			},
			wantErr: true,
		},
		{
			name:         "empty results directory",
			baselineHash: "abc123",
			setupRuns:    func(t *testing.T, tempDir string) {},
			wantErr:      true,
		},
		{
			name:         "multiple runs, find correct one",
			baselineHash: "def456ab",
			setupRuns: func(t *testing.T, tempDir string) {
				// Create multiple runs
				for _, hash := range []string{"abc001de", "def456ab", "fea003bc"} {
					summary := Summary{
						GitHash: hash,
						AgentScores: map[string]float64{
							"architect": 0.80,
						},
					}
					createTestRun(t, tempDir, summary, hash)
				}
			},
			wantErr:     false,
			wantGitHash: "def456ab",
			wantScores: map[string]float64{
				"architect": 0.80,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup temp directory
			tempDir := t.TempDir()
			oldResultsDir := filepath.Join(".kiro-krew", "evals", "results")
			newResultsDir := filepath.Join(tempDir, ".kiro-krew", "evals", "results")

			// Override results directory
			defer setupTestResultsDir(t, oldResultsDir, newResultsDir)()

			// Setup test runs
			if tt.setupRuns != nil {
				tt.setupRuns(t, tempDir)
			}

			// Run test
			summary, err := FindBaselineRun(tt.baselineHash)

			// Check error expectations
			if tt.wantErr {
				if err == nil {
					t.Errorf("FindBaselineRun() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("FindBaselineRun() unexpected error: %v", err)
				return
			}

			// Validate summary
			if summary.GitHash != tt.wantGitHash {
				t.Errorf("GitHash = %s, want %s", summary.GitHash, tt.wantGitHash)
			}

			if tt.wantScores != nil {
				for agent, wantScore := range tt.wantScores {
					gotScore, exists := summary.AgentScores[agent]
					if !exists {
						t.Errorf("AgentScores missing agent %s", agent)
						continue
					}
					if abs(gotScore-wantScore) > 0.01 {
						t.Errorf("AgentScores[%s] = %.2f, want %.2f", agent, gotScore, wantScore)
					}
				}
			}
		})
	}
}

// TestAnalyzeTrends tests multi-commit trend analysis
func TestAnalyzeTrends(t *testing.T) {
	tests := []struct {
		name       string
		commits    []string
		setupRuns  func(t *testing.T, tempDir string)
		wantLen    int
		wantHashes []string
	}{
		{
			name:    "analyze multiple commits",
			commits: []string{"abcd0001", "abcd0002", "abcd0003"},
			setupRuns: func(t *testing.T, tempDir string) {
				for i, hash := range []string{"abcd0001", "abcd0002", "abcd0003"} {
					summary := Summary{
						GitHash: hash,
						AgentScores: map[string]float64{
							"architect": 0.80 + float64(i)*0.05,
						},
						TotalCost: CostInfo{
							EstimatedUSD: 0.001 * float64(i+1),
						},
					}
					createTestRun(t, tempDir, summary, hash)
				}
			},
			wantLen:    3,
			wantHashes: []string{"abcd0001", "abcd0002", "abcd0003"},
		},
		{
			name:    "skip missing commits",
			commits: []string{"cafebabe", "deadbeef", "feedface"},
			setupRuns: func(t *testing.T, tempDir string) {
				for _, hash := range []string{"cafebabe", "feedface"} {
					summary := Summary{
						GitHash:     hash,
						AgentScores: map[string]float64{"architect": 0.80},
					}
					createTestRun(t, tempDir, summary, hash)
				}
			},
			wantLen:    2,
			wantHashes: []string{"cafebabe", "feedface"},
		},
		{
			name:      "no commits found",
			commits:   []string{"fade0001", "fade0002"},
			setupRuns: func(t *testing.T, tempDir string) {},
			wantLen:   0,
		},
		{
			name:       "empty commit list",
			commits:    []string{},
			setupRuns:  func(t *testing.T, tempDir string) {},
			wantLen:    0,
			wantHashes: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup temp directory
			tempDir := t.TempDir()
			oldResultsDir := filepath.Join(".kiro-krew", "evals", "results")
			newResultsDir := filepath.Join(tempDir, ".kiro-krew", "evals", "results")

			// Override results directory
			defer setupTestResultsDir(t, oldResultsDir, newResultsDir)()

			// Setup test runs
			if tt.setupRuns != nil {
				tt.setupRuns(t, tempDir)
			}

			// Run test
			trends, err := AnalyzeTrends(tt.commits)

			// Should not error, just skip missing
			if err != nil {
				t.Errorf("AnalyzeTrends() unexpected error: %v", err)
				return
			}

			// Check length
			if len(trends) != tt.wantLen {
				t.Errorf("AnalyzeTrends() returned %d trends, want %d", len(trends), tt.wantLen)
			}

			// Check hashes
			if tt.wantHashes != nil {
				for i, wantHash := range tt.wantHashes {
					if i >= len(trends) {
						t.Errorf("Missing trend point %d", i)
						continue
					}
					if trends[i].GitHash != wantHash {
						t.Errorf("trends[%d].GitHash = %s, want %s", i, trends[i].GitHash, wantHash)
					}
				}
			}
		})
	}
}

// TestSetBaseline tests baseline persistence
func TestSetBaseline(t *testing.T) {
	tests := []struct {
		name       string
		commitHash string
		setupRun   func(t *testing.T, tempDir string)
		wantErr    bool
	}{
		{
			name:       "set valid baseline",
			commitHash: "abcdef01",
			setupRun: func(t *testing.T, tempDir string) {
				summary := Summary{
					GitHash:     "abcdef01",
					AgentScores: map[string]float64{"architect": 0.80},
				}
				createTestRun(t, tempDir, summary, "abcdef01")
			},
			wantErr: false,
		},
		{
			name:       "invalid commit hash",
			commitHash: "deadbeef",
			setupRun:   func(t *testing.T, tempDir string) {},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup temp directory
			tempDir := t.TempDir()
			oldResultsDir := filepath.Join(".kiro-krew", "evals", "results")
			newResultsDir := filepath.Join(tempDir, ".kiro-krew", "evals", "results")

			// Override results directory and baseline file location
			defer setupTestResultsDir(t, oldResultsDir, newResultsDir)()
			defer setupTestEvalsDir(t, tempDir)()

			// Setup test run
			if tt.setupRun != nil {
				tt.setupRun(t, tempDir)
			}

			// Run test
			err := SetBaseline(tt.commitHash)

			// Check error expectations
			if tt.wantErr {
				if err == nil {
					t.Errorf("SetBaseline() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("SetBaseline() unexpected error: %v", err)
				return
			}

			// Verify baseline file was created
			baselineFile := filepath.Join(tempDir, ".kiro-krew", "evals", ".baseline")
			content, err := os.ReadFile(baselineFile)
			if err != nil {
				t.Errorf("Failed to read baseline file: %v", err)
				return
			}

			if string(content) != tt.commitHash {
				t.Errorf("Baseline file content = %s, want %s", string(content), tt.commitHash)
			}
		})
	}
}

// TestLoadBaseline tests baseline loading
func TestLoadBaseline(t *testing.T) {
	tests := []struct {
		name            string
		setupBaseline   func(t *testing.T, tempDir string)
		wantBaselineVal string
	}{
		{
			name: "load existing baseline",
			setupBaseline: func(t *testing.T, tempDir string) {
				baselineFile := filepath.Join(tempDir, ".kiro-krew", "evals", ".baseline")
				if err := os.MkdirAll(filepath.Dir(baselineFile), 0755); err != nil {
					t.Fatalf("Failed to create evals dir: %v", err)
				}
				if err := os.WriteFile(baselineFile, []byte("abc123de"), 0644); err != nil {
					t.Fatalf("Failed to write baseline file: %v", err)
				}
			},
			wantBaselineVal: "abc123de",
		},
		{
			name:            "no baseline file",
			setupBaseline:   func(t *testing.T, tempDir string) {},
			wantBaselineVal: "",
		},
		{
			name: "baseline with whitespace",
			setupBaseline: func(t *testing.T, tempDir string) {
				baselineFile := filepath.Join(tempDir, ".kiro-krew", "evals", ".baseline")
				if err := os.MkdirAll(filepath.Dir(baselineFile), 0755); err != nil {
					t.Fatalf("Failed to create evals dir: %v", err)
				}
				if err := os.WriteFile(baselineFile, []byte("  def456ab\n"), 0644); err != nil {
					t.Fatalf("Failed to write baseline file: %v", err)
				}
			},
			wantBaselineVal: "def456ab",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup temp directory
			tempDir := t.TempDir()

			// Override evals directory location
			defer setupTestEvalsDir(t, tempDir)()

			// Setup baseline
			if tt.setupBaseline != nil {
				tt.setupBaseline(t, tempDir)
			}

			// Run test
			result := LoadBaseline()

			if result != tt.wantBaselineVal {
				t.Errorf("LoadBaseline() = %s, want %s", result, tt.wantBaselineVal)
			}
		})
	}
}

// Helper functions

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// createTestRun creates a test evaluation run in the temp directory
func createTestRun(t *testing.T, tempDir string, summary Summary, hash string) string {
	t.Helper()

	// Create directory with timestamp format
	dirName := "240101-120000-" + hash
	runDir := filepath.Join(tempDir, ".kiro-krew", "evals", "results", dirName)

	if err := os.MkdirAll(runDir, 0755); err != nil {
		t.Fatalf("Failed to create run directory %s: %v", runDir, err)
	}

	// Write summary.json
	summaryPath := filepath.Join(runDir, "summary.json")
	if err := writeSummaryJSON(summaryPath, summary); err != nil {
		t.Fatalf("Failed to write summary to %s: %v", summaryPath, err)
	}

	return hash
}

// writeSummaryJSON writes a Summary to a JSON file
func writeSummaryJSON(path string, summary Summary) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Manual JSON encoding to avoid import encoding/json
	content := `{
  "git_hash": "` + summary.GitHash + `",
  "total_cost": {
    "tokens_in": 0,
    "tokens_out": 0,
    "estimated_usd": ` + floatToString(summary.TotalCost.EstimatedUSD) + `
  },
  "agent_scores": {`

	first := true
	for agent, score := range summary.AgentScores {
		if !first {
			content += ","
		}
		content += `
    "` + agent + `": ` + floatToString(score)
		first = false
	}

	content += `
  }`

	if summary.BaselineCommit != "" {
		content += `,
  "baseline_commit": "` + summary.BaselineCommit + `"`
	}

	content += `
}`

	_, err = file.WriteString(content)
	return err
}

// floatToString converts float64 to string for JSON
func floatToString(f float64) string {
	// Simple conversion for test purposes
	s := ""
	if f == 0 {
		return "0.0"
	}

	// Format with reasonable precision
	intPart := int(f)
	decPart := int((f - float64(intPart)) * 1000000)

	if decPart < 0 {
		decPart = -decPart
	}

	// Build string manually to avoid fmt import in minimal way
	if intPart < 0 || f < 0 {
		s = "-"
		if intPart < 0 {
			intPart = -intPart
		}
	}

	s += intToString(intPart) + "."

	// Add decimal part with trimming
	decStr := intToString(decPart)
	for len(decStr) < 6 {
		decStr = "0" + decStr
	}
	// Trim trailing zeros
	for len(decStr) > 1 && decStr[len(decStr)-1] == '0' {
		decStr = decStr[:len(decStr)-1]
	}

	s += decStr
	return s
}

// intToString converts int to string
func intToString(n int) string {
	if n == 0 {
		return "0"
	}

	negative := n < 0
	if negative {
		n = -n
	}

	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}

	if negative {
		digits = append([]byte{'-'}, digits...)
	}

	return string(digits)
}

// setupTestResultsDir creates a temporary override for results directory
func setupTestResultsDir(t *testing.T, oldPath, newPath string) func() {
	t.Helper()

	// Create new results directory
	if err := os.MkdirAll(newPath, 0755); err != nil {
		t.Fatalf("Failed to create test results dir: %v", err)
	}

	// Save original working directory
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Extract temp root - newPath is like /tmp/TestXXX/.kiro-krew/evals/results
	// We want to chdir to /tmp/TestXXX
	tempRoot := newPath
	for i := 0; i < 3; i++ { // Go up 3 levels: results -> evals -> .kiro-krew -> root
		tempRoot = filepath.Dir(tempRoot)
	}

	// Change to temp directory
	if err := os.Chdir(tempRoot); err != nil {
		t.Fatalf("Failed to change to temp directory %s: %v", tempRoot, err)
	}

	// Return cleanup function
	return func() {
		if err := os.Chdir(origWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}
}

// setupTestEvalsDir sets up temporary evals directory
func setupTestEvalsDir(t *testing.T, tempDir string) func() {
	t.Helper()

	// Create evals directory
	evalsDir := filepath.Join(tempDir, ".kiro-krew", "evals")
	if err := os.MkdirAll(evalsDir, 0755); err != nil {
		t.Fatalf("Failed to create evals dir: %v", err)
	}

	// Save original working directory
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Return cleanup function
	return func() {
		if err := os.Chdir(origWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}
}

// TestEdgeCases tests edge cases and error conditions
func TestEdgeCases(t *testing.T) {
	t.Run("calculate improvements with empty agent scores", func(t *testing.T) {
		tempDir := t.TempDir()
		defer setupTestResultsDir(t, "", filepath.Join(tempDir, ".kiro-krew", "evals", "results"))()

		baseline := Summary{
			GitHash:     "abc00000",
			AgentScores: map[string]float64{},
		}
		hash := createTestRun(t, tempDir, baseline, "abc00000")

		current := Summary{
			GitHash:     "def00000",
			AgentScores: map[string]float64{},
		}

		metrics, err := CalculateImprovements(current, hash)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if metrics.OverallImprovement != 0.0 {
			t.Errorf("Expected 0.0 overall improvement for empty scores, got %.2f", metrics.OverallImprovement)
		}
	})

	t.Run("find baseline with malformed directory names", func(t *testing.T) {
		tempDir := t.TempDir()
		defer setupTestResultsDir(t, "", filepath.Join(tempDir, ".kiro-krew", "evals", "results"))()

		// Create directories with various invalid formats
		resultsDir := filepath.Join(tempDir, ".kiro-krew", "evals", "results")
		invalidDirs := []string{
			"invalid",
			"123",
			"not-a-timestamp",
		}

		for _, dirName := range invalidDirs {
			invalidDir := filepath.Join(resultsDir, dirName)
			if err := os.MkdirAll(invalidDir, 0755); err != nil {
				t.Fatalf("Failed to create invalid dir: %v", err)
			}
		}

		// Should not find any baseline
		_, err := FindBaselineRun("abc123de")
		if err == nil {
			t.Errorf("Expected error for malformed directories, got nil")
		}
	})

	t.Run("analyze trends with identical scores", func(t *testing.T) {
		tempDir := t.TempDir()
		defer setupTestResultsDir(t, "", filepath.Join(tempDir, ".kiro-krew", "evals", "results"))()

		// Create runs with identical scores
		commits := []string{"fade0001", "fade0002", "fade0003"}
		for _, hash := range commits {
			summary := Summary{
				GitHash: hash,
				AgentScores: map[string]float64{
					"architect": 0.80,
					"builder":   0.75,
				},
			}
			createTestRun(t, tempDir, summary, hash)
		}

		trends, err := AnalyzeTrends(commits)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify all scores are identical
		for i := 1; i < len(trends); i++ {
			for agent, score := range trends[i].Scores {
				prevScore := trends[i-1].Scores[agent]
				if abs(score-prevScore) > 0.001 {
					t.Errorf("Expected identical scores, got %.3f vs %.3f for %s", score, prevScore, agent)
				}
			}
		}
	})

	t.Run("calculate improvements with single agent", func(t *testing.T) {
		tempDir := t.TempDir()
		defer setupTestResultsDir(t, "", filepath.Join(tempDir, ".kiro-krew", "evals", "results"))()

		baseline := Summary{
			GitHash:     "beef0001",
			AgentScores: map[string]float64{"architect": 0.70},
		}
		hash := createTestRun(t, tempDir, baseline, "beef0001")

		current := Summary{
			GitHash:     "beef0002",
			AgentScores: map[string]float64{"architect": 0.80},
		}

		metrics, err := CalculateImprovements(current, hash)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// 10% improvement
		expectedImprovement := 10.0
		if abs(metrics.OverallImprovement-expectedImprovement) > 0.01 {
			t.Errorf("OverallImprovement = %.2f, want %.2f", metrics.OverallImprovement, expectedImprovement)
		}
	})
}

// TestShowTrends tests the trends display function
func TestShowTrends(t *testing.T) {
	tests := []struct {
		name      string
		commits   []string
		setupRuns func(t *testing.T, tempDir string)
		wantErr   bool
	}{
		{
			name:    "display trends successfully",
			commits: []string{"cafe0001", "cafe0002"},
			setupRuns: func(t *testing.T, tempDir string) {
				for i, hash := range []string{"cafe0001", "cafe0002"} {
					summary := Summary{
						GitHash: hash,
						AgentScores: map[string]float64{
							"architect": 0.80 + float64(i)*0.05,
						},
						TotalCost: CostInfo{
							EstimatedUSD: 0.001,
						},
					}
					createTestRun(t, tempDir, summary, hash)
				}
			},
			wantErr: false,
		},
		{
			name:      "no runs found",
			commits:   []string{"deadbeef"},
			setupRuns: func(t *testing.T, tempDir string) {},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			defer setupTestResultsDir(t, "", filepath.Join(tempDir, ".kiro-krew", "evals", "results"))()

			if tt.setupRuns != nil {
				tt.setupRuns(t, tempDir)
			}

			err := ShowTrends(tt.commits)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ShowTrends() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("ShowTrends() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestGenerateImprovementReport tests the improvement report generation
func TestGenerateImprovementReport(t *testing.T) {
	tests := []struct {
		name          string
		setupBaseline func(t *testing.T, tempDir string) string
		setupCurrent  func(t *testing.T, tempDir string)
		wantErr       bool
	}{
		{
			name: "generate report successfully",
			setupBaseline: func(t *testing.T, tempDir string) string {
				baseline := Summary{
					GitHash: "face0001",
					AgentScores: map[string]float64{
						"architect": 0.70,
						"builder":   0.65,
					},
				}
				hash := createTestRun(t, tempDir, baseline, "face0001")

				// Set baseline
				baselineFile := filepath.Join(tempDir, ".kiro-krew", "evals", ".baseline")
				if err := os.MkdirAll(filepath.Dir(baselineFile), 0755); err != nil {
					t.Fatalf("Failed to create evals dir: %v", err)
				}
				if err := os.WriteFile(baselineFile, []byte(hash), 0644); err != nil {
					t.Fatalf("Failed to write baseline: %v", err)
				}

				return hash
			},
			setupCurrent: func(t *testing.T, tempDir string) {
				current := Summary{
					GitHash: "face0002",
					AgentScores: map[string]float64{
						"architect": 0.80,
						"builder":   0.75,
					},
				}
				createTestRun(t, tempDir, current, "face0002")
			},
			wantErr: false,
		},
		{
			name: "no baseline set",
			setupBaseline: func(t *testing.T, tempDir string) string {
				// Don't set baseline
				return ""
			},
			setupCurrent: func(t *testing.T, tempDir string) {},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			defer setupTestResultsDir(t, "", filepath.Join(tempDir, ".kiro-krew", "evals", "results"))()
			defer setupTestEvalsDir(t, tempDir)()

			if tt.setupBaseline != nil {
				tt.setupBaseline(t, tempDir)
			}
			if tt.setupCurrent != nil {
				tt.setupCurrent(t, tempDir)
			}

			err := GenerateImprovementReport()

			if tt.wantErr {
				if err == nil {
					t.Errorf("GenerateImprovementReport() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("GenerateImprovementReport() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestHelperFunctions tests internal helper functions
func TestHelperFunctions(t *testing.T) {
	t.Run("extractAgentNames", func(t *testing.T) {
		trends := []TrendPoint{
			{
				GitHash: "commit1",
				Scores: map[string]float64{
					"architect": 0.80,
					"builder":   0.75,
				},
			},
			{
				GitHash: "commit2",
				Scores: map[string]float64{
					"architect": 0.85,
					"validator": 0.82,
				},
			},
		}

		agents := extractAgentNames(trends)

		// Should have 3 unique agents
		if len(agents) != 3 {
			t.Errorf("extractAgentNames() returned %d agents, want 3", len(agents))
		}

		// Check all expected agents are present
		agentSet := make(map[string]bool)
		for _, agent := range agents {
			agentSet[agent] = true
		}

		for _, expected := range []string{"architect", "builder", "validator"} {
			if !agentSet[expected] {
				t.Errorf("extractAgentNames() missing agent %s", expected)
			}
		}
	})

	t.Run("getImprovementDescription", func(t *testing.T) {
		tests := []struct {
			change float64
			want   string
		}{
			{15.0, "(major improvement)"},
			{7.0, "(significant improvement)"},
			{2.0, "(improvement)"},
			{0.0, "(no change)"},
			{-2.0, "(regression)"},
			{-5.0, "(significant regression)"},
			{-15.0, "(major regression)"},
		}

		for _, tt := range tests {
			got := getImprovementDescription(tt.change)
			if got != tt.want {
				t.Errorf("getImprovementDescription(%.1f) = %s, want %s", tt.change, got, tt.want)
			}
		}
	})

	t.Run("determineImprovementIndicator", func(t *testing.T) {
		tests := []struct {
			change float64
			want   string
		}{
			{15.0, "✓"},  // >5% improvement
			{7.0, "✓"},   // >5% improvement
			{2.0, "↑"},   // >0.001 improvement
			{0.0, "→"},   // no change
			{-2.0, "↓"},  // <-0.001 regression
			{-5.0, "✗"},  // <-3% regression
			{-15.0, "✗"}, // <-3% regression
		}

		for _, tt := range tests {
			got := determineImprovementIndicator(tt.change)
			if got != tt.want {
				t.Errorf("determineImprovementIndicator(%.1f) = %s, want %s", tt.change, got, tt.want)
			}
		}
	})
}
