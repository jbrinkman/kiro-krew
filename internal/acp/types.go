package acp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"
)

// Common errors
var (
	ErrNotConnected          = errors.New("not connected to ACP server")
	ErrAlreadyConnected      = errors.New("already connected to ACP server")
	ErrConnectionFailed      = errors.New("failed to connect to ACP server")
	ErrInvalidRequest        = errors.New("invalid request")
	ErrMissingAgent          = errors.New("agent name is required")
	ErrMissingMessage        = errors.New("message is required")
	ErrInvalidResponseFormat = errors.New("response format must be 'json' or 'text'")
	ErrAgentNotFound         = errors.New("agent not found")
	ErrRequestTimeout        = errors.New("request timed out")
	ErrStreamingFailed       = errors.New("streaming failed")
	ErrAuthenticationFailed  = errors.New("authentication failed")
)

// MessageRequest represents a request to send a message to an ACP agent
type MessageRequest struct {
	// Agent is the name of the agent to communicate with
	Agent string `json:"agent"`

	// Message is the text message to send to the agent
	Message string `json:"message"`

	// Context provides additional context for the message
	Context map[string]interface{} `json:"context,omitempty"`

	// Streaming indicates if the response should be streamed
	Streaming bool `json:"streaming,omitempty"`

	// ResponseFormat specifies the expected response format ("json" or "text")
	ResponseFormat string `json:"response_format,omitempty"`

	// Timeout for the request
	Timeout time.Duration `json:"timeout,omitempty"`
}

// MessageResponse represents a response from an ACP agent
type MessageResponse struct {
	// Success indicates if the request was successful
	Success bool `json:"success"`

	// Message contains the response text from the agent
	Message string `json:"message,omitempty"`

	// Data contains structured response data when ResponseFormat is "json"
	Data json.RawMessage `json:"data,omitempty"`

	// Error contains error information if the request failed
	Error string `json:"error,omitempty"`

	// Metadata contains additional response metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Timestamp when the response was received
	Timestamp time.Time `json:"timestamp"`
}

// StreamingResponse represents a streaming response from an ACP agent
type StreamingResponse struct {
	// Type indicates the type of streaming data ("text", "json", "error", "done")
	Type string `json:"type"`

	// Content contains the streaming content
	Content string `json:"content,omitempty"`

	// Data contains structured data for "json" type responses
	Data json.RawMessage `json:"data,omitempty"`

	// Error contains error information for "error" type responses
	Error string `json:"error,omitempty"`

	// Timestamp when the streaming data was received
	Timestamp time.Time `json:"timestamp"`
}

// AgentInfo represents information about an available ACP agent
type AgentInfo struct {
	// Name is the agent name/identifier
	Name string `json:"name"`

	// Description provides a description of the agent's purpose
	Description string `json:"description,omitempty"`

	// Status indicates the agent's current status
	Status string `json:"status"`

	// Capabilities lists the agent's capabilities
	Capabilities []string `json:"capabilities,omitempty"`

	// Model indicates the AI model used by the agent
	Model string `json:"model,omitempty"`

	// Available indicates if the agent is currently available
	Available bool `json:"available"`
}

// ConnectionConfig represents ACP connection configuration
type ConnectionConfig struct {
	// KiroCLIPath is the path to the Kiro CLI executable
	KiroCLIPath string `json:"kiro_cli_path,omitempty"`

	// MaxRetries is the maximum number of retry attempts
	MaxRetries int `json:"max_retries,omitempty"`

	// RetryDelay is the delay between retry attempts
	RetryDelay time.Duration `json:"retry_delay,omitempty"`

	// ConnectionTimeout is the timeout for establishing connections
	ConnectionTimeout time.Duration `json:"connection_timeout,omitempty"`

	// RequestTimeout is the default timeout for requests
	RequestTimeout time.Duration `json:"request_timeout,omitempty"`
}

// Client defines the interface for ACP client operations
type Client interface {
	// Connect establishes a connection to the ACP server
	Connect(ctx context.Context) error

	// Disconnect closes the connection to the ACP server
	Disconnect() error

	// IsConnected returns true if connected to the ACP server
	IsConnected() bool

	// SendMessage sends a message to an agent and returns the response
	SendMessage(ctx context.Context, req *MessageRequest) (*MessageResponse, error)

	// StreamMessage sends a message to an agent and returns a streaming response
	StreamMessage(ctx context.Context, req *MessageRequest) (<-chan *StreamingResponse, error)

	// ListAgents returns a list of available agents
	ListAgents(ctx context.Context) ([]AgentInfo, error)

	// GetAgent returns information about a specific agent
	GetAgent(ctx context.Context, name string) (*AgentInfo, error)

	// Close closes the client and cleans up resources
	Close() error
}

// StreamReader wraps an io.Reader to provide streaming response parsing
type StreamReader interface {
	io.Reader

	// ReadResponse reads the next streaming response
	ReadResponse() (*StreamingResponse, error)

	// IsDone returns true if the stream is complete
	IsDone() bool
}

// AuthProvider defines the interface for ACP authentication
type AuthProvider interface {
	// GetCredentials returns the authentication credentials
	GetCredentials(ctx context.Context) (map[string]string, error)

	// RefreshCredentials refreshes the authentication credentials
	RefreshCredentials(ctx context.Context) error
}

// DefaultConnectionConfig returns default connection configuration
func DefaultConnectionConfig() *ConnectionConfig {
	return &ConnectionConfig{
		KiroCLIPath:       "kiro-cli",
		MaxRetries:        3,
		RetryDelay:        1 * time.Second,
		ConnectionTimeout: 30 * time.Second,
		RequestTimeout:    60 * time.Second,
	}
}

// ValidateMessageRequest validates a MessageRequest
func ValidateMessageRequest(req *MessageRequest) error {
	if req == nil {
		return ErrInvalidRequest
	}

	if req.Agent == "" {
		return ErrMissingAgent
	}

	if req.Message == "" {
		return ErrMissingMessage
	}

	if req.ResponseFormat != "" && req.ResponseFormat != "json" && req.ResponseFormat != "text" {
		return ErrInvalidResponseFormat
	}

	return nil
}
