package incidents

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func (il *IncidentLogger) ensureIncidentDir() error {
	incidentDir := filepath.Join(il.baseDir, il.repoName, "incidents")
	return os.MkdirAll(incidentDir, 0755)
}

func (il *IncidentLogger) generateFilename(issue, attempt int) string {
	timestamp := time.Now().UTC().Format("20060102-150405")
	return fmt.Sprintf("incident-%d-%d-%s.md", issue, attempt, timestamp)
}

func (il *IncidentLogger) writeIncidentFile(filename, content string) error {
	incidentDir := filepath.Join(il.baseDir, il.repoName, "incidents")
	filePath := filepath.Join(incidentDir, filename)
	
	return os.WriteFile(filePath, []byte(content), 0644)
}
