package tui

import "os/exec"

// execCommand creates an exec.Cmd for subprocess overlay.
func execCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
