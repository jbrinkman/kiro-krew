package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type Issue struct {
	Number int      `json:"number"`
	Title  string   `json:"title"`
	Labels []string `json:"labels"`
}

func GetToken() (string, error) {
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func ListIssues(repo, label string) ([]Issue, error) {
	cmd := exec.Command("gh", "issue", "list", "--repo", repo, "--label", label, "--state", "open", "--json", "number,title,labels")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, err
	}

	var filtered []Issue
	for _, issue := range issues {
		hasExcluded := false
		for _, l := range issue.Labels {
			if l == "kiro-krew-done" || l == "kiro-krew-failed" {
				hasExcluded = true
				break
			}
		}
		if !hasExcluded {
			filtered = append(filtered, issue)
		}
	}

	return filtered, nil
}

func AddLabel(repo string, issueNumber int, label string) error {
	cmd := exec.Command("gh", "issue", "edit", fmt.Sprintf("%d", issueNumber), "--repo", repo, "--add-label", label)
	return cmd.Run()
}