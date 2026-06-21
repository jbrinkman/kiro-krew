package sandbox

import (
	"os"
	"path/filepath"
)

// ProjectType represents supported project types
type ProjectType string

const (
	ProjectTypeGo     ProjectType = "go"
	ProjectTypeNodeJS ProjectType = "nodejs"
	ProjectTypePython ProjectType = "python"
	ProjectTypeRust   ProjectType = "rust"
	ProjectTypeJava   ProjectType = "java"
	ProjectTypeTask   ProjectType = "task"
)

// ProjectInfo contains detection results
type ProjectInfo struct {
	Type  ProjectType
	Files []string
}

// DetectProject analyzes project files to determine type
func DetectProject(projectPath string) []ProjectInfo {
	var projects []ProjectInfo

	// Check for Go
	if fileExists(filepath.Join(projectPath, "go.mod")) || fileExists(filepath.Join(projectPath, "go.sum")) {
		projects = append(projects, ProjectInfo{Type: ProjectTypeGo, Files: []string{"go.mod", "go.sum"}})
	}

	// Check for Node.js
	if fileExists(filepath.Join(projectPath, "package.json")) {
		projects = append(projects, ProjectInfo{Type: ProjectTypeNodeJS, Files: []string{"package.json"}})
	}

	// Check for Python
	if fileExists(filepath.Join(projectPath, "requirements.txt")) || fileExists(filepath.Join(projectPath, "pyproject.toml")) {
		projects = append(projects, ProjectInfo{Type: ProjectTypePython, Files: []string{"requirements.txt", "pyproject.toml"}})
	}

	// Check for Rust
	if fileExists(filepath.Join(projectPath, "Cargo.toml")) {
		projects = append(projects, ProjectInfo{Type: ProjectTypeRust, Files: []string{"Cargo.toml"}})
	}

	// Check for Java
	if fileExists(filepath.Join(projectPath, "pom.xml")) || fileExists(filepath.Join(projectPath, "build.gradle")) {
		projects = append(projects, ProjectInfo{Type: ProjectTypeJava, Files: []string{"pom.xml", "build.gradle"}})
	}

	// Check for Task runner
	if fileExists(filepath.Join(projectPath, "Taskfile.yml")) {
		projects = append(projects, ProjectInfo{Type: ProjectTypeTask, Files: []string{"Taskfile.yml"}})
	}

	return projects
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
