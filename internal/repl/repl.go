package repl

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Agent struct {
	IssueNumber int
	Title       string
	Status      string
	StartTime   time.Time
}

type REPL struct {
	agents  map[int]*Agent
	watcher bool
}

func New() *REPL {
	return &REPL{
		agents: make(map[int]*Agent),
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
		r.watcher = true
		fmt.Println("Watcher started")
	case "stop":
		r.watcher = false
		fmt.Println("Watcher stopped")
	default:
		fmt.Println("Usage: watch start|stop")
	}
}

func (r *REPL) handleStatus() {
	if len(r.agents) == 0 {
		fmt.Println("No agents running")
		return
	}
	
	fmt.Printf("%-8s %-30s %-10s %s\n", "Issue", "Title", "Status", "Elapsed")
	fmt.Println(strings.Repeat("-", 70))
	
	for _, agent := range r.agents {
		elapsed := time.Since(agent.StartTime).Truncate(time.Second)
		fmt.Printf("%-8d %-30s %-10s %s\n", 
			agent.IssueNumber, 
			truncate(agent.Title, 30), 
			agent.Status, 
			elapsed)
	}
}

func (r *REPL) handleStop(issueStr string) {
	issueNum, err := strconv.Atoi(issueStr)
	if err != nil {
		fmt.Printf("Invalid issue number: %s\n", issueStr)
		return
	}
	
	if _, exists := r.agents[issueNum]; !exists {
		fmt.Printf("No agent running for issue %d\n", issueNum)
		return
	}
	
	delete(r.agents, issueNum)
	fmt.Printf("Stopped agent for issue %d\n", issueNum)
}

func (r *REPL) handleExit() bool {
	if len(r.agents) > 0 {
		fmt.Printf("There are %d agents still running. Stop all and exit? (y/N): ", len(r.agents))
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			response := strings.ToLower(strings.TrimSpace(scanner.Text()))
			if response != "y" && response != "yes" {
				return false
			}
		}
		r.agents = make(map[int]*Agent)
	}
	fmt.Println("Goodbye!")
	return true
}

func (r *REPL) handleHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  watch start    - Start the watcher")
	fmt.Println("  watch stop     - Stop the watcher")
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

func (r *REPL) AddAgent(issueNumber int, title, status string) {
	r.agents[issueNumber] = &Agent{
		IssueNumber: issueNumber,
		Title:       title,
		Status:      status,
		StartTime:   time.Now(),
	}
}