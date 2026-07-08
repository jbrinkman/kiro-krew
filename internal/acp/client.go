package acp

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/coder/acp-go-sdk"
)

// KiroACPClient implements the Client interface using the official ACP SDK
type KiroACPClient struct {
	config    *ConnectionConfig
	conn      *acp.ClientSideConnection
	cmd       *exec.Cmd
	connected bool
	mu        sync.RWMutex
	client    acp.Client
}

// KiroClient implements the acp.Client interface for permission handling
type KiroClient struct {
	autoApprove bool
}

// RequestPermission handles permission requests from agents
func (k *KiroClient) RequestPermission(ctx context.Context, params acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	if k.autoApprove {
		// Auto-approve - choose first allow option or first available option
		for _, opt := range params.Options {
			if opt.Kind == "allow" {
				return acp.RequestPermissionResponse{
					Outcome: acp.RequestPermissionOutcome{
						Selected: &acp.RequestPermissionOutcomeSelected{
							OptionId: opt.OptionId,
							Outcome:  "selected",
						},
					},
				}, nil
			}
		}
		if len(params.Options) > 0 {
			return acp.RequestPermissionResponse{
				Outcome: acp.RequestPermissionOutcome{
					Selected: &acp.RequestPermissionOutcomeSelected{
						OptionId: params.Options[0].OptionId,
						Outcome:  "selected",
					},
				},
			}, nil
		}
	}

	// For now, auto-approve for planning mode - choose first option if available
	if len(params.Options) > 0 {
		return acp.RequestPermissionResponse{
			Outcome: acp.RequestPermissionOutcome{
				Selected: &acp.RequestPermissionOutcomeSelected{
					OptionId: params.Options[0].OptionId,
					Outcome:  "selected",
				},
			},
		}, nil
	}

	return acp.RequestPermissionResponse{}, fmt.Errorf("no permission options available")
}

// ReadTextFile handles file read requests
func (k *KiroClient) ReadTextFile(ctx context.Context, params acp.ReadTextFileRequest) (acp.ReadTextFileResponse, error) {
	// For now, return an error indicating this capability is not implemented
	// In a full implementation, this would read the requested file
	return acp.ReadTextFileResponse{}, fmt.Errorf("ReadTextFile not implemented in planning mode")
}

// WriteTextFile handles file write requests
func (k *KiroClient) WriteTextFile(ctx context.Context, params acp.WriteTextFileRequest) (acp.WriteTextFileResponse, error) {
	// For now, return an error indicating this capability is not implemented
	// In a full implementation, this would write to the requested file
	return acp.WriteTextFileResponse{}, fmt.Errorf("WriteTextFile not implemented in planning mode")
}

// SessionUpdate handles session update notifications
func (k *KiroClient) SessionUpdate(ctx context.Context, params acp.SessionNotification) error {
	// Log session updates for debugging
	// In a full implementation, this would update the UI with progress
	u := params.Update
	switch {
	case u.AgentMessageChunk != nil:
		content := u.AgentMessageChunk.Content
		if content.Text != nil {
			// This is where we'd capture agent responses
			// For now, just log for debugging
		}
	case u.ToolCall != nil:
		// Tool call started
	case u.ToolCallUpdate != nil:
		// Tool call updated
	case u.Plan != nil:
		// Plan update received
	case u.AgentThoughtChunk != nil:
		// Agent thought process
	}
	return nil
}

// CreateTerminal handles terminal creation requests
func (k *KiroClient) CreateTerminal(ctx context.Context, params acp.CreateTerminalRequest) (acp.CreateTerminalResponse, error) {
	// For planning mode, we don't support terminal operations
	return acp.CreateTerminalResponse{}, fmt.Errorf("terminal operations not supported in planning mode")
}

