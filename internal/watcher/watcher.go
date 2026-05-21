package watcher

import (
	"sync"
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

		if !tracked {
			w.mu.Lock()
			w.tracked[issue.Number] = true
			w.mu.Unlock()

			w.manager.Spawn(issue.Number, w.config.Repo)
		}
	}
}