package session

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// PlannerProcess manages a kiro-cli subprocess for planning sessions
type PlannerProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
	output chan string
	ctx    context.Context
	cancel context.CancelFunc
}

// NewPlannerProcess starts a new kiro-cli subprocess for planning
func NewPlannerProcess() (*PlannerProcess, error) {
	ctx, cancel := context.WithCancel(context.Background())

	cmd := exec.CommandContext(ctx, "kiro-cli", "chat")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Set process group for clean termination
	setSysProcAttr(cmd)

	process := &PlannerProcess{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
		output: make(chan string, 100),
		ctx:    ctx,
		cancel: cancel,
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start kiro-cli: %w", err)
	}

	go process.captureOutput()

	return process, nil
}

// SendMessage sends input to the planning subprocess with error recovery
func (p *PlannerProcess) SendMessage(message string) error {
	select {
	case <-p.ctx.Done():
		return fmt.Errorf("process stopped")
	default:
	}

	// Validate process is still running
	if p.cmd.Process == nil {
		return fmt.Errorf("subprocess terminated unexpectedly")
	}

	_, err := p.stdin.Write([]byte(message + "\n"))
	if err != nil {
		// Check if process died
		if !p.IsRunning() {
			return fmt.Errorf("subprocess died: %w", err)
		}
		return fmt.Errorf("failed to write to subprocess: %w", err)
	}

	return nil
}

// GetOutput returns a channel for reading subprocess output
func (p *PlannerProcess) GetOutput() <-chan string {
	return p.output
}

// Suspend suspends the subprocess by sending SIGSTOP
func (p *PlannerProcess) Suspend() error {
	if p.cmd.Process == nil {
		return fmt.Errorf("process not running")
	}

	return suspendProcess(p.cmd.Process.Pid)
}

// Resume resumes the suspended subprocess by sending SIGCONT
func (p *PlannerProcess) Resume() error {
	if p.cmd.Process == nil {
		return fmt.Errorf("process not running")
	}

	return resumeProcess(p.cmd.Process.Pid)
}

// Stop terminates the subprocess gracefully
func (p *PlannerProcess) Stop() error {
	p.cancel()

	if p.cmd.Process != nil {
		// Send SIGTERM to the process group
		err := terminateProcess(p.cmd.Process.Pid)
		if err != nil {
			// Force kill if graceful termination fails
			_ = p.cmd.Process.Kill()
		}
	}

	// Close pipes
	_ = p.stdin.Close()
	_ = p.stdout.Close()
	_ = p.stderr.Close()

	// Wait for process to finish
	_ = p.cmd.Wait()

	// p.output is closed by captureOutput once stdout/stderr readers have exited.
	return nil
}

// IsRunning returns true if the subprocess is still running
func (p *PlannerProcess) IsRunning() bool {
	if p.cmd.Process == nil {
		return false
	}

	select {
	case <-p.ctx.Done():
		return false
	default:
		return true
	}
}

// captureOutput reads from stdout/stderr and forwards to output channel
func (p *PlannerProcess) captureOutput() {
	var wg sync.WaitGroup
	wg.Add(2)

	forwardOutput := func(reader io.Reader, prefix string) {
		defer wg.Done()

		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()
			if prefix != "" {
				line = prefix + line
			}

			select {
			case p.output <- line:
			case <-p.ctx.Done():
				return
			}
		}
	}

	go forwardOutput(p.stdout, "")
	go forwardOutput(p.stderr, "[ERROR] ")

	wg.Wait()
	close(p.output)
}

// PlanningSession manages a complete planning session with subprocess and history
type PlanningSession struct {
	ID        string
	Process   *PlannerProcess
	State     *SessionState
	Manager   *SessionManager
	Suspended bool
}

// NewPlanningSession creates a new planning session
func NewPlanningSession(manager *SessionManager) (*PlanningSession, error) {
	id, err := manager.Create(Planning)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	state, err := manager.Load(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	process, err := NewPlannerProcess()
	if err != nil {
		return nil, fmt.Errorf("failed to start planner process: %w", err)
	}

	session := &PlanningSession{
		ID:      id,
		Process: process,
		State:   state,
		Manager: manager,
	}

	// Start capturing conversation history
	go session.captureHistory()

	return session, nil
}

// SendMessage sends a message and captures it in history with error handling
func (ps *PlanningSession) SendMessage(content string) error {
	// Validate session state
	if ps.State == nil {
		return fmt.Errorf("session state corrupted")
	}

	ps.State.AddMessage("user", content)
	if err := ps.Manager.Save(ps.ID, ps.State); err != nil {
		return fmt.Errorf("failed to save session state: %w", err)
	}

	err := ps.Process.SendMessage(content)
	if err != nil {
		// Log the error but don't fail the session
		ps.State.AddMessage("system", fmt.Sprintf("Error sending message: %v", err))
		_ = ps.Manager.Save(ps.ID, ps.State)
		return err
	}

	return nil
}

// Suspend suspends the planning session with error recovery
func (ps *PlanningSession) Suspend() error {
	if ps.State == nil {
		return fmt.Errorf("session state corrupted, cannot suspend")
	}

	if err := ps.Manager.Save(ps.ID, ps.State); err != nil {
		return fmt.Errorf("failed to save session state before suspend: %w", err)
	}

	ps.Suspended = true
	if err := ps.Process.Suspend(); err != nil {
		ps.Suspended = false
		return fmt.Errorf("failed to suspend process: %w", err)
	}

	return nil
}

// SuspendAndDetach suspends the planning session and detaches from process with error handling
func (ps *PlanningSession) SuspendAndDetach() error {
	if ps.State == nil {
		return fmt.Errorf("session state corrupted, cannot suspend")
	}

	if err := ps.Manager.Save(ps.ID, ps.State); err != nil {
		return fmt.Errorf("failed to save session state before suspend: %w", err)
	}

	ps.Suspended = true
	if err := ps.Process.Suspend(); err != nil {
		ps.Suspended = false
		return fmt.Errorf("failed to suspend process: %w", err)
	}
	return nil
}

// Resume resumes the planning session with validation
func (ps *PlanningSession) Resume() error {
	if !ps.Suspended {
		return fmt.Errorf("session not suspended")
	}

	if err := ps.Process.Resume(); err != nil {
		return fmt.Errorf("failed to resume process: %w", err)
	}

	ps.Suspended = false
	return nil
}

// Stop terminates the planning session with graceful cleanup
func (ps *PlanningSession) Stop() error {
	if ps.State != nil {
		ps.State.AddMessage("system", "Session terminated")
		_ = ps.Manager.Save(ps.ID, ps.State)
	}

	if ps.Process != nil {
		if err := ps.Process.Stop(); err != nil {
			return fmt.Errorf("failed to stop process: %w", err)
		}
	}

	return nil
}

// Recover attempts to recover from a corrupted planning session
func (ps *PlanningSession) Recover() error {
	// Try to reload session state
	state, err := ps.Manager.Load(ps.ID)
	if err != nil {
		// Create new state if recovery failed
		state = NewSessionState(Planning)
		state.AddMessage("system", "Session recovered from corruption")
	}

	ps.State = state
	return ps.Manager.Save(ps.ID, ps.State)
}

// captureHistory captures subprocess output and adds to conversation history
func (ps *PlanningSession) captureHistory() {
	for output := range ps.Process.GetOutput() {
		ps.State.AddMessage("assistant", output)
		// Periodically save state
		_ = ps.Manager.Save(ps.ID, ps.State)
	}
}
