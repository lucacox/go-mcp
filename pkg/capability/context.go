// Package capability defines the capabilities for the MCP protocol
package capability

import (
	"github.com/google/uuid"
)

// CapabilityContext provides the context for executing a capability
type CapabilityContext struct {
	// SessionID is the identifier of the session
	SessionID string

	// RequestID is the identifier of the current request
	RequestID string

	// Timeout is the timeout for the operation
	Timeout int64

	// Cancellable indicates if the operation can be cancelled
	Cancellable bool

	// Options specific to the capability
	Options map[string]interface{}
}

// NewCapabilityContext creates a new CapabilityContext
func NewCapabilityContext() *CapabilityContext {
	return &CapabilityContext{
		RequestID:   uuid.New().String(),
		Timeout:     30000, // 30 seconds by default
		Cancellable: true,
		Options:     make(map[string]interface{}),
	}
}

// WithSessionID sets the session ID
func (c *CapabilityContext) WithSessionID(sessionID string) *CapabilityContext {
	c.SessionID = sessionID
	return c
}

// WithTimeout sets the timeout in milliseconds
func (c *CapabilityContext) WithTimeout(timeout int64) *CapabilityContext {
	c.Timeout = timeout
	return c
}

// WithCancellable sets whether the operation is cancellable
func (c *CapabilityContext) WithCancellable(cancellable bool) *CapabilityContext {
	c.Cancellable = cancellable
	return c
}

// WithOption sets a specific option for the capability
func (c *CapabilityContext) WithOption(key string, value interface{}) *CapabilityContext {
	c.Options[key] = value
	return c
}
