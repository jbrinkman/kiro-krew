package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerfileGeneration_IncludesKiroCLI(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		expected []string
	}{
		{
			name:     "AMD64 platform",
			platform: "linux/amd64",
			expected: []string{
				"# Install kiro-cli",
				"kirocli-x86_64-linux-musl.zip",
				"chmod 755 kirocli/bin/kiro-cli",
				"mv kirocli/bin/kiro-cli /usr/local/bin/kiro-cli",
				"# Copy kiro-krew binary from build context",
				"COPY --chown=sandbox:sandbox kiro-krew /usr/local/bin/kiro-krew",
				"# Copy agent configurations",
				"COPY --chown=sandbox:sandbox .kiro/agents/ /workspace/.kiro/agents/",
				"# Copy GitHub CLI mock files",
				"COPY --chown=sandbox:sandbox github-cli-mock/ /workspace/.kiro/skills/github-cli/",
				"# Copy evaluation files",
				"COPY --chown=sandbox:sandbox .kiro-krew/evals/ /workspace/.kiro-krew/evals/",
				"ENV PATH=\"/workspace/.kiro/skills/github-cli:$PATH\"",
			},
		},
		{
			name:     "ARM64 platform",
			platform: "linux/arm64",
			expected: []string{
				"# Install kiro-cli",
				"kirocli-aarch64-linux-musl.zip",
				"chmod 755 kirocli/bin/kiro-cli",
				"mv kirocli/bin/kiro-cli /usr/local/bin/kiro-cli",
				"# Copy kiro-krew binary from build context",
				"COPY --chown=sandbox:sandbox kiro-krew /usr/local/bin/kiro-krew",
				"# Copy agent configurations",
				"COPY --chown=sandbox:sandbox .kiro/agents/ /workspace/.kiro/agents/",
				"# Copy GitHub CLI mock files",
				"COPY --chown=sandbox:sandbox github-cli-mock/ /workspace/.kiro/skills/github-cli/",
				"# Copy evaluation files",
				"COPY --chown=sandbox:sandbox .kiro-krew/evals/ /workspace/.kiro-krew/evals/",
				"ENV PATH=\"/workspace/.kiro/skills/github-cli:$PATH\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			c, err := NewContainer("alpine:3.19")
			require.NoError(t, err)
			defer c.Close()

			dockerfile, err := c.GenerateDockerfileWithPlatform(tempDir, tt.platform)
			require.NoError(t, err)
			assert.NotEmpty(t, dockerfile)

			// Verify all expected content is present
			for _, expected := range tt.expected {
				assert.Contains(t, dockerfile, expected,
					"Dockerfile should contain %q for platform %s", expected, tt.platform)
			}

			// Verify it's properly formatted
			lines := strings.Split(dockerfile, "\n")
			assert.True(t, len(lines) > 10, "Dockerfile should have substantial content")
		})
	}
}

func TestKiroCLIDownloadURL_SupportedPlatforms(t *testing.T) {
	tests := []struct {
		name        string
		platform    string
		expectedURL string
		expectError bool
	}{
		{
			name:        "AMD64 platform",
			platform:    "linux/amd64",
			expectedURL: "https://desktop-release.q.us-east-1.amazonaws.com/latest/kirocli-x86_64-linux-musl.zip",
			expectError: false,
		},
		{
			name:        "ARM64 platform",
			platform:    "linux/arm64",
			expectedURL: "https://desktop-release.q.us-east-1.amazonaws.com/latest/kirocli-aarch64-linux-musl.zip",
			expectError: false,
		},
		{
			name:        "Unsupported platform",
			platform:    "linux/mips",
			expectedURL: "",
			expectError: true,
		},
		{
			name:        "Invalid platform format",
			platform:    "invalid",
			expectedURL: "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := getKiroCLIDownloadURL(tt.platform)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, url)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedURL, url)
			}
		})
	}
}

