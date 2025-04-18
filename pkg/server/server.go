// Package server provides the implementation of an MCP server
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"github.com/google/uuid"

	"github.com/lucacox/go-mcp/internal/logging"
	"github.com/lucacox/go-mcp/pkg/capabilities/prompts"
	"github.com/lucacox/go-mcp/pkg/capabilities/resources"
	"github.com/lucacox/go-mcp/pkg/capabilities/roots"
	"github.com/lucacox/go-mcp/pkg/capabilities/sampling"
	"github.com/lucacox/go-mcp/pkg/capabilities/tools"
	"github.com/lucacox/go-mcp/pkg/capability"
	"github.com/lucacox/go-mcp/pkg/protocol"
	httptransport "github.com/lucacox/go-mcp/pkg/transport/http"
	stdiotransport "github.com/lucacox/go-mcp/pkg/transport/stdio"
)

// Server represents an MCP server that manages client connections
type Server struct {
	// Server ID
	ID string

	// Server information
	Info map[string]string

	// Server version
	Version string

	// Server name
	Name string

	// supported MCP versions
	SupportedVersions []protocol.ProtocolVersion

	// Endpoint registry for RPC method management
	endpointRegistry *protocol.EndpointRegistry

	// Registry for supported capabilities
	capabilityRegistry *capability.CapabilityRegistry

	// Transport registry for communication
	transportRegistry *protocol.TransportRegistry

	// Active sessions
	sessions      map[string]*protocol.Session
	sessionsMutex sync.RWMutex

	// Logger
	logger        *slog.Logger
	loggerFactory *logging.LoggerFactory
}

// NewServer creates a new MCP server
func NewServer(options ...ServerOption) *Server {
	server := &Server{
		ID:                 uuid.New().String(),
		Info:               make(map[string]string),
		Name:               "MCP Server",
		Version:            "1.0.0",
		SupportedVersions:  []protocol.ProtocolVersion{protocol.ProtocolVersion20250326},
		endpointRegistry:   protocol.NewEndpointRegistry(),
		capabilityRegistry: capability.NewCapabilityRegistry(),
		transportRegistry:  protocol.DefaultTransportRegistry, // Default to global registry
		sessions:           make(map[string]*protocol.Session),
	}

	// Register default capabilities for client and server
	server.capabilityRegistry.RegisterFactory(capability.Roots, roots.RootsCapabilityFactory)
	server.capabilityRegistry.RegisterFactory(capability.Sampling, sampling.SamplingCapabilityFactory)
	server.capabilityRegistry.RegisterFactory(capability.Prompts, prompts.PromptsCapabilityFactory)
	server.capabilityRegistry.RegisterFactory(capability.Resources, resources.ResourcesCapabilityFactory)
	server.capabilityRegistry.RegisterFactory(capability.Tools, tools.ToolsCapabilityFactory)

	// Apply the options
	for _, option := range options {
		option(server)
	}

	if server.loggerFactory != nil {
		server.logger = server.loggerFactory.CreateLogger("mcp-server")
	}

	// Register the base MCP endpoint
	mcpEndpoint := protocol.NewBaseEndpoint(protocol.EmptyNamespace)
	mcpEndpoint.RegisterMethod("initialize", server.handleInitialize)
	mcpEndpoint.RegisterMethod("ping", server.handlePing)
	mcpEndpoint.RegisterNotification("initialized", server.handleInitialized)
	server.endpointRegistry.RegisterEndpoint(mcpEndpoint)

	return server
}

// HandleConnection handles a new client connection
func (s *Server) HandleConnection(transport protocol.Transport) {
	// Create a new JSON-RPC dispatcher for this connection
	dispatcher := protocol.NewJSONRPCDispatcher(transport, s)

	// Create a new session
	session := protocol.NewSession(dispatcher)
	session.ServerID = s.ID
	session.ServerInfo = s.Info

	// Store the session
	s.sessionsMutex.Lock()
	s.sessions[session.ID] = session
	s.sessionsMutex.Unlock()

	// Set the session ID in the dispatcher
	dispatcher.SetSessionID(session.ID)

	// Start the dispatcher
	dispatcher.Start()
}

// HandleRequest implements the RPCHandler interface
func (s *Server) HandleRequest(ctx context.Context, method string, params json.RawMessage) (interface{}, error) {
	// Delegate request handling to the endpoint registry
	return s.endpointRegistry.HandleRequest(ctx, method, params)
}

// Handler methods for the base MCP endpoint

