package agent

import (
	"fmt"
	"log"
	"os"
	"os/exec"
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

func (m *Manager) Spawn(issueNumber int, repo string) (*Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := fmt.Sprintf("agent-%d-%d", issueNumber, time.Now().Unix())

	worktreeName := fmt.Sprintf("issue-%d-%d", issueNumber, os.Getpid())

	cmd := exec.Command("kiro-cli", "chat",
		"--agent", "krew-lead",
		"--no-interactive",
		"--trust-all-tools",
		fmt.Sprintf("Process issue #%d from repo %s. Worktree name: %s", issueNumber, repo, worktreeName))
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("ISSUE_NUMBER=%d", issueNumber),
		fmt.Sprintf("REPO=%s", repo),
		fmt.Sprintf("KIRO_KREW_WATCHER_PID=%d", os.Getpid()))

	if err := cmd.Start(); err != nil {
		log.Printf("[agent] failed to spawn agent %s for issue #%d: %v", id, issueNumber, err)
		return nil, fmt.Errorf("failed to start agent: %w", err)
	}

	agent := &Agent{
		ID:          id,
		IssueNumber: issueNumber,
		IssueTitle:  fmt.Sprintf("Issue #%d", issueNumber),
		Process:     cmd.Process,
		Status:      StatusRunning,
		RetryCount:  0,
		StartTime:   time.Now(),
	}

	m.agents[id] = agent
	log.Printf("[agent] started %s (pid %d) for issue #%d", id, cmd.Process.Pid, issueNumber)

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

// verifyPRExists checks if a PR exists for the given issue number
func (m *Manager) verifyPRExists(issueNumber int) bool {
	// Check for PRs with branch pattern spec/issue-<number>-*
	cmd := exec.Command("gh", "pr", "list", "--repo", m.config.Repo, "--search", fmt.Sprintf("head:spec/issue-%d-", issueNumber), "--json", "number")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("[agent] failed to check for PR for issue #%d: %v", issueNumber, err)
		return false
	}
	// If output is not empty JSON array, PR exists
	return len(output) > 3 // More than just "[]"
}

func (m *Manager) HandleExit(id string, exitCode int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, exists := m.agents[id]
	if !exists {
		return
	}

	if exitCode == 0 {
		log.Printf("[agent] %s completed with exit code 0 (issue #%d), verifying PR exists", id, agent.IssueNumber)
		
		// Verify PR exists before marking as done
		if m.verifyPRExists(agent.IssueNumber) {
			agent.Status = StatusCompleted
			log.Printf("[agent] %s completed successfully with PR (issue #%d)", id, agent.IssueNumber)
			doneLabel := m.config.Label + "-done"
			if err := github.AddLabel(m.config.Repo, agent.IssueNumber, doneLabel); err != nil {
				log.Printf("[agent] failed to add %s label to issue #%d: %v", doneLabel, agent.IssueNumber, err)
			}
		} else {
			// No PR found, treat as failure
			agent.Status = StatusFailed
			log.Printf("[agent] %s completed but no PR found (issue #%d, retry %d/%d)", id, agent.IssueNumber, agent.RetryCount, m.config.MaxRetries)
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
	} else {
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
	err := cmd.Wait()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
		}
	}
	m.HandleExit(agent.ID, exitCode)
}

func (m *Manager) retryAgent(agent *Agent) {
	delay := time.Duration(agent.RetryCount) * time.Second
	log.Printf("[agent] waiting %s before retry for issue #%d", delay, agent.IssueNumber)
	time.Sleep(delay)

	worktreeName := fmt.Sprintf("issue-%d-%d", agent.IssueNumber, os.Getpid())

	cmd := exec.Command("kiro-cli", "chat",
		"--agent", "krew-lead",
		"--no-interactive",
		"--trust-all-tools",
		fmt.Sprintf("Process issue #%d from repo %s. Worktree name: %s", agent.IssueNumber, m.config.Repo, worktreeName))
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("ISSUE_NUMBER=%d", agent.IssueNumber),
		fmt.Sprintf("REPO=%s", m.config.Repo),
		fmt.Sprintf("KIRO_KREW_WATCHER_PID=%d", os.Getpid()))

	if err := cmd.Start(); err != nil {
		log.Printf("[agent] retry failed for issue #%d: %v", agent.IssueNumber, err)
		m.mu.Lock()
		agent.Status = StatusFailed
		m.mu.Unlock()
		return
	}

	m.mu.Lock()
	agent.Process = cmd.Process
	agent.Status = StatusRunning
	agent.StartTime = time.Now()
	m.mu.Unlock()

	log.Printf("[agent] retry started for issue #%d (pid %d)", agent.IssueNumber, cmd.Process.Pid)
	go m.monitorAgent(agent, cmd)
}
