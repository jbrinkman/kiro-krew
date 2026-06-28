package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/docker/docker/api/types/container"
)

//go:embed testdata/github-cli-mock/*
var mockGitHubSkill embed.FS

// SetupGitHubMocking replaces .kiro/skills/github-cli/ with mock version in container
func (c *Container) SetupGitHubMocking(ctx context.Context, workspacePath string) error {
	if c.containerID == "" {
		return fmt.Errorf("container not created - call Create() before SetupGitHubMocking")
	}

	// Create skills directory if it doesn't exist
	if _, err := c.ExecWithOutput(ctx, []string{"mkdir", "-p", "/workspace/.kiro/skills"}); err != nil {
		return err
	}

	// Remove existing github-cli skill if present
	if _, err := c.ExecWithOutput(ctx, []string{"rm", "-rf", "/workspace/.kiro/skills/github-cli"}); err != nil {
		return err
	}

	// Create mock skill directory
	if _, err := c.ExecWithOutput(ctx, []string{"mkdir", "-p", "/workspace/.kiro/skills/github-cli"}); err != nil {
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
		destPath := filepath.Join("/workspace/.kiro/skills/github-cli", relPath)

		content, err := mockGitHubSkill.ReadFile(path)
		if err != nil {
			return err
		}

		return c.copyContentToContainer(ctx, destPath, content, 0755)
	})
}

// copyContentToContainer copies byte content to a file in the container
func (c *Container) copyContentToContainer(ctx context.Context, destPath string, content []byte, mode int64) error {
	// Create tar archive with the content
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	header := &tar.Header{
		Name: filepath.Base(destPath),
		Mode: mode,
		Size: int64(len(content)),
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	if _, err := tw.Write(content); err != nil {
		return err
	}

	if err := tw.Close(); err != nil {
		return err
	}

	// Copy to container
	return c.client.CopyToContainer(ctx, c.containerID, filepath.Dir(destPath), &buf, container.CopyToContainerOptions{})
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