// KillTerminal handles terminal kill requests
func (k *KiroClient) KillTerminal(ctx context.Context, params acp.KillTerminalRequest) (acp.KillTerminalResponse, error) {
	return acp.KillTerminalResponse{}, fmt.Errorf("terminal operations not supported in planning mode")
}

// TerminalOutput handles terminal output requests
func (k *KiroClient) TerminalOutput(ctx context.Context, params acp.TerminalOutputRequest) (acp.TerminalOutputResponse, error) {
	return acp.TerminalOutputResponse{}, fmt.Errorf("terminal operations not supported in planning mode")
}

// ReleaseTerminal handles terminal release requests
func (k *KiroClient) ReleaseTerminal(ctx context.Context, params acp.ReleaseTerminalRequest) (acp.ReleaseTerminalResponse, error) {
	return acp.ReleaseTerminalResponse{}, fmt.Errorf("terminal operations not supported in planning mode")
}

// WaitForTerminalExit handles terminal exit wait requests
func (k *KiroClient) WaitForTerminalExit(ctx context.Context, params acp.WaitForTerminalExitRequest) (acp.WaitForTerminalExitResponse, error) {
	return acp.WaitForTerminalExitResponse{}, fmt.Errorf("terminal operations not supported in planning mode")
}

// NewClient creates a new ACP client instance
func NewClient(config *ConnectionConfig) *KiroACPClient {
	if config == nil {
		config = DefaultConnectionConfig()
	}

	return &KiroACPClient{
		config:    config,
		connected: false,
		client:    &KiroClient{autoApprove: true},
	}
}

// Connect establishes a connection to the ACP server via Kiro CLI
func (c *KiroACPClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return ErrAlreadyConnected
	}

	// Start kiro-cli in ACP mode
	cmd := exec.CommandContext(ctx, c.config.KiroCLIPath, "acp")

	// Get pipes for stdin/stdout communication
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		return fmt.Errorf("failed to start kiro-cli: %w", err)
	}

	// Create ACP connection
	conn := acp.NewClientSideConnection(c.client, stdin, stdout)

	// Initialize the connection
	initResp, err := conn.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Fs: acp.FileSystemCapabilities{
				ReadTextFile:  true,
				WriteTextFile: true,
			},
		},
	})
	if err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("failed to initialize ACP connection: %w", err)
	}

	c.conn = conn
	c.cmd = cmd
	c.connected = true

	// Log successful connection
	fmt.Printf("✅ Connected to Kiro CLI via ACP (protocol v%v)\n", initResp.ProtocolVersion)

	return nil
}

// Disconnect closes the connection to the ACP server
func (c *KiroACPClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
		c.cmd.Wait()
		c.cmd = nil
	}

	c.conn = nil
	c.connected = false
	return nil
}

// IsConnected returns true if connected to the ACP server
func (c *KiroACPClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// SendMessage sends a message to an agent and returns the response
func (c *KiroACPClient) SendMessage(ctx context.Context, req *MessageRequest) (*MessageResponse, error) {
	if err := ValidateMessageRequest(req); err != nil {
		return nil, err
	}

	c.mu.RLock()
	if !c.connected || c.conn == nil {
		c.mu.RUnlock()
		return nil, ErrNotConnected
	}
	conn := c.conn
	c.mu.RUnlock()

	// Apply request timeout if specified
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}

	// Create a new session for this request
	sessionResp, err := conn.NewSession(ctx, acp.NewSessionRequest{
		McpServers: []acp.McpServer{},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Send the prompt - responses will come via SessionUpdate
	_, err = conn.Prompt(ctx, acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock(req.Message)},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to send prompt: %w", err)
	}

	// For now, return a basic response indicating the message was sent
	// In a full implementation, this would collect responses from SessionUpdate callbacks
	response := &MessageResponse{
		Success:   true,
		Message:   "Message sent successfully. Response will be received via session updates.",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"session_id": sessionResp.SessionId,
			"agent":      req.Agent,
		},
	}

	return response, nil
}

