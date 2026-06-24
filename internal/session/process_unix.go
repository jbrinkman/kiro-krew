//go:build !windows

package session

import (
	"os/exec"
	"syscall"
)

func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func suspendProcess(pid int) error {
	return syscall.Kill(-pid, syscall.SIGSTOP)
}

func resumeProcess(pid int) error {
	return syscall.Kill(-pid, syscall.SIGCONT)
}

func terminateProcess(pid int) error {
	return syscall.Kill(-pid, syscall.SIGTERM)
}