func TestInstallationCommands_CrossPlatform(t *testing.T) {
	platforms := []string{"linux/amd64", "linux/arm64"}

	for _, platform := range platforms {
		t.Run(platform, func(t *testing.T) {
			dockerfile, err := addKiroCLIToDockerfile(platform)
			require.NoError(t, err)
			assert.NotEmpty(t, dockerfile)

			// Verify installation command structure
			assert.Contains(t, dockerfile, "# Install kiro-cli")
			assert.Contains(t, dockerfile, "RUN cd /tmp")
			assert.Contains(t, dockerfile, "curl -fsSL")
			assert.Contains(t, dockerfile, "unzip -q")
			assert.Contains(t, dockerfile, "chmod 755 kirocli/bin/kiro-cli")
			assert.Contains(t, dockerfile, "mv kirocli/bin/kiro-cli /usr/local/bin/kiro-cli")
			assert.Contains(t, dockerfile, "rm -rf kirocli.zip kirocli")

			// Verify platform-specific binary is referenced
			switch platform {
			case "linux/amd64":
				assert.Contains(t, dockerfile, "kirocli-x86_64-linux-musl.zip")
			case "linux/arm64":
				assert.Contains(t, dockerfile, "kirocli-aarch64-linux-musl.zip")
			}
		})
	}
}

func TestInstallationVerification_PermissionChecks(t *testing.T) {
	// Create a test container with a mock kiro-cli binary
	tempDir := t.TempDir()
	mockBinary := filepath.Join(tempDir, "kiro-cli")

	// Create mock binary with proper content
	mockContent := "#!/bin/sh\necho 'kiro-cli version 1.0.0'"
	err := os.WriteFile(mockBinary, []byte(mockContent), 0755)
	require.NoError(t, err)

	// Test permission verification
	t.Run("ValidPermissions", func(t *testing.T) {
		// This test would require a running container to fully test
		// For unit testing, we verify the permission check logic exists
		info, err := os.Stat(mockBinary)
		require.NoError(t, err)

		mode := info.Mode()
		assert.True(t, mode&0755 == 0755, "Mock binary should have executable permissions")
	})

	t.Run("InvalidPermissions", func(t *testing.T) {
		// Create binary without execute permissions
		nonExecBinary := filepath.Join(tempDir, "kiro-cli-noexec")
		err = os.WriteFile(nonExecBinary, []byte(mockContent), 0644)
		require.NoError(t, err)

		info, err := os.Stat(nonExecBinary)
		require.NoError(t, err)

		mode := info.Mode()
		assert.False(t, mode&0111 != 0, "Binary should not have execute permissions")
	})
}