// handleInitialize handles the initialization request
func (s *Server) handleInitialize(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var initParams protocol.InitializeParams
	if err := json.Unmarshal(params, &initParams); err != nil {
		return nil, &protocol.JSONRPCError{
			Code:    -32700,
			Message: "Parse error: " + err.Error(),
		}
	}

	// Get the session ID from the context
	sessionID, ok := protocol.GetSessionID(ctx)
	if !ok {
		return nil, &protocol.JSONRPCError{
			Code:    -32603,
			Message: "Internal error: session ID not found in context",
		}
	}

	// Find the session using the ID
	s.sessionsMutex.RLock()
	session, exists := s.sessions[sessionID]
	s.sessionsMutex.RUnlock()

	if !exists {
		return nil, &protocol.JSONRPCError{
			Code:    -32603,
			Message: "Internal error: session not found",
		}
	}

	// check if client version is compatible
	versionIdx := slices.Index(s.SupportedVersions, protocol.ProtocolVersion(initParams.ProtocolVersion))
	if versionIdx == -1 {
		logging.Error(s.logger, "Unsupported client protocol version", "version", initParams.ProtocolVersion)
		versionIdx = 0
	}
	version := s.SupportedVersions[versionIdx]

	// Get the server capabilities to send to the client
	serverCapabilities, err := s.GetCapabilities(ctx)
	if err != nil {
		return nil, &protocol.JSONRPCError{
			Code:    -32603,
			Message: "Internal error: " + err.Error(),
		}
	}

	// Store the server capabilities and info in the session
	session.ServerID = s.ID
	session.ServerCapabilities = serverCapabilities
	session.ServerInfo = map[string]string{
		"name":    s.Name,
		"version": s.Version,
	}
	// Add any additional server info
	for k, v := range s.Info {
		session.ServerInfo[k] = v
	}

	// Initialize the session
	result, err := session.Initialize(ctx, &initParams, version)
	if err != nil {
		return nil, &protocol.JSONRPCError{
			Code:    -32603,
			Message: "Internal error: " + err.Error(),
		}
	}

	return result, nil
}

func (s *Server) handlePing(ctx context.Context, params json.RawMessage) (interface{}, error) {
	// Get the session ID from the context (for logging, etc.)
	sessionID, ok := protocol.GetSessionID(ctx)
	if ok && s.logger != nil {
		logging.Debug(s.logger, "Ping received", slog.String("sessionID", sessionID))
	}

	// respond with an empty Response
	return nil, nil
}

func (s *Server) handleInitialized(ctx context.Context, params json.RawMessage) (interface{}, error) {
	// Get the session ID from the context
	sessionID, ok := protocol.GetSessionID(ctx)
	if !ok {
		return nil, &protocol.JSONRPCError{
			Code:    -32603,
			Message: "Internal error: session ID not found in context",
		}
	}

	// Find the session using the ID
	s.sessionsMutex.RLock()
	session, exists := s.sessions[sessionID]
	s.sessionsMutex.RUnlock()

	if !exists {
		return nil, &protocol.JSONRPCError{
			Code:    -32603,
			Message: "Internal error: session not found",
		}
	}

	if session.State != protocol.SessionStateInitializing {
		return nil, &protocol.JSONRPCError{
			Code:    -32603,
			Message: "Session not in initializing state",
		}
	}

	session.SetState(protocol.SessionStateActive)

	if s.logger != nil {
		logging.Debug(s.logger, "Session initialized", slog.String("sessionID", session.ID))
	}

	return nil, nil
}

// RegisterEndpoint registers a new endpoint
func (s *Server) RegisterEndpoint(endpoint protocol.Endpoint) {
	s.endpointRegistry.RegisterEndpoint(endpoint)
}

// GetSession returns a session by ID
func (s *Server) GetSession(id string) (*protocol.Session, bool) {
	s.sessionsMutex.RLock()
	defer s.sessionsMutex.RUnlock()
	session, exists := s.sessions[id]
	return session, exists
}

// GetActiveSessions returns all active sessions
func (s *Server) GetActiveSessions() []*protocol.Session {
	s.sessionsMutex.RLock()
	defer s.sessionsMutex.RUnlock()

	activeSessions := make([]*protocol.Session, 0, len(s.sessions))
	for _, session := range s.sessions {
		if session.IsActive() {
			activeSessions = append(activeSessions, session)
		}
	}

	return activeSessions
}

// CloseAllSessions closes all active sessions
func (s *Server) CloseAllSessions() {
	s.sessionsMutex.Lock()
	defer s.sessionsMutex.Unlock()

	for id, session := range s.sessions {
		if session.IsActive() {
			session.Close()
		}
		delete(s.sessions, id)
	}
}

// Shutdown closes the server
func (s *Server) Shutdown() {
	s.CloseAllSessions()
}

