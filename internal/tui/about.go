package tui

import (
	"fmt"

	"github.com/jbrinkman/kiro-krew/internal/version"
)

// AboutDialog manages about dialog state and content generation
type AboutDialog struct {
	baseContent []string
	statusLines []string
}

// NewAboutDialog creates a new AboutDialog instance
func NewAboutDialog() *AboutDialog {
	return &AboutDialog{}
}

// BuildContent generates base content without update status
func (d *AboutDialog) BuildContent() []string {
	info := version.Info()
	displayHash := formatCommitHash(info["commit_hash"])

	d.baseContent = []string{
		fmt.Sprintf("  Version:    %s", info["version"]),
		fmt.Sprintf("  Build Date: %s", info["build_date"]),
		fmt.Sprintf("  Commit:     %s", displayHash),
		fmt.Sprintf("  Go Version: %s", info["go_version"]),
		fmt.Sprintf("  Arch:       %s", info["arch"]),
		"",
	}

	return d.baseContent
}

// UpdateStatusLine updates the status lines for partial content updates
func (d *AboutDialog) UpdateStatusLine(statusLines []string) {
	d.statusLines = statusLines
}

// GetFullContent returns complete content with base info and status lines
func (d *AboutDialog) GetFullContent() []string {
	content := make([]string, len(d.baseContent))
	copy(content, d.baseContent)
	content = append(content, d.statusLines...)
	return content
}

// formatCommitHash returns a short display hash (7 chars) or "unknown"
func formatCommitHash(hash string) string {
	if hash == "unknown" || hash == "" {
		return "unknown"
	}
	if len(hash) >= 7 {
		return hash[:7]
	}
	return hash
}
