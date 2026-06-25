package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

// Container manages Docker container lifecycle
type Container struct {
	client      *client.Client
	containerID string
	imageName   string
}

// DetectHostArchitecture returns the Docker platform string for the host architecture
func DetectHostArchitecture() (string, error) {
	switch runtime.GOARCH {
	case "amd64":
		return "linux/amd64", nil
	case "arm64":
		return "linux/arm64", nil
	default:
		return "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}
}

// LogStartup displays container creation details with resource limits
func (c *Container) LogStartup(limits ResourceLimits) {
	shortID := c.containerID
	if len(shortID) > 7 {
		shortID = shortID[:7]
	}

	// Convert resource limits to human-readable format
	cpuCores := float64(limits.CPUQuota) / 1000000.0
	memoryMB := limits.Memory / (1024 * 1024)

	fmt.Printf("🐳 Starting sandbox container: %s [%s] (%.1f CPU, %dMB RAM, %v timeout)\n",
		c.imageName, shortID, cpuCores, memoryMB, limits.Timeout)
}

// GetContainerInfo returns container ID and resource information
func (c *Container) GetContainerInfo() (string, string) {
	shortID := c.containerID
	if len(shortID) > 7 {
		shortID = shortID[:7]
	}
	return shortID, c.imageName
}

// NewContainer creates a new container manager
func NewContainer(imageName string) (*Container, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	// Verify Docker is running
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := cli.Ping(ctx); err != nil {
		cli.Close()
		return nil, fmt.Errorf("Docker is not running. Start Docker and try again: %w", err)
	}

	return &Container{
		client:    cli,
		imageName: imageName,
	}, nil
}

// Create creates a Docker container
func (c *Container) Create(ctx context.Context, config *container.Config, hostConfig *container.HostConfig) error {
	// Detect host architecture for platform specification
	platformStr, err := DetectHostArchitecture()
	if err != nil {
		return fmt.Errorf("detecting host architecture: %w", err)
	}

	return c.CreateWithPlatform(ctx, config, hostConfig, platformStr)
}

// CreateWithPlatform creates a Docker container with specified platform
func (c *Container) CreateWithPlatform(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, platform string) error {

	// Pull image if not exists
	_, _, err := c.client.ImageInspectWithRaw(ctx, c.imageName)
	if err != nil {
		pullOptions := image.PullOptions{
			Platform: platform,
		}
		reader, err := c.client.ImagePull(ctx, c.imageName, pullOptions)
		if err != nil {
			return err
		}
		defer reader.Close()
		io.Copy(io.Discard, reader)
	}

	// Parse platform string for ContainerCreate
	parts := strings.Split(platform, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid platform format %q: expected OS/arch", platform)
	}
	platformSpec := &specs.Platform{
		OS:           parts[0],
		Architecture: parts[1],
	}

	resp, err := c.client.ContainerCreate(ctx, config, hostConfig, nil, platformSpec, "")
	if err != nil {
		return err
	}

	c.containerID = resp.ID
	return nil
}

// Start starts the container
func (c *Container) Start(ctx context.Context) error {
	return c.client.ContainerStart(ctx, c.containerID, container.StartOptions{})
}

// CopyTo copies a file to the container
func (c *Container) CopyTo(ctx context.Context, destPath string, srcPath string) error {
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	// Docker CopyToContainer requires a tar archive
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	header := &tar.Header{
		Name: filepath.Base(destPath),
		Mode: 0755,
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

	return c.client.CopyToContainer(ctx, c.containerID, filepath.Dir(destPath), &buf, container.CopyToContainerOptions{})
}

// Exec executes a command in the container
func (c *Container) Exec(ctx context.Context, cmd []string) error {
	execConfig := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	resp, err := c.client.ContainerExecCreate(ctx, c.containerID, execConfig)
	if err != nil {
		return err
	}

	return c.client.ContainerExecStart(ctx, resp.ID, container.ExecStartOptions{})
}

// ExecWithOutput executes a command and returns output
func (c *Container) ExecWithOutput(ctx context.Context, cmd []string) (string, error) {
	execConfig := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	resp, err := c.client.ContainerExecCreate(ctx, c.containerID, execConfig)
	if err != nil {
		return "", err
	}

	hijacked, err := c.client.ContainerExecAttach(ctx, resp.ID, container.ExecStartOptions{})
	if err != nil {
		return "", err
	}
	defer hijacked.Close()

	output, err := io.ReadAll(hijacked.Reader)
	if err != nil {
		return "", err
	}

	// Check execution result for exit code
	inspect, err := c.client.ContainerExecInspect(ctx, resp.ID)
	if err != nil {
		return "", err
	}

	if inspect.ExitCode != 0 {
		return strings.TrimSpace(string(output)), fmt.Errorf("command failed with exit code %d: %s", inspect.ExitCode, strings.TrimSpace(string(output)))
	}

	return strings.TrimSpace(string(output)), nil
}

// Cleanup stops and removes the container
func (c *Container) Cleanup(ctx context.Context) error {
	if c.containerID == "" {
		return nil
	}

	timeout := 10 * time.Second
	timeoutSec := int(timeout.Seconds())
	err := c.client.ContainerStop(ctx, c.containerID, container.StopOptions{Timeout: &timeoutSec})
	if err != nil {
		return err
	}

	return c.client.ContainerRemove(ctx, c.containerID, container.RemoveOptions{Force: true})
}

// GenerateDockerfile creates a custom Dockerfile based on project detection
func (c *Container) GenerateDockerfile(projectPath string) (string, error) {
	projects := DetectProject(projectPath)

	var dockerfile strings.Builder

	// Start with base image
	dockerfile.WriteString("FROM alpine:3.19\n\n")

	// Install essential tools
	dockerfile.WriteString("RUN apk add --no-cache \\\n")
	dockerfile.WriteString("    git \\\n")
	dockerfile.WriteString("    curl \\\n")
	dockerfile.WriteString("    bash \\\n")
	dockerfile.WriteString("    unzip \\\n")
	dockerfile.WriteString("    ca-certificates\n\n")

	// Add toolchain installations
	for _, project := range projects {
		template, err := c.loadTemplate(project.Type)
		if err != nil {
			return "", fmt.Errorf("loading template for %s: %w", project.Type, err)
		}
		dockerfile.WriteString(template)
		dockerfile.WriteString("\n")
	}

	// Add user and workspace setup
	dockerfile.WriteString("RUN adduser -D -s /bin/bash sandbox\n")
	dockerfile.WriteString("WORKDIR /workspace\n")
	dockerfile.WriteString("USER sandbox\n")
	dockerfile.WriteString("CMD [\"/bin/bash\"]\n")

	return dockerfile.String(), nil
}

func (c *Container) loadTemplate(projectType ProjectType) (string, error) {
	templatePath := fmt.Sprintf("internal/eval/dockerfile/templates/%s.Dockerfile", string(projectType))
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// getKiroCLIDownloadURL returns the direct download URL for kiro-cli binary
func getKiroCLIDownloadURL(platform string) (string, error) {
	baseURL := "https://desktop-release.q.us-east-1.amazonaws.com/latest/"
	switch platform {
	case "linux/amd64":
		return baseURL + "kirocli-x86_64-linux-musl.zip", nil
	case "linux/arm64":
		return baseURL + "kirocli-aarch64-linux-musl.zip", nil
	default:
		return "", fmt.Errorf("unsupported platform: %s", platform)
	}
}

// InstallKiroCLI installs kiro-cli binary via direct download for the specified platform
func (c *Container) InstallKiroCLI(ctx context.Context, platform string) error {
	// Get download URL
	downloadURL, err := getKiroCLIDownloadURL(platform)
	if err != nil {
		return err
	}

	// Download, extract, and install kiro-cli (runs as root in container setup phase)
	installCmd := fmt.Sprintf(`
		cd /tmp && \
		curl -fsSL %s -o kirocli.zip && \
		unzip -q kirocli.zip && \
		chmod 755 kiro-cli && \
		mv kiro-cli /usr/local/bin/kiro-cli
	`, downloadURL)

	if err := c.Exec(ctx, []string{"sh", "-c", installCmd}); err != nil {
		return fmt.Errorf("installing kiro-cli binary: %w", err)
	}

	// Verify installation
	if err := c.verifyKiroCLIInstallation(ctx); err != nil {
		return fmt.Errorf("installation verification failed: %w", err)
	}

	return nil
}

// verifyKiroCLIInstallation checks if kiro-cli is properly installed and functional
func (c *Container) verifyKiroCLIInstallation(ctx context.Context) error {
	// Check if binary exists and is executable
	_, err := c.ExecWithOutput(ctx, []string{"test", "-x", "/usr/local/bin/kiro-cli"})
	if err != nil {
		return fmt.Errorf("kiro-cli binary not found or not executable at /usr/local/bin/kiro-cli")
	}

	// Run version command to verify functionality
	version, err := c.ExecWithOutput(ctx, []string{"kiro-cli", "--version"})
	if err != nil {
		return fmt.Errorf("kiro-cli --version command failed: %w", err)
	}

	if version == "" {
		return fmt.Errorf("kiro-cli --version returned empty output")
	}

	// Only print success message after all verification passes
	fmt.Printf("✅ kiro-cli installation verified: %s\n", version)
	return nil
}

// Close closes the Docker client
func (c *Container) Close() error {
	return c.client.Close()
}
