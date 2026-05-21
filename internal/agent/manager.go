package agent

import (
	"fmt"
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
	mu      sync.RWMutex
	agents  map[string]*Agent
	config  *config.Config
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
	
	cmd := exec.Command("kiro-cli", "chat", "--headless")
	cmd.Env = append(os.Environ(), fmt.Sprintf("ISSUE_NUMBER=%d", issueNumber), fmt.Sprintf("REPO=%s", repo))
	
	if err := cmd.Start(); err != nil {
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
			agent.Process.Signal(syscall.SIGTERM)
		}
	}
}

func (m *Manager) HandleExit(id string, exitCode int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, exists := m.agents[id]
	if !exists {
		return
	}

	if exitCode == 0 {
		agent.Status = StatusCompleted
	} else {
		agent.Status = StatusFailed
		if agent.RetryCount < m.config.MaxRetries {
			agent.RetryCount++
			go m.retryAgent(agent)
		} else {
			github.AddLabel(m.config.Repo, agent.IssueNumber, "kiro-krew-failed")
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
	time.Sleep(time.Duration(agent.RetryCount) * time.Second)
	
	cmd := exec.Command("kiro-cli", "chat", "--headless")
	cmd.Env = append(os.Environ(), fmt.Sprintf("ISSUE_NUMBER=%d", agent.IssueNumber))
	
	if err := cmd.Start(); err != nil {
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

	go m.monitorAgent(agent, cmd)
}