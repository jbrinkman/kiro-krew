//go:build windows

package session

import (
	"fmt"
	"os/exec"
)

func setSysProcAttr(cmd *exec.Cmd) {
	// No process group on Windows
}

func suspendProcess(pid int) error {
	return fmt.Errorf("suspend not supported on Windows")
}

func resumeProcess(pid int) error {
	return fmt.Errorf("resume not supported on Windows")
}

func terminateProcess(pid int) error {
	// On Windows, use taskkill for the process tree
	return exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", pid)).Run()
}
