package eval

// CalculateImprovements computes improvement metrics between current and baseline results.
func CalculateImprovements(current *Summary, currentResults map[string]AgentResult, baseline *Summary, baselineResults map[string]AgentResult) *ImprovementData {
	if baseline == nil || baselineResults == nil {
		return &ImprovementData{
			AccuracyChange:    0,
			ErrorRateChange:   0,
			AgentImprovements: make(map[string]AgentImprovement),
		}
	}

	improvements := &ImprovementData{
		AgentImprovements: make(map[string]AgentImprovement),
	}

	// Calculate per-agent improvements
	for agentName, currentResult := range currentResults {
		baselineResult, exists := baselineResults[agentName]
		if !exists {
			improvements.AgentImprovements[agentName] = AgentImprovement{}
			continue
		}
		improvements.AgentImprovements[agentName] = calculateAgentImprovement(currentResult, baselineResult)
	}

	// Calculate overall accuracy change
	improvements.AccuracyChange = calculateAccuracyDelta(current.AgentScores, baseline.AgentScores)

	// Calculate error rate change (baseline errors - current errors)
	currentErrors := countErrors(currentResults)
	baselineErrors := countErrors(baselineResults)
	improvements.ErrorRateChange = float64(baselineErrors - currentErrors)

	return improvements
}

// calculateAgentImprovement computes improvement metrics for a single agent.
func calculateAgentImprovement(currentResult, baselineResult AgentResult) AgentImprovement {
	currentScore := calculateAgentScore(currentResult)
	baselineScore := calculateAgentScore(baselineResult)
	
	scoreDelta := currentScore - baselineScore
	
	var accuracyGained float64
	if baselineScore > 0 {
		accuracyGained = (currentScore - baselineScore) / baselineScore * 100
	}

	currentErrors := countAgentErrors(currentResult)
	baselineErrors := countAgentErrors(baselineResult)
	errorsReduced := baselineErrors - currentErrors

	return AgentImprovement{
		ScoreDelta:     scoreDelta,
		AccuracyGained: accuracyGained,
		ErrorsReduced:  errorsReduced,
	}
}

// calculateAccuracyDelta computes percentage improvement in accuracy scores.
func calculateAccuracyDelta(currentScores, baselineScores map[string]float64) float64 {
	if len(baselineScores) == 0 {
		return 0
	}

	var totalImprovement float64
	var count int

	for agent, currentScore := range currentScores {
		baselineScore, exists := baselineScores[agent]
		if !exists || baselineScore == 0 {
			continue
		}
		
		improvement := (currentScore - baselineScore) / baselineScore * 100
		totalImprovement += improvement
		count++
	}

	if count == 0 {
		return 0
	}

	return totalImprovement / float64(count)
}

// calculateAgentScore computes average score across all test cases for an agent.
func calculateAgentScore(result AgentResult) float64 {
	var totalScore, totalMax float64
	
	for _, caseResult := range result.Cases {
		for _, score := range caseResult.Scores {
			if !score.Skipped {
				totalScore += float64(score.Score)
				totalMax += float64(score.MaxScore)
			}
		}
	}
	
	if totalMax == 0 {
		return 0
	}
	
	return totalScore / totalMax * 100
}

// countErrors counts total errors across all agents.
func countErrors(results map[string]AgentResult) int {
	var total int
	for _, result := range results {
		total += countAgentErrors(result)
	}
	return total
}

// countAgentErrors counts errors for a single agent.
func countAgentErrors(result AgentResult) int {
	var errors int
	for _, caseResult := range result.Cases {
		for _, score := range caseResult.Scores {
			if !score.Skipped && score.Score == 0 {
				errors++
			}
		}
	}
	return errors
}