// StreamMessage sends a message to an agent and returns a streaming response
func (c *KiroACPClient) StreamMessage(ctx context.Context, req *MessageRequest) (<-chan *StreamingResponse, error) {
	if err := ValidateMessageRequest(req); err != nil {
		return nil, err
	}

	c.mu.RLock()
	if !c.connected || c.conn == nil {
		c.mu.RUnlock()
		return nil, ErrNotConnected
	}
	conn := c.conn
	c.mu.RUnlock()

	// Create response channel
	respChan := make(chan *StreamingResponse, 10)

	go func() {
		defer close(respChan)

		// Apply request timeout if specified
		streamCtx := ctx
		if req.Timeout > 0 {
			var cancel context.CancelFunc
			streamCtx, cancel = context.WithTimeout(ctx, req.Timeout)
			defer cancel()
		}

		// Create a new session for this streaming request
		sessionResp, err := conn.NewSession(streamCtx, acp.NewSessionRequest{
			McpServers: []acp.McpServer{},
		})
		if err != nil {
			respChan <- &StreamingResponse{
				Type:      "error",
				Error:     fmt.Sprintf("failed to create session: %v", err),
				Timestamp: time.Now(),
			}
			return
		}

		// Send streaming response indicating start
		respChan <- &StreamingResponse{
			Type:      "start",
			Content:   "Starting streaming response...",
			Timestamp: time.Now(),
		}

		// Send the prompt and handle response
		_, promptErr := conn.Prompt(streamCtx, acp.PromptRequest{
			SessionId: sessionResp.SessionId,
			Prompt:    []acp.ContentBlock{acp.TextBlock(req.Message)},
		})
		if promptErr != nil {
			respChan <- &StreamingResponse{
				Type:      "error",
				Error:     fmt.Sprintf("failed to send prompt: %v", promptErr),
				Timestamp: time.Now(),
			}
			return
		}

		// In a real implementation, streaming responses would come via SessionUpdate callbacks
		// For now, send a simple completion message
		respChan <- &StreamingResponse{
			Type:      "text",
			Content:   "Prompt sent successfully. Real-time responses will come via session updates.",
			Timestamp: time.Now(),
		}

		// Send completion signal
		respChan <- &StreamingResponse{
			Type:      "done",
			Content:   "",
			Timestamp: time.Now(),
		}
	}()

	return respChan, nil
}

// ListAgents returns a list of available agents
func (c *KiroACPClient) ListAgents(ctx context.Context) ([]AgentInfo, error) {
	c.mu.RLock()
	if !c.connected || c.conn == nil {
		c.mu.RUnlock()
		return nil, ErrNotConnected
	}
	c.mu.RUnlock()

	// For now, return mock agent info since the ACP protocol
	// doesn't have a direct "list agents" command.
	// In a real implementation, this would query the agent registry.
	agents := []AgentInfo{
		{
			Name:         "kiro-agent",
			Description:  "Kiro CLI Agent via ACP",
			Status:       "available",
			Capabilities: []string{"chat", "code_generation", "file_operations"},
			Model:        "claude-sonnet-4",
			Available:    true,
		},
	}

	return agents, nil
}

// GetAgent returns information about a specific agent
func (c *KiroACPClient) GetAgent(ctx context.Context, name string) (*AgentInfo, error) {
	if name == "" {
		return nil, ErrMissingAgent
	}

	agents, err := c.ListAgents(ctx)
	if err != nil {
		return nil, err
	}

	for _, agent := range agents {
		if agent.Name == name {
			return &agent, nil
		}
	}

	return nil, ErrAgentNotFound
}

// Close closes the client and cleans up resources
func (c *KiroACPClient) Close() error {
	return c.Disconnect()
}

// Ensure KiroACPClient implements Client interface
var _ Client = (*KiroACPClient)(nil)
var _ acp.Client = (*KiroClient)(nil)
