package tui

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/jbrinkman/kiro-krew/internal/acp"
)

// PlanningState represents the state of a planning session
type PlanningState int

const (
	StateActive PlanningState = iota
	StateCompleted
	StateFailed
	StateReadOnly
)

// ChatMessage represents a single message in the chat
type ChatMessage struct {
	Role      string // "user" or "assistant" or "system"
	Content   string
	Timestamp time.Time
	IsJSON    bool // For structured responses
}

// ContextInfo holds context information for the status row
type ContextInfo struct {
	ContextUsage string // "45k/200k"
	Model        string // "claude-sonnet-4"
	Directory    string // Current working directory
}

// PlanningTab implements the Tab interface for ACP-based planning
type PlanningTab struct {
	id          string
	acpClient   *acp.Client
	acpSession  *acp.Session
	messages    []ChatMessage
	viewport    viewport.Model
	input       textinput.Model
	width       int
	height      int
	state       PlanningState
	contextInfo *ContextInfo
	styles      *Styles
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewPlanningTab creates a new ACP-based planning tab
func NewPlanningTab(styles *Styles) (*PlanningTab, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize ACP client
	client, err := acp.NewClient(acp.Config{
		Context: ctx,
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create ACP client: %w", err)
	}

	// Start chat session
	session, err := client.StartChat(acp.ChatConfig{
		Agent: "planner",
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start chat session: %w", err)
	}

	// Initialize UI components
	input := textinput.New()
	input.Focus()
	input.Prompt = "[planner] > "

	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(24))
	vp.MouseWheelEnabled = true

	// Get current directory for context info
	dir, _ := os.Getwd()

	tab := &PlanningTab{
		id:         generatePlanningTabID(),
		acpClient:  client,
		acpSession: session,
		messages:   make([]ChatMessage, 0),
		viewport:   vp,
		input:      input,
		state:      StateActive,
		contextInfo: &ContextInfo{
			Directory: dir,
		},
		styles: styles,
		ctx:    ctx,
		cancel: cancel,
	}

	// Start message streaming
	go tab.streamMessages()

	// Add welcome message
	tab.appendMessage("system", "Planning session started. Type your question or requirement below.", false)

	return tab, nil
}

// generatePlanningTabID generates a unique ID for a planning tab
func generatePlanningTabID() string {
	return fmt.Sprintf("planning-%d-%d", time.Now().Unix(), rand.Intn(1000))
}

// streamMessages handles ACP message streaming in background
func (pt *PlanningTab) streamMessages() {
	for {
		select {
		case <-pt.ctx.Done():
			return
		case response, ok := <-pt.acpSession.Stream():
			if !ok {
				return
			}

			switch response.Type {
			case acp.ResponseTypeText:
				pt.appendMessage("assistant", response.Content, false)
			case acp.ResponseTypeJSON:
				if response.Complete {
					pt.appendMessage("assistant", response.Content, true)
				}
			case acp.ResponseTypeContext:
				pt.updateContextInfo(response.Context)
			case acp.ResponseTypeError:
				pt.appendMessage("system", fmt.Sprintf("Error: %v", response.Error), false)
			}
		}
	}
}

// appendMessage adds a new message to the chat
func (pt *PlanningTab) appendMessage(role, content string, isJSON bool) {
	message := ChatMessage{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		IsJSON:    isJSON,
	}
	pt.messages = append(pt.messages, message)

	// Update viewport content
	pt.refreshViewport()
}

// updateContextInfo updates the context information
func (pt *PlanningTab) updateContextInfo(context acp.Context) {
	if pt.contextInfo == nil {
		pt.contextInfo = &ContextInfo{}
	}

	if context.ContextUsage != "" {
		pt.contextInfo.ContextUsage = context.ContextUsage
	}
	if context.Model != "" {
		pt.contextInfo.Model = context.Model
	}
}

// refreshViewport updates the viewport with current messages
func (pt *PlanningTab) refreshViewport() {
	var content strings.Builder

	for i, msg := range pt.messages {
		if i > 0 {
			content.WriteString("\n\n")
		}

		switch msg.Role {
		case "user":
			content.WriteString(pt.styles.Prompt.Render(fmt.Sprintf("[planner] > %s", msg.Content)))
		case "assistant":
			if msg.IsJSON {
				content.WriteString(pt.styles.Success.Render(msg.Content)) // Use Success style for JSON
			} else {
				content.WriteString(msg.Content)
			}
		case "system":
			content.WriteString(pt.styles.Warning.Render(msg.Content))
		}
	}

	pt.viewport.SetContent(content.String())
	pt.viewport.GotoBottom()
}

// SendMessage sends a message through the ACP session
func (pt *PlanningTab) SendMessage(content string) error {
	if pt.IsReadOnly() {
		return fmt.Errorf("planning session is read-only")
	}

	// Add user message to chat
	pt.appendMessage("user", content, false)

	// Send through ACP
	err := pt.acpSession.SendMessage(content)
	if err != nil {
		if acp.IsRetryableError(err) {
			pt.appendMessage("system", "Connection interrupted, retrying...", false)
		} else {
			pt.appendMessage("system", fmt.Sprintf("Failed to send message: %v", err), false)
			return fmt.Errorf("failed to send message: %w", err)
		}
	}

	return nil
}

// IsReadOnly returns whether the tab is in read-only state
func (pt *PlanningTab) IsReadOnly() bool {
	return pt.state == StateCompleted || pt.state == StateFailed
}

// handleIssueCreated handles when a GitHub issue is created
func (pt *PlanningTab) handleIssueCreated(issueURL string) {
	pt.state = StateCompleted
	pt.input.Blur() // Make read-only
	pt.appendMessage("system", fmt.Sprintf("✅ GitHub issue created: %s", issueURL), false)
}

// handleError handles planning session errors
func (pt *PlanningTab) handleError(err error) {
	pt.state = StateFailed
	pt.input.Blur() // Make read-only
	pt.appendMessage("system", fmt.Sprintf("❌ Planning failed: %v", err), false)
}

// Tab interface implementation

// ID returns the tab's unique identifier
func (pt *PlanningTab) ID() string {
	return pt.id
}

// Type returns the tab type
func (pt *PlanningTab) Type() TabType {
	return TabTypePlanning
}

// Title returns the tab title
func (pt *PlanningTab) Title() string {
	return "Planning"
}

// IsClosable returns whether the tab can be closed
func (pt *PlanningTab) IsClosable() bool {
	return true
}

// View renders the tab content
func (pt *PlanningTab) View() string {
	if pt.width == 0 || pt.height == 0 {
		return "Initializing..."
	}

	// Calculate heights
	inputHeight := 1
	availableHeight := pt.height - inputHeight - 1 // Reserve space for input and padding

	// Update viewport size using SetWidth and SetHeight
	pt.viewport.SetWidth(pt.width)
	pt.viewport.SetHeight(availableHeight)

	// Render viewport
	viewportView := pt.viewport.View()

	// Render input area
	inputView := pt.input.View()

	// Combine views
	return viewportView + "\n" + inputView
}

// Update handles bubble tea messages
func (pt *PlanningTab) Update(msg tea.Msg) (Tab, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if pt.IsReadOnly() && msg.String() != "esc" {
			// Ignore all input except escape (to close tab) when read-only
			return pt, nil
		}

		switch msg.String() {
		case "enter":
			if pt.input.Value() != "" {
				content := pt.input.Value()
				pt.input.SetValue("")

				// Send message asynchronously
				go func() {
					if err := pt.SendMessage(content); err != nil {
						// Error is already logged in SendMessage
					}
				}()
			}
		default:
			pt.input, cmd = pt.input.Update(msg)
		}

	case tea.MouseMsg:
		// Handle mouse events in viewport for scrolling
		pt.viewport, _ = pt.viewport.Update(msg)

	default:
		// Update viewport for other messages (like resize)
		pt.viewport, _ = pt.viewport.Update(msg)
		pt.input, cmd = pt.input.Update(msg)
	}

	return pt, cmd
}

// Resize resizes the tab
func (pt *PlanningTab) Resize(width, height int) {
	pt.width = width
	pt.height = height

	// Viewport will be resized in View() method
}

// Close closes the planning tab and cleans up resources
func (pt *PlanningTab) Close() error {
	if pt.cancel != nil {
		pt.cancel()
	}

	if pt.acpSession != nil {
		if err := pt.acpSession.Close(); err != nil {
			return fmt.Errorf("failed to close ACP session: %w", err)
		}
	}

	if pt.acpClient != nil {
		if err := pt.acpClient.Close(); err != nil {
			return fmt.Errorf("failed to close ACP client: %w", err)
		}
	}

	return nil
}
