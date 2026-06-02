package agent

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jbrinkman/kiro-krew/internal/config"
	"github.com/jbrinkman/kiro-krew/internal/github"
)

type Status string

const (
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

type Agent struct {
	ID          string
	IssueNumber int
	IssueTitle  string
	Process     *os.Process
	LogFile     *os.File
	Status      Status
	RetryCount  int
	StartTime   time.Time
}

type Manager struct {
	mu     sync.RWMutex
	agents map[string]*Agent
	config *config.Config
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		agents: make(map[string]*Agent),
		config: cfg,
	}
}

type prefixedWriter struct {
	prefix string
	writer io.Writer
}

func (pw *prefixedWriter) Write(p []byte) (n int, err error) {
	prefixed := fmt.Sprintf("%s%s", pw.prefix, string(p))
	return pw.writer.Write([]byte(prefixed))
}

func (m *Manager) createPrefixedWriter(issueNumber int) io.Writer {
	prefix := fmt.Sprintf("[agent issue-%d] ", issueNumber)
	return &prefixedWriter{
		prefix: prefix,
		writer: os.Stdout,
	}
}

func (m *Manager) Spawn(issueNumber int, repo string) (*Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := fmt.Sprintf("agent-%d-%d", issueNumber, time.Now().Unix())

	worktreeName := fmt.Sprintf("issue-%d-%d", issueNumber, os.Getpid())

	// Create worktree before spawning agent so it runs inside it
	createScript := filepath.Join(".kiro-krew", "scripts", "worktree-create.sh")
	createCmd := exec.Command("bash", createScript, worktreeName)
	wtOutput, err := createCmd.Output()
	if err != nil {
		log.Printf("[agent] failed to create worktree for issue #%d: %v", issueNumber, err)
		return nil, fmt.Errorf("failed to create worktree: %w", err)
	}
	worktreePath := strings.TrimSpace(string(wtOutput))
	log.Printf("[agent] created worktree at %s", worktreePath)

	// Create per-issue log file for agent output
	agentLogDir := filepath.Join(".kiro-krew", "logs")
	os.MkdirAll(agentLogDir, 0755)
	agentLogPath := filepath.Join(agentLogDir, fmt.Sprintf("issue-%d.log", issueNumber))
	agentLogFile, err := os.OpenFile(agentLogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("[agent] failed to create log file for issue #%d: %v", issueNumber, err)
		return nil, fmt.Errorf("failed to create agent log file: %w", err)
	}

	cmd := exec.Command("kiro-cli", "chat",
		"--agent", "krew-lead",
		"--no-interactive",
		"--trust-all-tools",
		fmt.Sprintf("Process issue #%d from repo %s. Worktree name: %s. You are already in the worktree directory — all file operations happen here. Skip worktree creation (step 2).", issueNumber, repo, worktreeName))
	cmd.Dir = worktreePath
	
	// Create conditional writer based on console logging configuration
	var outputWriter io.Writer
	if m.config.ConsoleLogging {
		outputWriter = io.MultiWriter(agentLogFile, m.createPrefixedWriter(issueNumber))
	} else {
		outputWriter = agentLogFile
	}
	cmd.Stdout = outputWriter
	cmd.Stderr = outputWriter
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("ISSUE_NUMBER=%d", issueNumber),
		fmt.Sprintf("REPO=%s", repo),
		fmt.Sprintf("KIRO_KREW_WATCHER_PID=%d", os.Getpid()),
		fmt.Sprintf("WORKTREE_PATH=%s", worktreePath))

	if err := cmd.Start(); err != nil {
		agentLogFile.Close()
		log.Printf("[agent] failed to spawn agent %s for issue #%d: %v", id, issueNumber, err)
		return nil, fmt.Errorf("failed to start agent: %w", err)
	}

	agent := &Agent{
		ID:          id,
		IssueNumber: issueNumber,
		IssueTitle:  fmt.Sprintf("Issue #%d", issueNumber),
		Process:     cmd.Process,
		LogFile:     agentLogFile,
		Status:      StatusRunning,
		RetryCount:  0,
		StartTime:   time.Now(),
	}

	m.agents[id] = agent
	log.Printf("[agent] spawned for issue #%d (worktree: %s, log: %s)", issueNumber, worktreeName, agentLogPath)

	go m.monitorAgent(agent, cmd)

	return agent, nil
}

