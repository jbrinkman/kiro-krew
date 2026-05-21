package watcher

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jbrinkman/kiro-krew/internal/agent"
	"github.com/jbrinkman/kiro-krew/internal/config"
	"github.com/jbrinkman/kiro-krew/internal/github"
)

type Watcher struct {
	config  *config.Config
	manager *agent.Manager
	stop    chan struct{}
	tracked map[int]bool
	mu      sync.RWMutex
}

func New(cfg *config.Config, mgr *agent.Manager) *Watcher {
	return &Watcher{
		config:  cfg,
		manager: mgr,
		stop:    make(chan struct{}),
		tracked: make(map[int]bool),
	}
}

func (w *Watcher) Start() {
	w.cleanupOrphanedWorktrees()
	go w.pollLoop()
}

func (w *Watcher) Stop() {
	close(w.stop)
}

func (w *Watcher) pollLoop() {
	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stop:
			return
		case <-ticker.C:
			w.checkIssues()
		}
	}
}

func (w *Watcher) checkIssues() {
	issues, err := github.ListIssues(w.config.Repo, w.config.Label)
	if err != nil {
		return
	}

	for _, issue := range issues {
		w.mu.RLock()
		tracked := w.tracked[issue.Number]
		w.mu.RUnlock()

		if tracked {
			continue
		}

		if w.hasActiveWorktree(issue.Number) {
			continue
		}

		if w.hasExceededGlobalRetries(issue.Number) {
			log.Printf("Skipping issue #%d: exceeded retry limit (%d attempts)", issue.Number, w.config.MaxRetries)
			continue
		}

		w.mu.Lock()
		w.tracked[issue.Number] = true
		w.mu.Unlock()

		w.incrementGlobalRetryCount(issue.Number)
		log.Printf("Spawning agent for issue #%d (attempt %d)", issue.Number, w.getCurrentRetryCount(issue.Number))
		w.manager.Spawn(issue.Number, w.config.Repo)
	}
}

func (w *Watcher) cleanupOrphanedWorktrees() {
	worktreesDir := ".worktrees"
	entries, err := os.ReadDir(worktreesDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, "issue-") {
			continue
		}

		parts := strings.Split(name, "-")
		if len(parts) < 3 {
			continue
		}

		pidStr := parts[len(parts)-1]
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		if !w.isProcessRunning(pid) {
			worktreePath := filepath.Join(worktreesDir, name)
			log.Printf("Cleaning up orphaned worktree: %s (PID %d no longer running)", name, pid)
			os.RemoveAll(worktreePath)
		}
	}
}

func (w *Watcher) hasActiveWorktree(issueNumber int) bool {
	w.cleanupOrphanedWorktrees()
	
	worktreesDir := ".worktrees"
	entries, err := os.ReadDir(worktreesDir)
	if err != nil {
		return false
	}

	prefix := fmt.Sprintf("issue-%d-", issueNumber)
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) {
			return true
		}
	}
	return false
}

func (w *Watcher) isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func (w *Watcher) hasExceededGlobalRetries(issueNumber int) bool {
	retryFile := fmt.Sprintf(".kiro-krew/retries/issue-%d.count", issueNumber)
	data, err := os.ReadFile(retryFile)
	if err != nil {
		return false
	}
	
	count, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false
	}
	
	return count >= w.config.MaxRetries
}

func (w *Watcher) incrementGlobalRetryCount(issueNumber int) {
	retryDir := ".kiro-krew/retries"
	os.MkdirAll(retryDir, 0755)
	
	retryFile := fmt.Sprintf("%s/issue-%d.count", retryDir, issueNumber)
	
	count := 0
	if data, err := os.ReadFile(retryFile); err == nil {
		if c, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
			count = c
		}
	}
	
	count++
	os.WriteFile(retryFile, []byte(fmt.Sprintf("%d\n", count)), 0644)
}

func (w *Watcher) getCurrentRetryCount(issueNumber int) int {
	retryFile := fmt.Sprintf(".kiro-krew/retries/issue-%d.count", issueNumber)
	data, err := os.ReadFile(retryFile)
	if err != nil {
		return 0
	}
	
	count, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	
	return count
}