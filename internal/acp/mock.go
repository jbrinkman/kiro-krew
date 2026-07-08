// Package acp provides a mock implementation of the Agent Communication Protocol SDK
// This is a development placeholder until the official acp-go-sdk is available
package acp

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ResponseType represents different types of ACP responses
type ResponseType int

const (
	ResponseTypeText ResponseType = iota
	ResponseTypeJSON
	ResponseTypeContext
	ResponseTypeError
)

// Response represents an ACP protocol response
type Response struct {
	Type     ResponseType
	Content  string
	Complete bool
	Context  Context
	Error    error
}

// Context represents ACP context information
type Context struct {
	ContextUsage string
	Model        string
}

// Config represents ACP client configuration
type Config struct {
	Context context.Context
}

// ChatConfig represents chat session configuration
type ChatConfig struct {
	Agent string
}

// Client represents an ACP client connection
type Client struct {
	ctx    context.Context
	closed bool
	mu     sync.RWMutex
}

// Session represents an ACP chat session
type Session struct {
	agent     string
	client    *Client
	responses chan Response
	closed    bool
	mu        sync.RWMutex
}

// NewClient creates a new mock ACP client
func NewClient(config Config) (*Client, error) {
	if config.Context == nil {
		config.Context = context.Background()
	}

	return &Client{
		ctx: config.Context,
	}, nil
}

// StartChat starts a new chat session with the specified agent
func (c *Client) StartChat(config ChatConfig) (*Session, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client is closed")
	}
	c.mu.RUnlock()

	session := &Session{
		agent:     config.Agent,
		client:    c,
		responses: make(chan Response, 100),
	}

	// Send welcome message
	go func() {
		time.Sleep(100 * time.Millisecond)
		select {
		case session.responses <- Response{
			Type:    ResponseTypeText,
			Content: fmt.Sprintf("Hello! I'm the %s agent. How can I help you with your planning today?", config.Agent),
		}:
		case <-c.ctx.Done():
		}

		// Send context info
		time.Sleep(50 * time.Millisecond)
		select {
		case session.responses <- Response{
			Type: ResponseTypeContext,
			Context: Context{
				ContextUsage: "2k/200k",
				Model:        "claude-sonnet-4",
			},
		}:
		case <-c.ctx.Done():
		}
	}()

	return session, nil
}

// Close closes the ACP client
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	return nil
}

// SendMessage sends a message to the agent (mock implementation)
func (s *Session) SendMessage(content string) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return fmt.Errorf("session is closed")
	}
	s.mu.RUnlock()

	// Simulate agent response
	go func() {
		// Simulate processing delay
		time.Sleep(500 * time.Millisecond)

		// Mock response based on content
		var response Response
		if content == "" {
			response = Response{
				Type:    ResponseTypeText,
				Content: "I didn't receive any message. Could you please try again?",
			}
		} else {
			response = Response{
				Type:    ResponseTypeText,
				Content: fmt.Sprintf("I understand you want help with: %s\n\nLet me think about this and provide you with a structured plan. What specific aspects would you like me to focus on?", content),
			}
		}

		select {
		case s.responses <- response:
		case <-s.client.ctx.Done():
		}

		// Update context
		time.Sleep(100 * time.Millisecond)
		select {
		case s.responses <- Response{
			Type: ResponseTypeContext,
			Context: Context{
				ContextUsage: "3k/200k",
				Model:        "claude-sonnet-4",
			},
		}:
		case <-s.client.ctx.Done():
		}
	}()

	return nil
}

// Stream returns a channel for receiving responses
func (s *Session) Stream() <-chan Response {
	return s.responses
}

// Close closes the session
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	close(s.responses)
	return nil
}

// IsRetryableError checks if an error is retryable (mock implementation)
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Mock: consider some errors as retryable
	errorStr := err.Error()
	return errorStr == "connection timeout" || errorStr == "temporary network error"
}
