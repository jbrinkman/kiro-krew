package sandbox

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed testdata/github-cli-mock/*
var githubCliMock embed.FS

// BuildContext manages preparation of Docker build context with all required files
type BuildContext struct {
	TempDir      string
	KrewBinary   string
	Files        map[string][]byte
	cleanupFuncs []func() error
}

// PrepareBuildContext creates a temporary directory with all files needed for Docker build
func PrepareBuildContext(krewBinary string) (*BuildContext, error) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "kiro-krew-build-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp directory: %w", err)
	}

	bc := &BuildContext{
		TempDir:    tempDir,
		KrewBinary: krewBinary,
		Files:      make(map[string][]byte),
		cleanupFuncs: []func() error{
			func() error { return os.RemoveAll(tempDir) },
		},
	}

	// Add all required files to build context
	if err := bc.AddKrewBinary(); err != nil {
		bc.Cleanup()
		return nil, fmt.Errorf("adding kiro-krew binary: %w", err)
	}

	if err := bc.AddAgentConfigs(); err != nil {
		bc.Cleanup()
		return nil, fmt.Errorf("adding agent configs: %w", err)
	}

	if err := bc.AddMockFiles(); err != nil {
		bc.Cleanup()
		return nil, fmt.Errorf("adding mock files: %w", err)
	}

	if err := bc.AddEvaluationFiles(); err != nil {
		bc.Cleanup()
		return nil, fmt.Errorf("adding evaluation files: %w", err)
	}

	return bc, nil
}

// AddKrewBinary copies the kiro-krew binary to build context
func (bc *BuildContext) AddKrewBinary() error {
	krewPath := bc.KrewBinary
	if krewPath == "" {
		// Try to find kiro-krew in PATH
		path, err := exec.LookPath("kiro-krew")
		if err != nil {
			return fmt.Errorf("kiro-krew binary not found in PATH and no explicit path provided: %w", err)
		}
		krewPath = path
	}

	// Copy binary to build context
	destPath := filepath.Join(bc.TempDir, "kiro-krew")
	content, err := os.ReadFile(krewPath)
	if err != nil {
		return fmt.Errorf("reading kiro-krew binary from %s: %w", krewPath, err)
	}

	if err := os.WriteFile(destPath, content, 0755); err != nil {
		return fmt.Errorf("writing kiro-krew binary to build context: %w", err)
	}

	bc.Files["kiro-krew"] = content
	return nil
}

// AddAgentConfigs copies .kiro/agents/ directory to build context
func (bc *BuildContext) AddAgentConfigs() error {
	// Look for .kiro/agents in current working directory and parent directories
	agentDir, err := bc.findAgentDirectory()
	if err != nil {
		return fmt.Errorf("finding agent directory: %w", err)
	}

	// Create .kiro/agents in build context
	destDir := filepath.Join(bc.TempDir, ".kiro", "agents")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("creating agent directory in build context: %w", err)
	}

	// Copy all agent files
	return filepath.Walk(agentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(agentDir, path)
		if err != nil {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading agent config %s: %w", path, err)
		}

		destPath := filepath.Join(destDir, relPath)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", destPath, err)
		}

		if err := os.WriteFile(destPath, content, 0644); err != nil {
			return fmt.Errorf("writing agent config to build context: %w", err)
		}

		bc.Files[filepath.Join(".kiro/agents", relPath)] = content
		return nil
	})
}

// AddMockFiles copies GitHub CLI mock files to build context
func (bc *BuildContext) AddMockFiles() error {
	// Create github-cli-mock directory in build context
	destDir := filepath.Join(bc.TempDir, "github-cli-mock")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("creating mock directory in build context: %w", err)
	}

	// Copy files from embedded filesystem
	return fs.WalkDir(githubCliMock, "testdata/github-cli-mock", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel("testdata/github-cli-mock", path)
		if err != nil {
			return err
		}

		content, err := githubCliMock.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading mock file %s: %w", path, err)
		}

		destPath := filepath.Join(destDir, relPath)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", destPath, err)
		}

		mode := os.FileMode(0644)
		if strings.Contains(relPath, "gh") || strings.HasSuffix(relPath, ".sh") {
			mode = 0755 // Make executable
		}

		if err := os.WriteFile(destPath, content, mode); err != nil {
			return fmt.Errorf("writing mock file to build context: %w", err)
		}

		bc.Files[filepath.Join("github-cli-mock", relPath)] = content
		return nil
	})
}

// AddEvaluationFiles copies .kiro-krew/evals/ directory to build context if it exists
func (bc *BuildContext) AddEvaluationFiles() error {
	// Look for .kiro-krew/evals in current working directory and parent directories
	evalsDir, err := bc.findEvalsDirectory()
	if err != nil {
		// Evaluation files are optional, so just log and continue
		return nil
	}

	// Create .kiro-krew/evals in build context
	destDir := filepath.Join(bc.TempDir, ".kiro-krew", "evals")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("creating evals directory in build context: %w", err)
	}

	// Copy all evaluation files
	return filepath.Walk(evalsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(evalsDir, path)
		if err != nil {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading eval file %s: %w", path, err)
		}

		destPath := filepath.Join(destDir, relPath)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", destPath, err)
		}

		if err := os.WriteFile(destPath, content, 0644); err != nil {
			return fmt.Errorf("writing eval file to build context: %w", err)
		}

		bc.Files[filepath.Join(".kiro-krew/evals", relPath)] = content
		return nil
	})
}

// findAgentDirectory searches for .kiro/agents directory in current and parent directories
func (bc *BuildContext) findAgentDirectory() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		agentDir := filepath.Join(dir, ".kiro", "agents")
		if info, err := os.Stat(agentDir); err == nil && info.IsDir() {
			return agentDir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // Reached filesystem root
		}
		dir = parent
	}

	return "", fmt.Errorf(".kiro/agents directory not found in current directory or any parent directory")
}

// findEvalsDirectory searches for .kiro-krew/evals directory in current and parent directories
func (bc *BuildContext) findEvalsDirectory() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		evalsDir := filepath.Join(dir, ".kiro-krew", "evals")
		if info, err := os.Stat(evalsDir); err == nil && info.IsDir() {
			return evalsDir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // Reached filesystem root
		}
		dir = parent
	}

	return "", fmt.Errorf(".kiro-krew/evals directory not found")
}

// Cleanup removes temporary files and directories
func (bc *BuildContext) Cleanup() error {
	var errors []string
	for _, cleanup := range bc.cleanupFuncs {
		if err := cleanup(); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %s", strings.Join(errors, "; "))
	}

	return nil
}
