// Package protocol defines the basic primitives for the MCP protocol
package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Session context keys
type sessionKeyType string

const sessionIDKey sessionKeyType = "session_id"

// WithSessionID adds a session ID to the context
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDKey, sessionID)
}

// GetSessionID retrieves a session ID from the context
func GetSessionID(ctx context.Context) (string, bool) {
	sessionID, ok := ctx.Value(sessionIDKey).(string)
	return sessionID, ok
}

// SessionState represents the state of an MCP session
type SessionState int

const (
	// SessionStateUninitialized represents a session that has not yet been initialized
	SessionStateUninitialized SessionState = iota
	// SessionStateInitializing represents a session that is being initialized
	SessionStateInitializing
	// SessionStateActive represents an active session
	SessionStateActive
	// SessionStateClosing represents a session that is being closed
	SessionStateClosing
	// SessionStateClosed represents a closed session
	SessionStateClosed
)

// String returns a textual representation of the session state
func (s SessionState) String() string {
	switch s {
	case SessionStateUninitialized:
		return "uninitialized"
	case SessionStateInitializing:
		return "initializing"
	case SessionStateActive:
		return "active"
	case SessionStateClosing:
		return "closing"
	case SessionStateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// CapabilityDefinition represents the definition of a supported capability
type CapabilityDefinition struct {
	Options json.RawMessage `json:"options,omitempty"`
}

// InitializeParams represents the initialization parameters of a session
type InitializeParams struct {
	ProtocolVersion string                          `json:"protocolVersion"`
	ClientID        string                          `json:"clientId"`
	ClientInfo      map[string]string               `json:"clientInfo,omitempty"`
	Capabilities    map[string]CapabilityDefinition `json:"capabilities"`
}

// InitializeResult represents the result of the initialization of a session
type InitializeResult struct {
	ProtocolVersion string                          `json:"protocolVersion"`
	ServerID        string                          `json:"-"`
	ServerInfo      map[string]string               `json:"serverInfo,omitempty"`
	Capabilities    map[string]CapabilityDefinition `json:"capabilities"`
	Instructions    string                          `json:"instructions,omitempty"`
}

// Session represents an MCP session
type Session struct {
	ID         string
	State      SessionState
	mutex      sync.RWMutex
	dispatcher *JSONRPCDispatcher

	// Initialization data
	ClientID           string
	ClientInfo         map[string]string
	ClientCapabilities map[string]CapabilityDefinition

	ServerID           string
	ServerInfo         map[string]string
	ServerCapabilities map[string]CapabilityDefinition

	// Metadata
	CreatedAt    time.Time
	LastActiveAt time.Time
}

// NewSession creates a new MCP session
func NewSession(dispatcher *JSONRPCDispatcher) *Session {
	return &Session{
		ID:           uuid.New().String(),
		State:        SessionStateUninitialized,
		dispatcher:   dispatcher,
		CreatedAt:    time.Now(),
		LastActiveAt: time.Now(),

		ClientCapabilities: make(map[string]CapabilityDefinition),
		ServerCapabilities: make(map[string]CapabilityDefinition),
		ClientInfo:         make(map[string]string),
		ServerInfo:         make(map[string]string),
	}
}

// GetState returns the current session state
func (s *Session) GetState() SessionState {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.State
}

// SetState sets the session state
func (s *Session) SetState(state SessionState) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.State = state
}

// UpdateLastActiveTime updates the last activity timestamp
func (s *Session) UpdateLastActiveTime() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.LastActiveAt = time.Now()
}

// Initialize initializes the session with the client's parameters
func (s *Session) Initialize(ctx context.Context, params *InitializeParams, version ProtocolVersion) (*InitializeResult, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.State != SessionStateUninitialized {
		return nil, fmt.Errorf("session already initialized")
	}

	s.State = SessionStateInitializing

	// Save client information
	s.ClientID = params.ClientID
	s.ClientInfo = params.ClientInfo
	s.ClientCapabilities = params.Capabilities

	// Update the timestamp
	s.LastActiveAt = time.Now()

	// At this point, in a complete implementation, we should:
	// 1. Verify the compatibility of the capabilities
	// 2. Negotiate the options

	// Prepare the result
	result := &InitializeResult{
		ProtocolVersion: string(version),
		ServerID:        s.ServerID,
		ServerInfo:      s.ServerInfo,
		Capabilities:    s.ServerCapabilities,
	}

	return result, nil
}

// Close closes the session
func (s *Session) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.State == SessionStateClosed {
		return nil
	}

	s.State = SessionStateClosing

	// Perform resource cleanup

	s.State = SessionStateClosed
	return nil
}

// IsActive checks if the session is active
func (s *Session) IsActive() bool {
	state := s.GetState()
	return state == SessionStateActive
}

// HasCapability checks if a capability is supported
func (s *Session) HasCapability(capabilityType string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	_, exists := s.ClientCapabilities[capabilityType]
	return exists
}

// Call sends an RPC request through the session
func (s *Session) Call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	if !s.IsActive() {
		return nil, fmt.Errorf("session not active")
	}

	s.UpdateLastActiveTime()
	return s.dispatcher.Call(ctx, method, params)
}

// Notify sends an RPC notification through the session
func (s *Session) Notify(ctx context.Context, method string, params interface{}) error {
	if !s.IsActive() {
		return fmt.Errorf("session not active")
	}

	s.UpdateLastActiveTime()
	return s.dispatcher.Notify(ctx, method, params)
}
