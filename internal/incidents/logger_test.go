package incidents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIncidentLogger(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "incident-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create logger with custom base directory
	logger := &IncidentLogger{
		baseDir:  tempDir,
		repoName: "test-repo",
	}

	// Test logging an incident
	testContent := "# Test Incident\n\nThis is a test incident."
	err = logger.LogIncident(123, 4, testContent)
	if err != nil {
		t.Fatalf("Failed to log incident: %v", err)
	}

	// Verify directory was created
	incidentDir := filepath.Join(tempDir, "test-repo", "incidents")
	if _, err := os.Stat(incidentDir); os.IsNotExist(err) {
		t.Fatalf("Incident directory was not created")
	}

	// List incidents
	incidents, err := logger.ListIncidents()
	if err != nil {
		t.Fatalf("Failed to list incidents: %v", err)
	}

	if len(incidents) != 1 {
		t.Fatalf("Expected 1 incident, got %d", len(incidents))
	}

	incident := incidents[0]
	if incident.IssueNumber != 123 {
		t.Errorf("Expected issue number 123, got %d", incident.IssueNumber)
	}
	if incident.Attempt != 4 {
		t.Errorf("Expected attempt 4, got %d", incident.Attempt)
	}

	// Get incident content
	content, err := logger.GetIncident(incident.FilePath)
	if err != nil {
		t.Fatalf("Failed to get incident content: %v", err)
	}

	if strings.TrimSpace(content) != strings.TrimSpace(testContent) {
		t.Errorf("Content mismatch.\nExpected: %q\nGot: %q", testContent, content)
	}
}

func TestGenerateFilename(t *testing.T) {
	logger := &IncidentLogger{}
	filename := logger.generateFilename(123, 4)
	
	if !strings.HasPrefix(filename, "incident-123-4-") {
		t.Errorf("Filename should start with 'incident-123-4-', got: %s", filename)
	}
	
	if !strings.HasSuffix(filename, ".md") {
		t.Errorf("Filename should end with '.md', got: %s", filename)
	}
}