func (m *Manager) List() []*Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agents := make([]*Agent, 0, len(m.agents))
	for _, agent := range m.agents {
		agents = append(agents, agent)
	}
	return agents
}

func (m *Manager) Stop(id string) error {
	m.mu.RLock()
	agent, exists := m.agents[id]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("agent %s not found", id)
	}

	log.Printf("[agent] stopping %s (issue #%d)", id, agent.IssueNumber)
	if agent.Process != nil {
		return agent.Process.Signal(syscall.SIGTERM)
	}
	return nil
}

func (m *Manager) StopAll() {
	m.mu.RLock()
	agents := make([]*Agent, 0, len(m.agents))
	for _, agent := range m.agents {
		if agent.Status == StatusRunning {
			agents = append(agents, agent)
		}
	}
	m.mu.RUnlock()

	for _, agent := range agents {
		if agent.Process != nil {
			log.Printf("[agent] stopping %s (issue #%d)", agent.ID, agent.IssueNumber)
			agent.Process.Signal(syscall.SIGTERM)
		}
	}
}

func (m *Manager) HandleExit(id string, exitCode int) {
	if exitCode == 0 {
		m.mu.Lock()
		agent, exists := m.agents[id]
		if !exists {
			m.mu.Unlock()
			return
		}
		issueNumber := agent.IssueNumber
		m.mu.Unlock()

		log.Printf("[agent] %s completed with exit code 0 (issue #%d), verifying PR exists", id, issueNumber)

		// Verify PR exists before marking as done
		prExists, err := github.PRExistsForIssue(m.config.Repo, issueNumber)
		if err != nil {
			log.Printf("[agent] failed to check for PR for issue #%d: %v", issueNumber, err)
		}
		if prExists {
			m.mu.Lock()
			updatedAgent, exists := m.agents[id]
			if !exists || updatedAgent.Status != StatusRunning {
				m.mu.Unlock()
				return
			}
			updatedAgent.Status = StatusCompleted
			log.Printf("[agent] %s completed successfully with PR (issue #%d)", id, updatedAgent.IssueNumber)
			doneLabel := m.config.Label + "-done"
			m.mu.Unlock()

			if err := github.AddLabel(m.config.Repo, issueNumber, doneLabel); err != nil {
				log.Printf("[agent] failed to add %s label to issue #%d: %v", doneLabel, issueNumber, err)
			}
		} else {
			m.mu.Lock()
			updatedAgent, exists := m.agents[id]
			if !exists || updatedAgent.Status != StatusRunning {
				m.mu.Unlock()
				return
			}
			// No PR found, treat as failure
			updatedAgent.Status = StatusFailed
			log.Printf("[agent] %s completed but no PR found (issue #%d, retry %d/%d)", id, updatedAgent.IssueNumber, updatedAgent.RetryCount, m.config.MaxRetries)
			if updatedAgent.RetryCount < m.config.MaxRetries {
				updatedAgent.RetryCount++
				log.Printf("[agent] retrying %s (issue #%d, attempt %d)", id, updatedAgent.IssueNumber, updatedAgent.RetryCount)
				go m.retryAgent(updatedAgent)
				m.mu.Unlock()
			} else {
				failedLabel := m.config.Label + "-failed"
				log.Printf("[agent] %s exhausted retries, labeling issue #%d as %s", id, updatedAgent.IssueNumber, failedLabel)
				m.mu.Unlock()
				github.AddLabel(m.config.Repo, issueNumber, failedLabel)
			}
		}
		// Perform cleanup after successful completion
		go m.performCleanup(agent.IssueNumber, os.Getpid())

	} else {
		m.mu.Lock()
		defer m.mu.Unlock()

		agent, exists := m.agents[id]
		if !exists {
			return
		}

		agent.Status = StatusFailed
		log.Printf("[agent] %s exited with code %d (issue #%d, retry %d/%d)", id, exitCode, agent.IssueNumber, agent.RetryCount, m.config.MaxRetries)
		if agent.RetryCount < m.config.MaxRetries {
			agent.RetryCount++
			log.Printf("[agent] retrying %s (issue #%d, attempt %d)", id, agent.IssueNumber, agent.RetryCount)
			go m.retryAgent(agent)
		} else {
			failedLabel := m.config.Label + "-failed"
			log.Printf("[agent] %s exhausted retries, labeling issue #%d as %s", id, agent.IssueNumber, failedLabel)
			github.AddLabel(m.config.Repo, agent.IssueNumber, failedLabel)
		}
	}
}

