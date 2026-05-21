package repl

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jbrinkman/kiro-krew/internal/agent"
	"github.com/jbrinkman/kiro-krew/internal/watcher"
)

type REPL struct {
	watcher        *watcher.Watcher
	manager        *agent.Manager
	watcherRunning bool
}

func New(w *watcher.Watcher, m *agent.Manager) *REPL {
	return &REPL{
		watcher: w,
		manager: m,
	}
}

func (r *REPL) Run() {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("kiro-krew> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		cmd := parts[0]

		switch cmd {
		case "watch":
			if len(parts) < 2 {
				fmt.Println("Usage: watch start|stop")
				continue
			}
			r.handleWatch(parts[1])
		case "status":
			r.handleStatus()
		case "stop":
			if len(parts) < 2 {
				fmt.Println("Usage: stop <issue-number>")
				continue
			}
			r.handleStop(parts[1])
		case "exit":
			if r.handleExit() {
				return
			}
		case "help":
			r.handleHelp()
		default:
			fmt.Printf("Unknown command: %s\n", cmd)
		}
	}
}

func (r *REPL) handleWatch(action string) {
	switch action {
	case "start":
		if r.watcherRunning {
			fmt.Println("Watcher already running")
			return
		}
		r.watcher.Start()
		r.watcherRunning = true
	case "stop":
		if !r.watcherRunning {
			fmt.Println("Watcher not running")
			return
		}
		r.watcher.Stop()
		r.watcherRunning = false
	default:
		fmt.Println("Usage: watch start|stop")
	}
}

func (r *REPL) handleStatus() {
	agents := r.manager.List()
	if len(agents) == 0 {
		fmt.Println("No agents running")
		return
	}

	fmt.Printf("%-8s %-30s %-10s %s\n", "Issue", "Title", "Status", "Elapsed")
	fmt.Println(strings.Repeat("-", 70))

	for _, a := range agents {
		elapsed := time.Since(a.StartTime).Truncate(time.Second)
		fmt.Printf("%-8d %-30s %-10s %s\n",
			a.IssueNumber,
			truncate(a.IssueTitle, 30),
			string(a.Status),
			elapsed)
	}
}

func (r *REPL) handleStop(issueStr string) {
	issueNum, err := strconv.Atoi(issueStr)
	if err != nil {
		fmt.Printf("Invalid issue number: %s\n", issueStr)
		return
	}

	agents := r.manager.List()
	for _, a := range agents {
		if a.IssueNumber == issueNum {
			if err := r.manager.Stop(a.ID); err != nil {
				fmt.Printf("Error stopping agent: %v\n", err)
			} else {
				fmt.Printf("Stopped agent for issue %d\n", issueNum)
			}
			return
		}
	}
	fmt.Printf("No agent running for issue %d\n", issueNum)
}

func (r *REPL) handleExit() bool {
	agents := r.manager.List()
	running := 0
	for _, a := range agents {
		if a.Status == agent.StatusRunning {
			running++
		}
	}

	if running > 0 {
		fmt.Printf("There are %d agents still running. Stop all and exit? (y/N): ", running)
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			response := strings.ToLower(strings.TrimSpace(scanner.Text()))
			if response != "y" && response != "yes" {
				return false
			}
		}
		r.manager.StopAll()
	}

	if r.watcherRunning {
		r.watcher.Stop()
	}

	fmt.Println("Goodbye!")
	return true
}

func (r *REPL) handleHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  watch start    - Start watching for labeled issues")
	fmt.Println("  watch stop     - Stop watching")
	fmt.Println("  status         - List all agents with details")
	fmt.Println("  stop <issue>   - Stop agent for specific issue number")
	fmt.Println("  exit           - Exit the REPL")
	fmt.Println("  help           - Show this help message")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