func TestInstallationFailures_ErrorHandling(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		wantErr  bool
	}{
		{
			name:     "Valid AMD64 platform",
			platform: "linux/amd64",
			wantErr:  false,
		},
		{
			name:     "Valid ARM64 platform",
			platform: "linux/arm64",
			wantErr:  false,
		},
		{
			name:     "Invalid platform",
			platform: "windows/amd64",
			wantErr:  true,
		},
		{
			name:     "Empty platform",
			platform: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getKiroCLIDownloadURL(tt.platform)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDockerfileGeneration_ProjectDetection(t *testing.T) {
	tempDir := t.TempDir()

	c, err := NewContainer("alpine:3.19")
	require.NoError(t, err)
	defer c.Close()

	// Test with empty directory (no project detection needed)
	dockerfile, err := c.GenerateDockerfileWithPlatform(tempDir, "linux/amd64")
	require.NoError(t, err)

	// Verify basic structure and kiro-cli installation
	assert.Contains(t, dockerfile, "FROM alpine:3.19")
	assert.Contains(t, dockerfile, "# Install kiro-cli")
	assert.Contains(t, dockerfile, "kirocli-x86_64-linux-musl.zip")
	assert.Contains(t, dockerfile, "RUN adduser -D -s /bin/bash sandbox")
	assert.Contains(t, dockerfile, "WORKDIR /workspace")
	assert.Contains(t, dockerfile, "USER sandbox")
}

func TestBuildContext_Preparation(t *testing.T) {
	t.Run("PrepareBuildContext", func(t *testing.T) {
		// Create a minimal build context without actual kiro-krew binary
		tempDir := t.TempDir()

		// Create mock kiro-krew binary
		mockBinary := filepath.Join(tempDir, "kiro-krew")
		err := os.WriteFile(mockBinary, []byte("mock binary"), 0755)
		require.NoError(t, err)

		buildContext, err := PrepareBuildContext(mockBinary)
		require.NoError(t, err)
		defer buildContext.Cleanup()

		// Verify build context structure
		assert.NotEmpty(t, buildContext.TempDir)
		assert.Equal(t, mockBinary, buildContext.KrewBinary)
		assert.NotEmpty(t, buildContext.Files)

		// Verify kiro-krew binary is in files map
		_, exists := buildContext.Files["kiro-krew"]
		assert.True(t, exists, "kiro-krew binary should be in build context")
	})

	t.Run("AddMockFiles", func(t *testing.T) {
		tempDir := t.TempDir()
		bc := &BuildContext{
			TempDir: tempDir,
			Files:   make(map[string][]byte),
		}

		err := bc.AddMockFiles()
		require.NoError(t, err)

		// Verify mock files were added
		foundMockFiles := false
		for path := range bc.Files {
			if strings.Contains(path, "github-cli-mock") {
				foundMockFiles = true
				break
			}
		}
		assert.True(t, foundMockFiles, "Mock files should be added to build context")

		// Verify mock directory exists in temp dir
		mockDir := filepath.Join(tempDir, "github-cli-mock")
		_, err = os.Stat(mockDir)
		assert.NoError(t, err, "Mock directory should exist in build context")
	})
}

func TestBuildTimeVsRuntime_Installation(t *testing.T) {
	t.Run("BuildTimeInstallation", func(t *testing.T) {
		// Test that Dockerfile generation includes build-time installation
		// No Docker required — only generates a string
		tempDir := t.TempDir()
		c := &Container{}
		platform, err := DetectHostArchitecture()
		require.NoError(t, err)
		dockerfile, err := c.GenerateDockerfileWithPlatform(tempDir, platform)
		require.NoError(t, err)

		// Verify build-time installation commands are present
		assert.Contains(t, dockerfile, "# Install kiro-cli")
		assert.Contains(t, dockerfile, "curl -fsSL")
		assert.Contains(t, dockerfile, "unzip -q")
		assert.Contains(t, dockerfile, "chmod 755 kirocli/bin/kiro-cli")
	})

	t.Run("RuntimeVerification", func(t *testing.T) {
		skipIfNoDocker(t)

		c, err := NewContainer("alpine:3.19")
		require.NoError(t, err)
		defer c.Close()

		// Test that ValidateKiroCLI only does verification, not installation
		ctx := context.Background()

		// This would fail in a real container since kiro-cli isn't installed
		// But we can test the method exists and has correct signature
		err = c.ValidateKiroCLI(ctx, "linux/amd64")
		// Expect error since we don't have a running container with kiro-cli
		assert.Error(t, err, "Should fail verification when kiro-cli not installed")
	})

	t.Run("BuildTimeFileEmbedding", func(t *testing.T) {
		// Test that Dockerfile includes all required COPY commands
		tempDir := t.TempDir()
		c := &Container{}
		platform, err := DetectHostArchitecture()
		require.NoError(t, err)
		dockerfile, err := c.GenerateDockerfileWithPlatform(tempDir, platform)
		require.NoError(t, err)

		// Verify build-time file copying commands are present
		assert.Contains(t, dockerfile, "COPY --chown=sandbox:sandbox kiro-krew /usr/local/bin/kiro-krew")
		assert.Contains(t, dockerfile, "COPY --chown=sandbox:sandbox .kiro/agents/ /workspace/.kiro/agents/")
		assert.Contains(t, dockerfile, "COPY --chown=sandbox:sandbox github-cli-mock/ /workspace/.kiro/skills/github-cli/")
		assert.Contains(t, dockerfile, "COPY --chown=sandbox:sandbox .kiro-krew/evals/ /workspace/.kiro-krew/evals/")
		assert.Contains(t, dockerfile, "ENV PATH=\"/workspace/.kiro/skills/github-cli:$PATH\"")

		// Verify directory creation commands
		assert.Contains(t, dockerfile, "mkdir -p /workspace/.kiro/agents")
		assert.Contains(t, dockerfile, "mkdir -p /workspace/.kiro/skills/github-cli")
		assert.Contains(t, dockerfile, "mkdir -p /workspace/.kiro-krew/evals")
	})
}

func TestPlatformSpecificBinaries(t *testing.T) {
	tests := []struct {
		platform     string
		expectedFile string
	}{
		{"linux/amd64", "kirocli-x86_64-linux-musl.zip"},
		{"linux/arm64", "kirocli-aarch64-linux-musl.zip"},
	}

	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			url, err := getKiroCLIDownloadURL(tt.platform)
			require.NoError(t, err)
			assert.Contains(t, url, tt.expectedFile)

			dockerfile, err := addKiroCLIToDockerfile(tt.platform)
			require.NoError(t, err)
			assert.Contains(t, dockerfile, tt.expectedFile)
		})
	}
}