func (m *Manager) monitorAgent(agent *Agent, cmd *exec.Cmd) {
	log.Printf("[agent] started working on issue #%d", agent.IssueNumber)

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				elapsed := time.Since(agent.StartTime).Truncate(time.Second)
				log.Printf("[agent] still working on issue #%d (%s elapsed)", agent.IssueNumber, elapsed)
			}
		}
	}()

	err := cmd.Wait()
	close(done)

	if agent.LogFile != nil {
		agent.LogFile.Close()
	}

	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
		}
	}

	elapsed := time.Since(agent.StartTime).Truncate(time.Second)
	if exitCode == 0 {
		log.Printf("[agent] finished issue #%d (%s elapsed)", agent.IssueNumber, elapsed)
	} else {
		log.Printf("[agent] failed issue #%d (exit %d, %s elapsed)", agent.IssueNumber, exitCode, elapsed)
	}

	m.HandleExit(agent.ID, exitCode)
}

func (m *Manager) retryAgent(agent *Agent) {
	delay := time.Duration(agent.RetryCount) * time.Second
	log.Printf("[agent] waiting %s before retry for issue #%d", delay, agent.IssueNumber)
	time.Sleep(delay)

	worktreeName := fmt.Sprintf("issue-%d-%d", agent.IssueNumber, os.Getpid())
	worktreePath := filepath.Join(".worktrees", worktreeName)

	// Ensure worktree exists (it should from initial spawn, but recreate if needed)
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		createScript := filepath.Join(".kiro-krew", "scripts", "worktree-create.sh")
		createCmd := exec.Command("bash", createScript, worktreeName)
		if wtOutput, err := createCmd.Output(); err != nil {
			log.Printf("[agent] retry failed to create worktree for issue #%d: %v", agent.IssueNumber, err)
			m.mu.Lock()
			agent.Status = StatusFailed
			m.mu.Unlock()
			return
		} else {
			worktreePath = strings.TrimSpace(string(wtOutput))
		}
	} else {
		// Convert to absolute path
		if abs, err := filepath.Abs(worktreePath); err == nil {
			worktreePath = abs
		}
	}

	// Reopen log file for retry (append mode)
	agentLogPath := filepath.Join(".kiro-krew", "logs", fmt.Sprintf("issue-%d.log", agent.IssueNumber))
	agentLogFile, err := os.OpenFile(agentLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("[agent] retry failed to open log file for issue #%d: %v", agent.IssueNumber, err)
		m.mu.Lock()
		agent.Status = StatusFailed
		m.mu.Unlock()
		return
	}

	cmd := exec.Command("kiro-cli", "chat",
		"--agent", "krew-lead",
		"--no-interactive",
		"--trust-all-tools",
		fmt.Sprintf("Process issue #%d from repo %s. Worktree name: %s. You are already in the worktree directory — all file operations happen here. Skip worktree creation (step 2).", agent.IssueNumber, m.config.Repo, worktreeName))
	cmd.Dir = worktreePath
	
	// Create conditional writer based on console logging configuration
	var outputWriter io.Writer
	if m.config.ConsoleLogging {
		outputWriter = io.MultiWriter(agentLogFile, m.createPrefixedWriter(agent.IssueNumber))
	} else {
		outputWriter = agentLogFile
	}
	cmd.Stdout = outputWriter
	cmd.Stderr = outputWriter
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("ISSUE_NUMBER=%d", agent.IssueNumber),
		fmt.Sprintf("REPO=%s", m.config.Repo),
		fmt.Sprintf("KIRO_KREW_WATCHER_PID=%d", os.Getpid()),
		fmt.Sprintf("WORKTREE_PATH=%s", worktreePath))

	if err := cmd.Start(); err != nil {
		agentLogFile.Close()
		log.Printf("[agent] retry failed for issue #%d: %v", agent.IssueNumber, err)
		m.mu.Lock()
		agent.Status = StatusFailed
		m.mu.Unlock()
		return
	}

	m.mu.Lock()
	agent.Process = cmd.Process
	agent.LogFile = agentLogFile
	agent.Status = StatusRunning
	agent.StartTime = time.Now()
	m.mu.Unlock()

	log.Printf("[agent] retry started for issue #%d (worktree: %s)", agent.IssueNumber, worktreeName)
	go m.monitorAgent(agent, cmd)
}

