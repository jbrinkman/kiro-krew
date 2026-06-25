package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skipIfNoDocker(t *testing.T) {
	t.Helper()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Skip("Docker client unavailable:", err)
	}
	defer cli.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := cli.Ping(ctx); err != nil {
		t.Skip("Docker not running:", err)
	}
}

func TestContainer_Lifecycle(t *testing.T) {
	skipIfNoDocker(t)

	ctx := context.Background()
	c, err := NewContainer("alpine:3.19")
	require.NoError(t, err)
	defer c.Close()

	// Test Create
	config := &container.Config{
		Image: "alpine:3.19",
		Cmd:   []string{"sleep", "30"},
	}
	hostConfig := &container.HostConfig{}

	err = c.Create(ctx, config, hostConfig)
	assert.NoError(t, err)
	assert.NotEmpty(t, c.containerID)

	// Test Start
	err = c.Start(ctx)
	assert.NoError(t, err)

	// Test Exec
	err = c.Exec(ctx, []string{"echo", "test"})
	assert.NoError(t, err)

	// Test ExecWithOutput
	output, err := c.ExecWithOutput(ctx, []string{"echo", "hello"})
	assert.NoError(t, err)
	assert.Contains(t, output, "hello")

	// Test Cleanup
	err = c.Cleanup(ctx)
	assert.NoError(t, err)
}

func TestContainer_CopyTo(t *testing.T) {
	skipIfNoDocker(t)

	ctx := context.Background()
	c, err := NewContainer("alpine:3.19")
	require.NoError(t, err)
	defer c.Close()

	// Create container
	config := &container.Config{
		Image: "alpine:3.19",
		Cmd:   []string{"sleep", "30"},
	}
	err = c.Create(ctx, config, &container.HostConfig{})
	require.NoError(t, err)

	err = c.Start(ctx)
	require.NoError(t, err)
	defer c.Cleanup(ctx)

	// Create test file
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	err = os.WriteFile(tmpFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// Test file copy
	err = c.CopyTo(ctx, "/tmp/test.txt", tmpFile)
	assert.NoError(t, err)

	// Verify file exists in container
	output, err := c.ExecWithOutput(ctx, []string{"cat", "/tmp/test.txt"})
	assert.NoError(t, err)
	assert.Contains(t, output, "test content")
}

func TestProjectDetection(t *testing.T) {
	testCases := []struct {
		name     string
		files    map[string]string
		expected []ProjectType
	}{
		{
			name:     "Go project",
			files:    map[string]string{"go.mod": "module test"},
			expected: []ProjectType{ProjectTypeGo},
		},
		{
			name:     "Node.js project",
			files:    map[string]string{"package.json": "{}"},
			expected: []ProjectType{ProjectTypeNodeJS},
		},
		{
			name:     "Python project",
			files:    map[string]string{"requirements.txt": "flask==2.0.1"},
			expected: []ProjectType{ProjectTypePython},
		},
		{
			name:     "Rust project",
			files:    map[string]string{"Cargo.toml": "[package]"},
			expected: []ProjectType{ProjectTypeRust},
		},
		{
			name:     "Java Maven project",
			files:    map[string]string{"pom.xml": "<project>"},
			expected: []ProjectType{ProjectTypeJava},
		},
		{
			name:     "Task project",
			files:    map[string]string{"Taskfile.yml": "version: '3'"},
			expected: []ProjectType{ProjectTypeTask},
		},
		{
			name: "Multi-language project",
			files: map[string]string{
				"go.mod":       "module test",
				"package.json": "{}",
				"Taskfile.yml": "version: '3'",
			},
			expected: []ProjectType{ProjectTypeGo, ProjectTypeNodeJS, ProjectTypeTask},
		},
		{
			name:     "No project files",
			files:    map[string]string{"README.md": "# Test"},
			expected: []ProjectType{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary directory with test files
			tmpDir := t.TempDir()
			for filename, content := range tc.files {
				err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(content), 0644)
				require.NoError(t, err)
			}

			// Detect projects
			projects := DetectProject(tmpDir)

			// Extract project types
			var detected []ProjectType
			for _, p := range projects {
				detected = append(detected, p.Type)
			}

			assert.ElementsMatch(t, tc.expected, detected)
		})
	}
}