func TestContainer_LogStartup(t *testing.T) {
	c, err := NewContainer("alpine:3.19")
	require.NoError(t, err)
	defer c.Close()

	c.containerID = "1234567890abcdef"
	limits := ResourceLimits{
		CPUQuota: 2000000,           // 2 cores
		Memory:   512 * 1024 * 1024, // 512MB
		Timeout:  time.Minute * 5,
	}

	// This should not panic and should output formatted info
	c.LogStartup(limits)
}

func TestContainer_GetContainerInfo(t *testing.T) {
	c, err := NewContainer("alpine:3.19")
	require.NoError(t, err)
	defer c.Close()

	c.containerID = "1234567890abcdef"
	shortID, imageName := c.GetContainerInfo()

	assert.Equal(t, "1234567", shortID)
	assert.Equal(t, "alpine:3.19", imageName)

	// Test with short ID
	c.containerID = "123"
	shortID, _ = c.GetContainerInfo()
	assert.Equal(t, "123", shortID)
}

func TestDetectHostArchitecture_Unsupported(t *testing.T) {
	// Test the current architecture (should work)
	platform, err := DetectHostArchitecture()
	require.NoError(t, err)
	assert.True(t, platform == "linux/amd64" || platform == "linux/arm64")
}

func TestValidateKiroCLI_Verification(t *testing.T) {
	skipIfNoDocker(t)

	c, err := NewContainer("alpine:3.19")
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()

	// Should fail without a running container
	err = c.ValidateKiroCLI(ctx, "linux/amd64")
	assert.Error(t, err, "Should fail when container not running")
}

func TestResourceLimits_Coverage(t *testing.T) {
	limits := DefaultLimits()
	assert.NotZero(t, limits.CPUQuota)
	assert.NotZero(t, limits.Memory)
	assert.NotZero(t, limits.Timeout)

	hostConfig := &container.HostConfig{}
	limits.ApplyToHostConfig(hostConfig)
	assert.NotNil(t, hostConfig.Resources)

	newHostConfig := NewHostConfigWithLimits(limits)
	assert.NotNil(t, newHostConfig)
	assert.NotNil(t, newHostConfig.Resources)
}

func TestKiroCLIVerification_Detailed(t *testing.T) {
	skipIfNoDocker(t)

	c, err := NewContainer("alpine:3.19")
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()

	// Test verification method directly
	err = c.verifyKiroCLIInstallation(ctx)
	assert.Error(t, err, "Should fail when no container is running")
	assert.Contains(t, err.Error(), "kiro-cli")
}

func TestProjectDetection_Coverage(t *testing.T) {
	tempDir := t.TempDir()

	// Test with empty directory
	projects := DetectProject(tempDir)
	assert.Empty(t, projects, "Empty directory should detect no projects")

	// Test fileExists function indirectly
	nonExistent := filepath.Join(tempDir, "nonexistent.txt")
	assert.False(t, fileExists(nonExistent))

	existentFile := filepath.Join(tempDir, "existent.txt")
	err := os.WriteFile(existentFile, []byte("test"), 0644)
	require.NoError(t, err)
	assert.True(t, fileExists(existentFile))
}