// cleanupWorktree removes the worktree directory for the given issue and PID
func (m *Manager) cleanupWorktree(issueNumber, pid int) error {
	worktreePath := filepath.Join(".worktrees", fmt.Sprintf("issue-%d-%d", issueNumber, pid))

	removeCmd := exec.Command("git", "worktree", "remove", "--force", worktreePath)
	if output, err := removeCmd.CombinedOutput(); err == nil {
		log.Printf("[cleanup] removed git worktree: %s", worktreePath)
		return nil
	} else {
		log.Printf("[cleanup] git worktree remove failed for %s: %v (output: %s)", worktreePath, err, string(output))
	}

	if err := os.RemoveAll(worktreePath); err != nil {
		return fmt.Errorf("failed to remove worktree %s after git worktree remove failure: %w", worktreePath, err)
	}

	pruneCmd := exec.Command("git", "worktree", "prune")
	if output, err := pruneCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("removed worktree directory %s but failed to prune git worktree metadata: %w (output: %s)", worktreePath, err, string(output))
	}

	log.Printf("[cleanup] removed worktree directory and pruned git metadata: %s", worktreePath)
	return nil
}

// cleanupRetryFile removes the retry count file for the given issue
func (m *Manager) cleanupRetryFile(issueNumber int) error {
	retryPath := filepath.Join(".kiro-krew", "retries", fmt.Sprintf("issue-%d.count", issueNumber))
	if err := os.Remove(retryPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to remove retry file %s: %w", retryPath, err)
	}
	log.Printf("[cleanup] removed retry file: %s", retryPath)
	return nil
}

// performCleanup verifies PR creation and cleans up worktree and retry files
func (m *Manager) performCleanup(issueNumber, pid int) {
	// Verify PR exists with expected branch name
	prExists, err := github.VerifyPRExists(m.config.Repo, issueNumber, pid)
	if err != nil {
		log.Printf("[cleanup] failed to verify PR for issue #%d: %v", issueNumber, err)
		return
	}

	if !prExists {
		log.Printf("[cleanup] no PR found with expected branch name for issue #%d, skipping cleanup", issueNumber)
		return
	}

	log.Printf("[cleanup] PR verified for issue #%d, proceeding with cleanup", issueNumber)

	// Clean up worktree
	if err := m.cleanupWorktree(issueNumber, pid); err != nil {
		log.Printf("[cleanup] worktree cleanup failed for issue #%d: %v", issueNumber, err)
	}

	// Clean up retry file
	if err := m.cleanupRetryFile(issueNumber); err != nil {
		log.Printf("[cleanup] retry file cleanup failed for issue #%d: %v", issueNumber, err)
	}
}