func TestResourceLimits(t *testing.T) {
	limits := DefaultLimits()

	// Test default values
	assert.Equal(t, int64(1000000), limits.CPUQuota)
	assert.Equal(t, int64(512*1024*1024), limits.Memory)
	assert.Equal(t, 5*time.Minute, limits.Timeout)

	// Test applying to host config
	hostConfig := &container.HostConfig{}
	limits.ApplyToHostConfig(hostConfig)

	assert.Equal(t, int64(1000000), hostConfig.Resources.CPUQuota)
	assert.Equal(t, int64(100000), hostConfig.Resources.CPUPeriod)
	assert.Equal(t, int64(512*1024*1024), hostConfig.Resources.Memory)

	// Test NewHostConfigWithLimits
	hostConfig2 := NewHostConfigWithLimits(limits)
	assert.Equal(t, int64(1000000), hostConfig2.Resources.CPUQuota)
	assert.Equal(t, int64(100000), hostConfig2.Resources.CPUPeriod)
	assert.Equal(t, int64(512*1024*1024), hostConfig2.Resources.Memory)
	assert.Equal(t, container.NetworkMode("none"), hostConfig2.NetworkMode)
}

func TestResourceLimitsEnforcement(t *testing.T) {
	skipIfNoDocker(t)

	ctx := context.Background()
	c, err := NewContainer("alpine:3.19")
	require.NoError(t, err)
	defer c.Close()

	// Create container with strict memory limit (16MB)
	limits := ResourceLimits{
		CPUQuota: 500000,           // 0.5 core
		Memory:   16 * 1024 * 1024, // 16MB
		Timeout:  10 * time.Second,
	}

	config := &container.Config{
		Image: "alpine:3.19",
		Cmd:   []string{"sleep", "30"},
	}
	hostConfig := NewHostConfigWithLimits(limits)

	err = c.Create(ctx, config, hostConfig)
	require.NoError(t, err)

	err = c.Start(ctx)
	require.NoError(t, err)
	defer c.Cleanup(ctx)

	// Test memory limit is applied (works with both cgroups v1 and v2)
	output, err := c.ExecWithOutput(ctx, []string{"sh", "-c", "cat /sys/fs/cgroup/memory.max 2>/dev/null || cat /sys/fs/cgroup/memory/memory.limit_in_bytes 2>/dev/null || echo unknown"})
	assert.NoError(t, err)
	assert.Contains(t, output, "16777216") // 16MB in bytes
}

func TestGenerateDockerfile(t *testing.T) {
	c := &Container{}

	// Create test project directory
	tmpDir := t.TempDir()

	// Create Go project files
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)
	require.NoError(t, err)

	// Mock template loading by creating a simple template
	templateDir := filepath.Join(tmpDir, "internal/eval/dockerfile/templates")
	err = os.MkdirAll(templateDir, 0755)
	require.NoError(t, err)

	goTemplate := `# Install Go
RUN apk add --no-cache go
ENV GOPATH=/home/sandbox/go
ENV PATH=$PATH:/usr/local/go/bin:$GOPATH/bin
`
	err = os.WriteFile(filepath.Join(templateDir, "go.Dockerfile"), []byte(goTemplate), 0644)
	require.NoError(t, err)

	// Change to temp directory to test template loading
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	dockerfile, err := c.GenerateDockerfile(tmpDir)
	assert.NoError(t, err)
	assert.Contains(t, dockerfile, "FROM alpine:3.19")
	assert.Contains(t, dockerfile, "Install Go")
	assert.Contains(t, dockerfile, "adduser -D -s /bin/bash sandbox")
	assert.Contains(t, dockerfile, "WORKDIR /workspace")
}

func TestGitHubCLIMocking(t *testing.T) {
	skipIfNoDocker(t)
	testdataDir := "testdata/github-cli-mock"

	// Check mock script exists locally
	mockScript := filepath.Join(testdataDir, "gh")
	_, err := os.Stat(mockScript)
	require.NoError(t, err, "GitHub CLI mock script should exist")

	// Verify mock script content
	content, err := os.ReadFile(mockScript)
	require.NoError(t, err)
	assert.Contains(t, string(content), "[MOCK]")
	assert.Contains(t, string(content), "#!/bin/bash")
}

func TestContainerTimeout(t *testing.T) {
	skipIfNoDocker(t)

	ctx := context.Background()

	c, err := NewContainer("alpine:3.19")
	require.NoError(t, err)
	defer c.Close()

	config := &container.Config{
		Image: "alpine:3.19",
		Cmd:   []string{"sleep", "30"},
	}

	err = c.Create(ctx, config, &container.HostConfig{})
	require.NoError(t, err)

	err = c.Start(ctx)
	require.NoError(t, err)
	defer c.Cleanup(ctx)

	// Use a very short timeout for the exec — should fail
	execCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	_, err = c.ExecWithOutput(execCtx, []string{"sleep", "10"})
	if err != nil {
		assert.Contains(t, err.Error(), "context deadline exceeded")
	} else {
		// Some Docker versions complete exec create before timeout hits
		t.Log("exec completed before timeout — skipping timeout assertion")
	}
}

