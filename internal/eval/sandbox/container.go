package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// Container manages Docker container lifecycle
type Container struct {
	client      *client.Client
	containerID string
	imageName   string
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
	// Pull image if not exists
	_, _, err := c.client.ImageInspectWithRaw(ctx, c.imageName)
	if err != nil {
		reader, err := c.client.ImagePull(ctx, c.imageName, image.PullOptions{})
		if err != nil {
			return err
		}
		defer reader.Close()
		io.Copy(io.Discard, reader)
	}

	resp, err := c.client.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
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

// Close closes the Docker client
func (c *Container) Close() error {
	return c.client.Close()
}
