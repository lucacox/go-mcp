// Package capability defines the capabilities for the MCP protocol
package capability

import (
	"context"
	"encoding/json"

	"github.com/lucacox/go-mcp/pkg/protocol"
)

// CapabilityType represents the type of a capability
type CapabilityType string

// Common capability types defined in the MCP specification
const (
	// Tools capability for tool execution
	Tools CapabilityType = "tools"
	// Resources capability for resource management
	Resources CapabilityType = "resources"
	// Prompts capability for prompt management
	Prompts CapabilityType = "prompts"
	// Sampling capability for controlling output generation
	Sampling CapabilityType = "sampling"
	// Roots capability for controlling the context window
	Roots CapabilityType = "roots"
)

// Capability represents a MCP capability
type Capability interface {
	// GetType returns the type of the capability
	GetType() CapabilityType

	// GetDescription returns the description of the capability
	GetDescription() string

	// GetOptions returns the options supported by the capability
	GetOptions() json.RawMessage

	GetEndpoint() protocol.Endpoint

	// Initialize initializes the capability with the provided options
	Initialize(ctx context.Context, options json.RawMessage) error

	// Shutdown shuts down the capability
	Shutdown(ctx context.Context) error
}

// CapabilityFactory is a function type that creates a new capability
type CapabilityFactory func(...interface{}) (Capability, error)

// CapabilityRegistry is a registry of capability factories
type CapabilityRegistry struct {
	factories map[CapabilityType]CapabilityFactory

	capabilities map[CapabilityType]Capability
}

// NewCapabilityRegistry creates a new capability registry
func NewCapabilityRegistry() *CapabilityRegistry {
	return &CapabilityRegistry{
		factories:    make(map[CapabilityType]CapabilityFactory),
		capabilities: make(map[CapabilityType]Capability),
	}
}

// RegisterFactory registers a new capability factory
func (r *CapabilityRegistry) RegisterFactory(capType CapabilityType, factory CapabilityFactory) {
	r.factories[capType] = factory
}

// GetSupportedTypes returns the supported capability types
func (r *CapabilityRegistry) GetSupportedTypes() []CapabilityType {
	types := make([]CapabilityType, 0, len(r.factories))
	for capType := range r.factories {
		types = append(types, capType)
	}
	return types
}

// Create creates a new capability instance
func (r *CapabilityRegistry) Create(capType CapabilityType, options ...interface{}) (Capability, error) {
	factory, exists := r.factories[capType]
	if !exists {
		return nil, &CapabilityError{
			Message: "capability type not supported: " + string(capType),
		}
	}

	return factory(options...)
}

func (r *CapabilityRegistry) GetCapabilities() []Capability {
	caps := make([]Capability, 0, len(r.capabilities))
	for _, cap := range r.capabilities {
		caps = append(caps, cap)
	}
	return caps
}

func (r *CapabilityRegistry) AddCapability(capability Capability) {
	r.capabilities[capability.GetType()] = capability
}

// GetCapability returns a capability by type
func (r *CapabilityRegistry) GetCapability(capType CapabilityType) Capability {
	return r.capabilities[capType]
}