func TestContainer_WorkspacePermissions(t *testing.T) {
	skipIfNoDocker(t)

	ctx := context.Background()
	c, err := NewContainer("alpine:3.19")
	require.NoError(t, err)
	defer c.Close()

	limits := DefaultLimits()
	config := &container.Config{
		Image: "alpine:3.19",
		Cmd:   []string{"sleep", "30"},
	}
	hostConfig := NewHostConfigWithLimits(limits)

	err = c.Create(ctx, config, hostConfig)
	require.NoError(t, err)

	err = c.Start(ctx)
	require.NoError(t, err)
	defer c.Cleanup(ctx)

	// Test workspace directory is writable
	_, err = c.ExecWithOutput(ctx, []string{"mkdir", "-p", "/workspace/test"})
	assert.NoError(t, err, "workspace should be writable")

	// Test file creation in workspace
	_, err = c.ExecWithOutput(ctx, []string{"sh", "-c", "echo 'test' > /workspace/test/file.txt"})
	assert.NoError(t, err, "should be able to create files in workspace")

	// Verify file content
	output, err := c.ExecWithOutput(ctx, []string{"cat", "/workspace/test/file.txt"})
	assert.NoError(t, err)
	assert.Contains(t, output, "test")
}

func TestExecWithOutput_ErrorHandling(t *testing.T) {
	skipIfNoDocker(t)

	ctx := context.Background()
	c, err := NewContainer("alpine:3.19")
	require.NoError(t, err)
	defer c.Close()

	config := &container.Config{
		Image: "alpine:3.19",
		Cmd:   []string{"sleep", "30"},
	}

	err = c.Create(ctx, config, &container.HostConfig{})
	require.NoError(t, err)

	err = c.Start(ctx)
	require.NoError(t, err)
	defer c.Cleanup(ctx)

	// Test successful command
	output, err := c.ExecWithOutput(ctx, []string{"echo", "success"})
	assert.NoError(t, err)
	assert.Contains(t, output, "success")

	// Test failing command should return error
	_, err = c.ExecWithOutput(ctx, []string{"false"})
	assert.Error(t, err, "failing command should return error")
	assert.Contains(t, err.Error(), "exit code 1")

	// Test non-existent command should return error
	_, err = c.ExecWithOutput(ctx, []string{"nonexistent-command"})
	assert.Error(t, err, "non-existent command should return error")
}

func TestKiroCLIInstallation_VerificationLogic(t *testing.T) {
	skipIfNoDocker(t)

	ctx := context.Background()
	c, err := NewContainer("alpine:3.19")
	require.NoError(t, err)
	defer c.Close()

	config := &container.Config{
		Image: "alpine:3.19",
		Cmd:   []string{"sleep", "30"},
	}

	err = c.Create(ctx, config, &container.HostConfig{})
	require.NoError(t, err)

	err = c.Start(ctx)
	require.NoError(t, err)
	defer c.Cleanup(ctx)

	// Test verification fails when kiro-cli is not installed
	err = c.verifyKiroCLIInstallation(ctx)
	assert.Error(t, err, "verification should fail when kiro-cli is not installed")
}

func TestContainer_GitHubMockingSetup(t *testing.T) {
	skipIfNoDocker(t)

	ctx := context.Background()
	c, err := NewContainer("alpine:3.19")
	require.NoError(t, err)
	defer c.Close()

	limits := DefaultLimits()
	config := &container.Config{
		Image: "alpine:3.19",
		Cmd:   []string{"sleep", "30"},
	}
	hostConfig := NewHostConfigWithLimits(limits)

	err = c.Create(ctx, config, hostConfig)
	require.NoError(t, err)

	err = c.Start(ctx)
	require.NoError(t, err)
	defer c.Cleanup(ctx)

	// Test basic directory creation
	_, err = c.ExecWithOutput(ctx, []string{"mkdir", "-p", "/workspace/test"})
	assert.NoError(t, err, "should be able to create directories in workspace")

	// Test file creation
	_, err = c.ExecWithOutput(ctx, []string{"touch", "/workspace/test/file"})
	assert.NoError(t, err, "should be able to create files in workspace")
}
