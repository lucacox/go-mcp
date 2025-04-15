// Package protocol defines the basic primitives for the MCP protocol
package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/lucacox/go-mcp/internal/logging"
)

// ProtocolVersion represents the version of the MCP protocol
type ProtocolVersion string

// Protocol versions supported
const (
	// ProtocolVersion20241105 represents the 2024-11-05 MCP protocol version
	ProtocolVersion20241105 ProtocolVersion = "2024-11-05"
	// ProtocolVersion20250326 represents the 2025-03-26 MCP protocol version
	ProtocolVersion20250326 ProtocolVersion = "2025-03-26"
)

// Namespace represents a namespace for RPC methods
type Namespace string

// Standard methods of the MCP protocol
const (
	EmptyNamespace          Namespace = ""
	NotifficationsNamespace Namespace = "notifications"
	RootsNamespace          Namespace = "roots"
	SamplingNamespace       Namespace = "sampling"
	PromptsNamespace        Namespace = "prompts"
	ResourcesNamespace      Namespace = "resources"
	ToolsNamespace          Namespace = "tools"
	CompletionNamespace     Namespace = "completion"
	LoggingNamespace        Namespace = "logging"
)

// Builds a complete RPC method with namespace
func BuildMethod(method string, namespace Namespace) string {
	if namespace == "" {
		return method
	}
	return fmt.Sprintf("%s/%s", namespace, method)
}

func BuildNotificationsMethod(method string, namespace Namespace) string {
	if namespace == "" {
		return fmt.Sprintf("%s/%s", NotifficationsNamespace, method)
	}
	return fmt.Sprintf("%s/%s/%s", NotifficationsNamespace, namespace, method)
}

// CancelParams represents the parameters to cancel a request
type CancelParams struct {
	ID json.RawMessage `json:"id"`
}

// ServerInfoParams represents the parameters for the server information request
type ServerInfoParams struct{}

// ServerInfoResult represents the result of the server information request
type ServerInfoResult struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Capabilities map[string]string `json:"capabilities,omitempty"`
}

// ShutdownParams represents the parameters to close a session
type ShutdownParams struct{}

// Endpoint handles RPC calls for a specific namespace
type Endpoint interface {
	// GetNamespace returns the namespace handled by this endpoint
	GetNamespace() Namespace

	// GetMethods returns the methods supported by this endpoint
	GetMethods() []string

	// HandleRequest handles an RPC request
	HandleRequest(ctx context.Context, method string, params json.RawMessage) (interface{}, error)

	// HandleNotification handles an RPC notification
	HandleNotification(ctx context.Context, method string, params json.RawMessage) (interface{}, error)
}

// EndpointRegistry is a registry of endpoints for the MCP protocol
type EndpointRegistry struct {
	endpoints map[Namespace]Endpoint
	mutex     sync.RWMutex
}

// NewEndpointRegistry creates a new endpoint registry
func NewEndpointRegistry() *EndpointRegistry {
	return &EndpointRegistry{
		endpoints: make(map[Namespace]Endpoint),
	}
}

// RegisterEndpoint registers a new endpoint
func (r *EndpointRegistry) RegisterEndpoint(endpoint Endpoint) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	logging.Info(logging.NewLoggerFactory().CreateLogger("endpoint-registry"), "Registering endpoint", "endpoint", endpoint.GetNamespace())

	r.endpoints[endpoint.GetNamespace()] = endpoint
}

// UnregisterEndpoint removes an endpoint
func (r *EndpointRegistry) UnregisterEndpoint(namespace Namespace) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.endpoints, namespace)
}

// GetEndpoint returns an endpoint
func (r *EndpointRegistry) GetEndpoint(namespace Namespace) (Endpoint, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	endpoint, exists := r.endpoints[namespace]
	return endpoint, exists
}

// HandleRequest handles an RPC request
func (r *EndpointRegistry) HandleRequest(ctx context.Context, fullMethod string, params json.RawMessage) (interface{}, error) {
	// Extract namespace and method
	var method string
	var namespace Namespace = EmptyNamespace
	var isNotification bool = false

	tokens := strings.Split(fullMethod, "/")
	method = tokens[len(tokens)-1]
	if len(tokens) > 1 {
		namespace = Namespace(tokens[len(tokens)-2])
	}
	if namespace == NotifficationsNamespace {
		isNotification = true
		namespace = EmptyNamespace
	}
	if len(tokens) > 2 {
		isNotification = tokens[len(tokens)-3] == string(NotifficationsNamespace)
	}

	if method == "" {
		return nil, &JSONRPCError{
			Code:    -32601,
			Message: "Method not found: invalid method format",
		}
	}

	// Find the endpoint
	r.mutex.RLock()
	endpoint, exists := r.endpoints[namespace]
	r.mutex.RUnlock()

	if !exists {
		return nil, &JSONRPCError{
			Code:    -32601,
			Message: fmt.Sprintf("Method not found: namespace %s not registered", namespace),
		}
	}

	if isNotification {
		// Handle notification
		return endpoint.HandleNotification(ctx, method, params)
	}

	// Handle the request
	return endpoint.HandleRequest(ctx, method, params)
}

// BaseEndpoint is a basic implementation of Endpoint
type BaseEndpoint struct {
	namespace     Namespace
	methods       map[string]MethodHandler
	notifications map[string]MethodHandler
}

// MethodHandler is a function that handles an RPC method
type MethodHandler func(ctx context.Context, params json.RawMessage) (interface{}, error)

// NewBaseEndpoint creates a new base endpoint
func NewBaseEndpoint(namespace Namespace) *BaseEndpoint {
	return &BaseEndpoint{
		namespace:     namespace,
		methods:       make(map[string]MethodHandler),
		notifications: make(map[string]MethodHandler),
	}
}

// GetNamespace returns the namespace handled by this endpoint
func (e *BaseEndpoint) GetNamespace() Namespace {
	return e.namespace
}

// GetMethods returns the methods supported by this endpoint
func (e *BaseEndpoint) GetMethods() []string {
	methods := make([]string, 0, len(e.methods))
	for method := range e.methods {
		methods = append(methods, method)
	}
	return methods
}

// RegisterMethod registers a new method
func (e *BaseEndpoint) RegisterMethod(method string, handler MethodHandler) {
	e.methods[method] = handler
}

func (e *BaseEndpoint) RegisterNotification(method string, handler MethodHandler) {
	e.notifications[method] = handler
}

// HandleRequest handles an RPC request
func (e *BaseEndpoint) HandleRequest(ctx context.Context, method string, params json.RawMessage) (interface{}, error) {
	handler, exists := e.methods[method]
	if !exists {
		return nil, &JSONRPCError{
			Code:    -32601,
			Message: fmt.Sprintf("Method not found: %s/%s", e.namespace, method),
		}
	}

	return handler(ctx, params)
}

func (e *BaseEndpoint) HandleNotification(ctx context.Context, method string, params json.RawMessage) (interface{}, error) {
	handler, exists := e.notifications[method]
	if !exists {
		return nil, &JSONRPCError{
			Code:    -32601,
			Message: fmt.Sprintf("Notification not found: %s/%s", e.namespace, method),
		}
	}

	return handler(ctx, params)
}
