// Package protocol defines the basic primitives for the MCP protocol
package protocol

import (
	"context"
	"io"
	"sync"
)

// Transport defines the basic transport interface for the MCP protocol
// Transports are responsible for transmitting messages between client and server
type Transport interface {
	// Send sends a message to the recipient
	Send(ctx context.Context, data []byte) error

	// Receive receives a message from the sender
	Receive(ctx context.Context) ([]byte, error)

	// Close closes the transport connection
	Close() error
}

// BiDirectionalTransport is a transport that supports separate input and output streams
type BiDirectionalTransport interface {
	Transport

	// Reader returns the reader for incoming messages
	Reader() io.Reader

	// Writer returns the writer for outgoing messages
	Writer() io.Writer
}

// TransportCreator is a factory function for creating transports
type TransportCreator func(ctx context.Context, options map[string]interface{}) (Transport, error)

// TransportRegistry maintains a registry of available transport creators
type TransportRegistry struct {
	creators map[string]TransportCreator
	mu       sync.RWMutex
}

// DefaultTransportRegistry is the global default transport registry
// It contains all available transport implementations
var DefaultTransportRegistry = NewTransportRegistry()

// TransportType constants for commonly used transports
const (
	TransportTypeStdio = "stdio"
	TransportTypeHTTP  = "http"
)

// NewTransportRegistry creates a new transport registry
func NewTransportRegistry() *TransportRegistry {
	return &TransportRegistry{
		creators: make(map[string]TransportCreator),
		mu:       sync.RWMutex{},
	}
}

// Register registers a new transport creator
func (r *TransportRegistry) Register(name string, creator TransportCreator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.creators[name] = creator
}

// Create creates a new transport with the specified type
func (r *TransportRegistry) Create(ctx context.Context, transportType string, options map[string]interface{}) (Transport, error) {
	r.mu.RLock()
	creator, exists := r.creators[transportType]
	r.mu.RUnlock()

	if !exists {
		return nil, &TransportError{
			Message: "transport type not supported: " + transportType,
		}
	}

	return creator(ctx, options)
}

// HasTransport checks if a specific transport is registered
func (r *TransportRegistry) HasTransport(transportType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.creators[transportType]
	return exists
}

// GetSupportedTransports returns a list of all registered transport types
func (r *TransportRegistry) GetSupportedTransports() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	transports := make([]string, 0, len(r.creators))
	for name := range r.creators {
		transports = append(transports, name)
	}
	return transports
}

// GetCreator returns the creator function for a specific transport type
func (r *TransportRegistry) GetCreator(transportType string) (TransportCreator, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	creator, exists := r.creators[transportType]
	if !exists {
		return nil, &TransportError{
			Message: "transport type not supported: " + transportType,
		}
	}

	return creator, nil
}

// CreateTransport is a convenience function to create a transport from the default registry
func CreateTransport(ctx context.Context, transportType string, options map[string]interface{}) (Transport, error) {
	return DefaultTransportRegistry.Create(ctx, transportType, options)
}

// TransportError represents a transport error
type TransportError struct {
	Message string
	Cause   error
}

// Error implements the error interface
func (e *TransportError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unwrap implements the unwrapping interface
func (e *TransportError) Unwrap() error {
	return e.Cause
}

// WithCause adds a causal error
func (e *TransportError) WithCause(err error) *TransportError {
	e.Cause = err
	return e
}
