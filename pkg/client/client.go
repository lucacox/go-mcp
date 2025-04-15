// Package client provides the implementation of an MCP client
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/lucacox/go-mcp/internal/logging"
	"github.com/lucacox/go-mcp/pkg/capabilities/prompts"
	"github.com/lucacox/go-mcp/pkg/capabilities/resources"
	"github.com/lucacox/go-mcp/pkg/capabilities/roots"
	"github.com/lucacox/go-mcp/pkg/capabilities/sampling"
	"github.com/lucacox/go-mcp/pkg/capabilities/tools"
	"github.com/lucacox/go-mcp/pkg/capability"
	"github.com/lucacox/go-mcp/pkg/protocol"
)

// Replace "github.com/lucacox/go-mcp" with your actual Go module

// Client represents an MCP client that connects to a server
type Client struct {
	// Client ID
	ID string

	// Client information
	Info map[string]string

	// Transport used for communication
	transport protocol.Transport

	// Transport registry used for creating transports
	transportRegistry *protocol.TransportRegistry

	// JSON-RPC dispatcher
	dispatcher *protocol.JSONRPCDispatcher

	// Session represents the active session with the server
	session *protocol.Session

	// Registry of supported capabilities
	capabilityRegistry *capability.CapabilityRegistry

	// Logger
	loggerFactory *logging.LoggerFactory
	logger        *slog.Logger
}

// NewClient creates a new MCP client with an existing transport
func NewClient(transport protocol.Transport, options ...ClientOption) *Client {
	client := &Client{
		ID:                 uuid.New().String(),
		Info:               make(map[string]string),
		transport:          transport,
		transportRegistry:  protocol.DefaultTransportRegistry,
		capabilityRegistry: capability.NewCapabilityRegistry(),
	}

	// Register default capabilities for client and server
	client.capabilityRegistry.RegisterFactory(capability.Roots, roots.RootsCapabilityFactory)
	client.capabilityRegistry.RegisterFactory(capability.Sampling, sampling.SamplingCapabilityFactory)
	client.capabilityRegistry.RegisterFactory(capability.Prompts, prompts.PromptsCapabilityFactory)
	client.capabilityRegistry.RegisterFactory(capability.Resources, resources.ResourcesCapabilityFactory)
	client.capabilityRegistry.RegisterFactory(capability.Tools, tools.ToolsCapabilityFactory)

	// Apply options
	for _, option := range options {
		option(client)
	}

	if client.loggerFactory != nil {
		client.logger = client.loggerFactory.CreateLogger("mcp-client")
	}

	// Create JSON-RPC dispatcher
	client.dispatcher = protocol.NewJSONRPCDispatcher(transport, client)

	return client
}

// NewClientWithTransport creates a new MCP client using the specified transport type
func NewClientWithTransport(ctx context.Context, transportType string, transportOptions map[string]interface{}, options ...ClientOption) (*Client, error) {
	// Create a client with default values
	client := &Client{
		ID:                 uuid.New().String(),
		Info:               make(map[string]string),
		transportRegistry:  protocol.DefaultTransportRegistry,
		capabilityRegistry: capability.NewCapabilityRegistry(),
	}

	// Apply options
	for _, option := range options {
		option(client)
	}

	// Create the transport
	transport, err := client.transportRegistry.Create(ctx, transportType, transportOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %w", err)
	}

	client.transport = transport

	// Create JSON-RPC dispatcher
	client.dispatcher = protocol.NewJSONRPCDispatcher(transport, client)

	return client, nil
}

// HandleRequest implements the RPCHandler interface
func (c *Client) HandleRequest(ctx context.Context, method string, params json.RawMessage) (interface{}, error) {
	// Here we handle incoming requests from the server
	// For now it's an empty implementation
	return nil, &protocol.JSONRPCError{
		Code:    -32601,
		Message: "Method not implemented",
	}
}

// Initialize initializes the client and establishes a session with the server
func (c *Client) Initialize(ctx context.Context) error {
	// Prepare initialization parameters
	params := &protocol.InitializeParams{
		ClientID:     c.ID,
		ClientInfo:   c.Info,
		Capabilities: make(map[string]protocol.CapabilityDefinition),
	}

	// TODO: capabilities

	// Start the dispatcher to receive messages
	c.dispatcher.Start()

	// Send initialization request
	result := &protocol.InitializeResult{}
	rawResult, err := c.dispatcher.Call(ctx, protocol.BuildMethod("initialize", protocol.EmptyNamespace), params)
	if err != nil {
		return fmt.Errorf("error initializing: %w", err)
	}

	if err := json.Unmarshal(rawResult, result); err != nil {
		return fmt.Errorf("error parsing initialization result: %w", err)
	}

	// TODO: check if server supports the requested capabilities and version

	// Create the session
	c.session = protocol.NewSession(c.dispatcher)
	c.session.ServerID = result.ServerID
	c.session.ServerInfo = result.ServerInfo
	c.session.ServerCapabilities = result.Capabilities
	c.session.SetState(protocol.SessionStateActive)

	// Notify the server that initialization is complete
	if err := c.dispatcher.Notify(ctx, protocol.BuildNotificationsMethod("initialized", protocol.EmptyNamespace), nil); err != nil {
		return fmt.Errorf("error sending initialized notification: %w", err)
	}

	return nil
}

// Call sends an RPC request to the server
func (c *Client) Call(ctx context.Context, method string, namespace protocol.Namespace, params interface{}) (json.RawMessage, error) {
	if c.session == nil || !c.session.IsActive() {
		return nil, fmt.Errorf("no active session")
	}

	fullMethod := protocol.BuildMethod(method, namespace)
	return c.session.Call(ctx, fullMethod, params)
}

// Notify sends an RPC notification to the server
func (c *Client) Notify(ctx context.Context, method string, namespace protocol.Namespace, params interface{}) error {
	if c.session == nil || !c.session.IsActive() {
		return fmt.Errorf("no active session")
	}

	fullMethod := protocol.BuildNotificationsMethod(method, namespace)
	return c.session.Notify(ctx, fullMethod, params)
}

// IsConnected checks if the client is connected
func (c *Client) IsConnected() bool {
	return c.session != nil && c.session.IsActive()
}

// GetServerID returns the server ID
func (c *Client) GetServerID() string {
	if c.session == nil {
		return ""
	}
	return c.session.ServerID
}

// GetServerInfo returns information about the server
func (c *Client) GetServerInfo() map[string]string {
	if c.session == nil {
		return nil
	}
	return c.session.ServerInfo
}

// HasServerCapability checks if the server supports a capability
func (c *Client) HasServerCapability(capType string) bool {
	if c.session == nil {
		return false
	}
	return c.session.HasCapability(capType)
}

// Close closes the client connection
func (c *Client) Close() error {
	if c.transport != nil {
		return c.transport.Close()
	}
	return nil
}