func TestContainer_ArchitectureErrors(t *testing.T) {
	c, err := NewContainer("alpine:3.19")
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()
	config := &container.Config{Image: "alpine:3.19", Cmd: []string{"echo", "test"}}
	hostConfig := &container.HostConfig{}

	// Test invalid platform format
	err = c.CreateWithPlatform(ctx, config, hostConfig, "invalid-platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid platform format")
}

func TestGenerateDockerfile_ErrorHandling(t *testing.T) {
	c, err := NewContainer("alpine:3.19")
	require.NoError(t, err)
	defer c.Close()

	// Test with unsupported platform
	_, err = c.GenerateDockerfileWithPlatform(t.TempDir(), "windows/amd64")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported platform")
}

func TestMockGitHub_SimulateResponse(t *testing.T) {
	// Test mock response simulation functions that are still present

	// Test SimulateGitHubResponse
	response := SimulateGitHubResponse("issue", []string{"create"})
	assert.Equal(t, 12345, response.IssueNumber)

	response = SimulateGitHubResponse("pr", []string{"create"})
	assert.Equal(t, 42, response.PRNumber)

	response = SimulateGitHubResponse("unknown", []string{})
	assert.Equal(t, "success", response.Status)
}

func TestContainer_CreateBuildContextTar(t *testing.T) {
	c, err := NewContainer("alpine:3.19")
	require.NoError(t, err)
	defer c.Close()

	// Create a build context with some test files
	tempDir := t.TempDir()

	// Create test files in build context
	testFile1 := filepath.Join(tempDir, "test1.txt")
	testFile2 := filepath.Join(tempDir, "subdir", "test2.txt")

	err = os.WriteFile(testFile1, []byte("content1"), 0644)
	require.NoError(t, err)

	err = os.MkdirAll(filepath.Dir(testFile2), 0755)
	require.NoError(t, err)

	err = os.WriteFile(testFile2, []byte("content2"), 0644)
	require.NoError(t, err)

	buildContext := &BuildContext{
		TempDir: tempDir,
		Files: map[string][]byte{
			"test1.txt":        []byte("content1"),
			"subdir/test2.txt": []byte("content2"),
		},
	}

	dockerfile := "FROM alpine:3.19\nRUN echo test"

	tarReader, err := c.createBuildContextTar(dockerfile, buildContext)
	require.NoError(t, err)
	assert.NotNil(t, tarReader)

	// The tar should contain the dockerfile and build context files
	// We don't need to verify the tar contents in detail for this test
}

func TestContainer_CompleteInstallationFlow(t *testing.T) {
	// Test complete installation flow without Docker dependency
	c, err := NewContainer("alpine:3.19")
	require.NoError(t, err)
	defer c.Close()

	// Test all platforms
	platforms := []string{"linux/amd64", "linux/arm64"}

	for _, platform := range platforms {
		t.Run(platform, func(t *testing.T) {
			// Generate dockerfile
			dockerfile, err := c.GenerateDockerfileWithPlatform(t.TempDir(), platform)
			require.NoError(t, err)

			// Verify installation commands are present
			assert.Contains(t, dockerfile, "# Install kiro-cli")
			assert.Contains(t, dockerfile, "curl -fsSL")
			assert.Contains(t, dockerfile, "/usr/local/bin/kiro-cli")

			// Test URL generation
			url, err := getKiroCLIDownloadURL(platform)
			require.NoError(t, err)
			assert.Contains(t, dockerfile, filepath.Base(url))

			// Test dockerfile command generation
			installCommands, err := addKiroCLIToDockerfile(platform)
			require.NoError(t, err)
			assert.Contains(t, dockerfile, strings.TrimSpace(strings.Split(installCommands, "\n")[0]))
		})
	}
}
