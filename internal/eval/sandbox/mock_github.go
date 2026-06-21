package sandbox

import (
	"context"
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed testdata/github-cli-mock/*
var mockGitHubSkill embed.FS

// SetupGitHubMocking replaces .kiro/skills/github-cli/ with mock version in container
func (c *Container) SetupGitHubMocking(ctx context.Context, workspacePath string) error {
	skillPath := filepath.Join(workspacePath, ".kiro", "skills", "github-cli")

	// Create skills directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(skillPath), 0755); err != nil {
		return err
	}

	// Remove existing github-cli skill if present
	os.RemoveAll(skillPath)

	// Create mock skill directory
	if err := os.MkdirAll(skillPath, 0755); err != nil {
		return err
	}

	// Copy mock skill files
	return fs.WalkDir(mockGitHubSkill, "testdata/github-cli-mock", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel("testdata/github-cli-mock", path)
		destPath := filepath.Join(skillPath, relPath)

		content, err := mockGitHubSkill.ReadFile(path)
		if err != nil {
			return err
		}

		if err := os.WriteFile(destPath, content, 0755); err != nil {
			return err
		}

		return nil
	})
}

// ConfigureMockGitHubPath configures PATH to use mock gh binary in container
func (c *Container) ConfigureMockGitHubPath(ctx context.Context) error {
	// Add mock skill directory to PATH so 'gh' resolves to our mock
	return c.Exec(ctx, []string{"bash", "-c", "echo 'export PATH=/workspace/.kiro/skills/github-cli:$PATH' >> /home/sandbox/.bashrc"})
}

// GitHubMockResponse represents a mock GitHub API response
type GitHubMockResponse struct {
	IssueNumber int    `json:"issue_number,omitempty"`
	PRNumber    int    `json:"pr_number,omitempty"`
	URL         string `json:"url,omitempty"`
	Status      string `json:"status,omitempty"`
}

// SimulateGitHubResponse returns realistic GitHub API responses for testing
func SimulateGitHubResponse(operation string, args []string) GitHubMockResponse {
	switch operation {
	case "issue":
		if len(args) > 0 && args[0] == "create" {
			return GitHubMockResponse{
				IssueNumber: 12345,
				URL:         "https://github.com/test/test/issues/12345",
			}
		}
		return GitHubMockResponse{Status: "success"}

	case "pr":
		if len(args) > 0 && args[0] == "create" {
			return GitHubMockResponse{
				PRNumber: 42,
				URL:      "https://github.com/test/test/pull/42",
			}
		}
		return GitHubMockResponse{Status: "success"}

	default:
		return GitHubMockResponse{Status: "success"}
	}
}
