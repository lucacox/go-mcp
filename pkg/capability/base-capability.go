// Package capability defines the capabilities for the MCP protocol
package capability

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lucacox/go-mcp/pkg/protocol"
)

// CapabilityError represents a capability error
type CapabilityError struct {
	Message string
	Cause   error
}

// Error implements the error interface
func (e *CapabilityError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unwrap implements the unwrapping interface
func (e *CapabilityError) Unwrap() error {
	return e.Cause
}

// WithCause adds a causal error
func (e *CapabilityError) WithCause(err error) *CapabilityError {
	e.Cause = err
	return e
}

// BasicCapability provides a simple base implementation of the Capability interface
type BasicCapability struct {
	TypeName    CapabilityType
	DescText    string
	OptionsData json.RawMessage
}

// GetType returns the capability type
func (bc *BasicCapability) GetType() CapabilityType {
	return bc.TypeName
}

// GetDescription returns the capability description
func (bc *BasicCapability) GetDescription() string {
	return bc.DescText
}

// GetOptions returns the capability options
func (bc *BasicCapability) GetOptions() json.RawMessage {
	return bc.OptionsData
}

// GetEndpoint returns the capability endpoint
func (bc *BasicCapability) GetEndpoint() protocol.Endpoint {
	// Basic implementation does not have an endpoint
	return nil
}

// Initialize is a no-op in this basic implementation
func (bc *BasicCapability) Initialize(ctx context.Context, options json.RawMessage) error {
	// No-op for basic implementation
	return nil
}

// Shutdown is a no-op in this basic implementation
func (bc *BasicCapability) Shutdown(ctx context.Context) error {
	// No-op for basic implementation
	return nil
}

// CreateBasicCapability creates a new basic capability for the given type
func CreateBasicCapability(capType CapabilityType) (Capability, error) {
	desc := fmt.Sprintf("Basic %s capability implementation", capType)

	// Create a basic implementation with empty options
	return &BasicCapability{
		TypeName:    capType,
		DescText:    desc,
		OptionsData: json.RawMessage(`{}`),
	}, nil
}

// BaseServerCapability base implementation for all server capabilities
type BaseServerCapability struct {
	BasicCapability
}

// IsServerCapability identifies this as a server capability
func (s *BaseServerCapability) IsServerCapability() bool {
	return true
}

// IsClientCapability implementation for server capabilities
func (s *BaseServerCapability) IsClientCapability() bool {
	return false
}

// BaseClientCapability base implementation for all client capabilities
type BaseClientCapability struct {
	BasicCapability
}

// IsServerCapability implementation for client capabilities
func (c *BaseClientCapability) IsServerCapability() bool {
	return false
}

// IsClientCapability identifies this as a client capability
func (c *BaseClientCapability) IsClientCapability() bool {
	return true
}