// Start starts the server on the configured transports
func (s *Server) Start(ctx context.Context) error {
	if s.logger != nil {
		logging.Info(s.logger, "Starting MCP server",
			slog.String("id", s.ID),
			slog.String("name", s.Name),
			slog.String("version", s.Version))
		logging.Info(s.logger, "Supported transports",
			slog.Any("transports", s.GetSupportedTransports()))
	}

	for _, transportType := range s.GetSupportedTransports() {
		if err := s.startTransport(ctx, transportType); err != nil {
			return fmt.Errorf("failed to start transport %s: %w", transportType, err)
		}
	}

	return nil
}

// StartWithOptions starts the server using the specified options for each transport
func (s *Server) StartWithOptions(ctx context.Context, transportOptions map[string]map[string]interface{}) error {
	if s.logger != nil {
		logging.Info(s.logger, "Starting MCP server with custom options",
			slog.String("id", s.ID),
			slog.String("name", s.Name),
			slog.String("version", s.Version))
	}

	for transportType, options := range transportOptions {
		if err := s.startTransportWithOptions(ctx, transportType, options); err != nil {
			return fmt.Errorf("failed to start transport %s: %w", transportType, err)
		}
	}

	return nil
}

// startTransport starts a specific transport with default options
func (s *Server) startTransport(ctx context.Context, transportType string) error {
	options := make(map[string]interface{})

	// Configure default options based on transport type
	switch transportType {
	case protocol.TransportTypeStdio:
		// No special options required for stdio
	case protocol.TransportTypeHTTP:
		// Default options for HTTP (port 8080)
		options["listenAddress"] = ":8080"
	}

	return s.startTransportWithOptions(ctx, transportType, options)
}

// startTransportWithOptions starts a specific transport with the provided options
func (s *Server) startTransportWithOptions(ctx context.Context, transportType string, options map[string]interface{}) error {
	if s.logger != nil {
		logging.Info(s.logger, "Starting transport",
			slog.String("type", transportType),
			slog.Any("options", options))
	}

	transport, err := s.CreateTransport(ctx, transportType, options)
	if err != nil {
		return err
	}

	// For HTTP transports, start the server
	if transportType == protocol.TransportTypeHTTP {
		if httpTransport, ok := transport.(*httptransport.HTTPTransport); ok {
			// Configure an HTTP handler that manages incoming connections
			_ = httptransport.NewHTTPHandler(func(data []byte) []byte {
				// Create a new connection for each request
				s.HandleConnection(httpTransport)
				return []byte(`{"result":"ok"}`)
			})

			if err := httpTransport.StartServer(); err != nil {
				return err
			}
		}
	} else if transportType == protocol.TransportTypeStdio {
		// For STDIO, start the transport and begin handling connections
		if stdioTransport, ok := transport.(*stdiotransport.STDIOTransport); ok {
			stdioTransport.Start()
			go s.HandleConnection(stdioTransport)
		}
	}

	return nil
}

// CreateTransport creates a new transport instance with the specified type and options
func (s *Server) CreateTransport(ctx context.Context, transportType string, options map[string]interface{}) (protocol.Transport, error) {
	return s.transportRegistry.Create(ctx, transportType, options)
}

// GetSupportedTransports returns a list of all transport types supported by this server
func (s *Server) GetSupportedTransports() []string {
	return s.transportRegistry.GetSupportedTransports()
}

// GetCapabilities generates a map of server capabilities to be sent to the client
// during the initialization process
func (s *Server) GetCapabilities(ctx context.Context) (map[string]protocol.CapabilityDefinition, error) {
	capabilities := make(map[string]protocol.CapabilityDefinition)

	for _, cap := range s.capabilityRegistry.GetCapabilities() {
		capType := cap.GetType()

		// Get the raw options JSON
		rawOptions := cap.GetOptions()

		// Create the capability definition with direct options
		capabilities[string(capType)] = protocol.CapabilityDefinition{
			Options: rawOptions,
		}
	}

	return capabilities, nil
}

func (s *Server) GetCapability(capType capability.CapabilityType) (capability.Capability, error) {
	cap := s.capabilityRegistry.GetCapability(capType)
	if cap == nil {
		return nil, fmt.Errorf("capability %s not found", capType)
	}
	return cap, nil
}

func (s *Server) AddTool(tool *tools.ToolWithHandler) error {
	cap := s.capabilityRegistry.GetCapability(capability.Tools)
	if cap == nil {
		return fmt.Errorf("tools capability not found")
	}
	if toolsCap, ok := cap.(*tools.ToolsCapability); ok {
		toolsCap.RegisterTool(tool)
	} else {
		return fmt.Errorf("failed to cast capability to ToolsCapability")
	}
	return nil
}
