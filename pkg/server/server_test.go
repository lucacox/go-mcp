package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/lucacox/go-mcp/pkg/capabilities/tools"
	"github.com/lucacox/go-mcp/pkg/capability"
	"github.com/lucacox/go-mcp/pkg/protocol"
)

// Aliases for server options to match test usage
var (
	WithID      = WithServerID
	WithName    = WithServerName
	WithVersion = WithServerVersion
	// WithLogger is already properly named in the original file
)

// Helper functions that are missing in the protocol package
func contextWithSessionID(ctx context.Context, sessionID string) context.Context {
	return protocol.WithSessionID(ctx, sessionID)
}

func getSessionIDFromContext(ctx context.Context) (string, bool) {
	return protocol.GetSessionID(ctx)
}

// MockTransport is a mock implementation of protocol.Transport for testing
type MockTransport struct {
	mock.Mock
}

func (m *MockTransport) Send(ctx context.Context, data []byte) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *MockTransport) Receive(ctx context.Context) ([]byte, error) {
	args := m.Called(ctx)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockTransport) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Additional methods that might be required by the protocol.Transport interface
func (m *MockTransport) Start() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockTransport) StartServer() error {
	args := m.Called()
	return args.Error(0)
}

// MockCapability is a mock implementation of capability.Capability for testing
type MockCapability struct {
	mock.Mock
}

func (m *MockCapability) GetType() capability.CapabilityType {
	args := m.Called()
	return args.Get(0).(capability.CapabilityType)
}

func (m *MockCapability) GetOptions() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

func (m *MockCapability) Initialize(ctx context.Context, options map[string]interface{}) error {
	args := m.Called(ctx, options)
	return args.Error(0)
}

// TestNewServer tests the NewServer function with all available options
func TestNewServer(t *testing.T) {
	// Basic server creation without options
	server := NewServer(WithLogger(slog.LevelDebug))
	assert.NotNil(t, server)
	assert.NotEmpty(t, server.ID)
	assert.Equal(t, "MCP Server", server.Name)
	assert.Equal(t, "1.0.0", server.Version)
	assert.NotEmpty(t, server.SupportedVersions)
	assert.NotNil(t, server.endpointRegistry)
	assert.NotNil(t, server.capabilityRegistry)
	assert.NotNil(t, server.transportRegistry)
	assert.NotNil(t, server.sessions)

	// Test all server options
	t.Run("WithServerID", func(t *testing.T) {
		customID := "custom-server-id"
		server := NewServer(WithServerID(customID))
		assert.Equal(t, customID, server.ID)
	})

	t.Run("WithServerName", func(t *testing.T) {
		customName := "Custom Server"
		server := NewServer(WithServerName(customName))
		assert.Equal(t, customName, server.Name)
	})

	t.Run("WithServerVersion", func(t *testing.T) {
		customVersion := "2.0.0"
		server := NewServer(WithServerVersion(customVersion))
		assert.Equal(t, customVersion, server.Version)
	})

	t.Run("WithServerInfo", func(t *testing.T) {
		customInfo := map[string]string{
			"vendor": "Test Vendor",
			"email":  "test@example.com",
		}
		server := NewServer(WithServerInfo(customInfo))
		for k, v := range customInfo {
			assert.Equal(t, v, server.Info[k])
		}
	})

	t.Run("WithLogger", func(t *testing.T) {
		server := NewServer(WithLogger(slog.LevelDebug))
		// Verify logger is configured
		assert.NotNil(t, server.logger)
		assert.NotNil(t, server.loggerFactory)
	})

	t.Run("WithTransportRegistry", func(t *testing.T) {
		customRegistry := protocol.NewTransportRegistry()
		server := NewServer(WithTransportRegistry(customRegistry))
		assert.Equal(t, customRegistry, server.transportRegistry)
	})

	t.Run("WithTransports", func(t *testing.T) {
		transports := []string{protocol.TransportTypeStdio, protocol.TransportTypeHTTP}
		server := NewServer(WithTransports(transports...))

		supportedTypes := server.GetSupportedTransports()
		for _, transportType := range transports {
			assert.Contains(t, supportedTypes, transportType)
		}
	})

	t.Run("WithProtocolVersion", func(t *testing.T) {
		version := protocol.ProtocolVersion20250326
		server := NewServer(WithProtocolVersion(version))
		assert.Contains(t, server.SupportedVersions, version)
	})

	t.Run("WithCapability", func(t *testing.T) {
		server := NewServer(WithCapability(capability.Resources))
		caps, err := server.GetCapabilities(context.Background())
		assert.NoError(t, err)
		assert.Contains(t, caps, string(capability.Resources))
	})

	t.Run("WithTool", func(t *testing.T) {
		// Create a test tool
		toolWithHandler := tools.NewTool("test_tool", "Test Tool", protocol.ObjectSchema(nil, nil), nil, func(ctx context.Context, arguments json.RawMessage) (*tools.ToolResult, error) {
			return tools.NewToolResult([]tools.ContentItem{
				tools.NewTextContent("ciao"),
			}, false), nil
		})

		// Add tool to server
		server := NewServer(WithTool(toolWithHandler))

		// Verify tool is registered
		toolsCap, err := server.GetCapability(capability.Tools)
		assert.NoError(t, err)
		assert.NotNil(t, toolsCap)

		// Check if the tool is properly registered and can be invoked
		if toolsCapability, ok := toolsCap.(*tools.ToolsCapability); ok {
			registeredTools := toolsCapability.ListTools()
			assert.Contains(t, registeredTools, toolWithHandler)
		}
	})

	t.Run("MultipleOptions", func(t *testing.T) {
		customID := "multi-option-server"
		customName := "Multi Option Server"
		customVersion := "3.0.0"
		customInfo := map[string]string{"env": "test"}

		server := NewServer(
			WithServerID(customID),
			WithServerName(customName),
			WithServerVersion(customVersion),
			WithServerInfo(customInfo),
			WithLogger(slog.LevelInfo),
			WithProtocolVersion(protocol.ProtocolVersion20250326),
			WithCapability(capability.Roots),
			WithCapability(capability.Tools),
		)

		// Verify all options were applied correctly
		assert.Equal(t, customID, server.ID)
		assert.Equal(t, customName, server.Name)
		assert.Equal(t, customVersion, server.Version)
		assert.Equal(t, "test", server.Info["env"])
		assert.NotNil(t, server.logger)
		assert.Contains(t, server.SupportedVersions, protocol.ProtocolVersion20250326)

		caps, err := server.GetCapabilities(context.Background())
		assert.NoError(t, err)
		assert.Contains(t, caps, string(capability.Roots))
		assert.Contains(t, caps, string(capability.Tools))
	})
}

// TestServer_RegisterEndpoint tests endpoint registration
func TestServer_RegisterEndpoint(t *testing.T) {
	server := NewServer(WithLogger(slog.LevelDebug))

	// Create a mock endpoint
	mockEndpoint := protocol.NewBaseEndpoint("test")
	mockEndpoint.RegisterMethod("test_method", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		return "test_result", nil
	})

	// Register the endpoint
	server.RegisterEndpoint(mockEndpoint)

	// Test handling a request to the registered endpoint
	result, err := server.HandleRequest(context.Background(), "test/test_method", json.RawMessage(`{}`))
	assert.NoError(t, err)
	assert.Equal(t, "test_result", result)
}

// TestServer_HandleInitialize tests the initialize request handling
func TestServer_HandleInitialize(t *testing.T) {
	server := NewServer(
		WithLogger(slog.LevelDebug),
	)

	// Create test session
	sessionID := uuid.New().String()
	dispatcher := protocol.NewJSONRPCDispatcher(&MockTransport{}, server)
	session := protocol.NewSession(dispatcher)
	session.ID = sessionID

	// Store session in server
	server.sessionsMutex.Lock()
	server.sessions[sessionID] = session
	server.sessionsMutex.Unlock()

	// Create initialize params
	initParams := protocol.InitializeParams{
		ClientID:        "test-client",
		ClientInfo:      map[string]string{"name": "Test Client"},
		ProtocolVersion: string(protocol.ProtocolVersion20250326),
	}

	paramsJSON, err := json.Marshal(initParams)
	require.NoError(t, err)

	// Create context with session ID
	ctx := contextWithSessionID(context.Background(), sessionID)

	// Handle initialize request
	result, err := server.handleInitialize(ctx, paramsJSON)

	// Verify result
	assert.NoError(t, err)
	initResult, ok := result.(*protocol.InitializeResult)
	assert.True(t, ok)
	assert.NotNil(t, initResult.Capabilities)
}

// TestServer_HandlePing tests the ping request handling
func TestServer_HandlePing(t *testing.T) {
	server := NewServer(WithLogger(slog.LevelDebug))

	// Handle ping request
	result, err := server.handlePing(context.Background(), json.RawMessage(`{}`))

	// Verify result
	assert.NoError(t, err)
	assert.Nil(t, result)
}

// TestServer_HandleInitialized tests the initialized notification handling
func TestServer_HandleInitialized(t *testing.T) {
	server := NewServer(WithLogger(slog.LevelDebug))

	// Create test session
	sessionID := uuid.New().String()
	dispatcher := protocol.NewJSONRPCDispatcher(&MockTransport{}, server)
	session := protocol.NewSession(dispatcher)
	session.ID = sessionID
	session.State = protocol.SessionStateInitializing

	// Store session in server
	server.sessionsMutex.Lock()
	server.sessions[sessionID] = session
	server.sessionsMutex.Unlock()

	// Create context with session ID
	ctx := contextWithSessionID(context.Background(), sessionID)

	// Handle initialized notification
	result, err := server.handleInitialized(ctx, json.RawMessage(`{}`))

	// Verify result
	assert.NoError(t, err)
	assert.Nil(t, result)
	assert.Equal(t, protocol.SessionStateActive, session.State)
}

// TestServer_GetCapabilities tests capability retrieval
func TestServer_GetCapabilities(t *testing.T) {
	server := NewServer(WithCapability(capability.Prompts), WithCapability(capability.Resources), WithCapability(capability.Tools))

	// Get capabilities
	capabilities, err := server.GetCapabilities(context.Background())

	// Verify capabilities
	assert.NoError(t, err)
	assert.NotNil(t, capabilities)

	// Check for default capabilities
	assert.Contains(t, capabilities, string(capability.Prompts))
	assert.Contains(t, capabilities, string(capability.Resources))
	assert.Contains(t, capabilities, string(capability.Tools))
}

// TestServer_GetCapability tests retrieving a specific capability
func TestServer_GetCapability(t *testing.T) {
	server := NewServer(WithCapability(capability.Tools))

	// Get a valid capability
	cap, err := server.GetCapability(capability.Tools)
	assert.NoError(t, err)
	assert.NotNil(t, cap)

	// Try to get an invalid capability
	invalidCapType := capability.CapabilityType("invalid_capability")
	cap, err = server.GetCapability(invalidCapType)
	assert.Error(t, err)
	assert.Nil(t, cap)
	assert.Contains(t, err.Error(), fmt.Sprintf("capability %s not found", invalidCapType))
}

// TestServer_GetSession tests session retrieval
func TestServer_GetSession(t *testing.T) {
	server := NewServer(WithLogger(slog.LevelDebug))

	// Create test session
	sessionID := uuid.New().String()
	dispatcher := protocol.NewJSONRPCDispatcher(&MockTransport{}, server)
	session := protocol.NewSession(dispatcher)
	session.ID = sessionID

	// Store session in server
	server.sessionsMutex.Lock()
	server.sessions[sessionID] = session
	server.sessionsMutex.Unlock()

	// Get valid session
	retrievedSession, exists := server.GetSession(sessionID)
	assert.True(t, exists)
	assert.Equal(t, session, retrievedSession)

	// Try to get invalid session
	retrievedSession, exists = server.GetSession("invalid_id")
	assert.False(t, exists)
	assert.Nil(t, retrievedSession)
}

// TestServer_GetActiveSessions tests active session retrieval
func TestServer_GetActiveSessions(t *testing.T) {
	server := NewServer(WithLogger(slog.LevelDebug))

	// Create active session
	activeSessionID := uuid.New().String()
	activeDispatcher := protocol.NewJSONRPCDispatcher(&MockTransport{}, server)
	activeSession := protocol.NewSession(activeDispatcher)
	activeSession.ID = activeSessionID
	activeSession.State = protocol.SessionStateActive

	// Create inactive session
	inactiveSessionID := uuid.New().String()
	inactiveDispatcher := protocol.NewJSONRPCDispatcher(&MockTransport{}, server)
	inactiveSession := protocol.NewSession(inactiveDispatcher)
	inactiveSession.ID = inactiveSessionID
	inactiveSession.State = protocol.SessionStateInitializing

	// Store sessions in server
	server.sessionsMutex.Lock()
	server.sessions[activeSessionID] = activeSession
	server.sessions[inactiveSessionID] = inactiveSession
	server.sessionsMutex.Unlock()

	// Get active sessions
	activeSessions := server.GetActiveSessions()

	// Verify active sessions
	assert.Len(t, activeSessions, 1)
	assert.Equal(t, activeSession, activeSessions[0])
}

// TestServer_CloseAllSessions tests closing all sessions
func TestServer_CloseAllSessions(t *testing.T) {
	server := NewServer(WithLogger(slog.LevelDebug))

	// Create mock transport
	mockTransport := new(MockTransport)
	mockTransport.On("Close").Return(nil)

	// Create session
	sessionID := uuid.New().String()
	dispatcher := protocol.NewJSONRPCDispatcher(mockTransport, server)
	session := protocol.NewSession(dispatcher)
	session.ID = sessionID
	session.State = protocol.SessionStateActive

	// Store session in server
	server.sessionsMutex.Lock()
	server.sessions[sessionID] = session
	server.sessionsMutex.Unlock()

	// Close all sessions
	server.CloseAllSessions()

	// Verify sessions are closed and removed
	assert.Empty(t, server.sessions)
}

// TestServer_Shutdown tests server shutdown
func TestServer_Shutdown(t *testing.T) {
	server := NewServer(WithLogger(slog.LevelDebug))

	// Create mock transport
	mockTransport := new(MockTransport)
	mockTransport.On("Close").Return(nil)

	// Create session
	sessionID := uuid.New().String()
	dispatcher := protocol.NewJSONRPCDispatcher(mockTransport, server)
	session := protocol.NewSession(dispatcher)
	session.ID = sessionID
	session.State = protocol.SessionStateActive

	// Store session in server
	server.sessionsMutex.Lock()
	server.sessions[sessionID] = session
	server.sessionsMutex.Unlock()

	// Shutdown server
	server.Shutdown()

	// Verify sessions are closed
	assert.Empty(t, server.sessions)
}

// TestServer_CreateTransport tests transport creation
func TestServer_CreateTransport(t *testing.T) {
	// Skip this test for now as it would require mocking the transport registry
	t.Skip("Requires mocking the transport registry")
}

// TestServer_GetSupportedTransports tests retrieval of supported transports
func TestServer_GetSupportedTransports(t *testing.T) {
	server := NewServer(WithTransports(protocol.TransportTypeStdio, protocol.TransportTypeHTTP))

	// Get supported transports
	transports := server.GetSupportedTransports()

	// Verify supported transports (assuming default registry has some transports)
	assert.NotEmpty(t, transports)
}

// TestServer_HandleConnection tests connection handling
func TestServer_HandleConnection(t *testing.T) {
	server := NewServer(WithLogger(slog.LevelDebug))

	// Create mock transport
	mockTransport := new(MockTransport)
	mockTransport.On("Receive", mock.Anything).Return([]byte(`{"jsonrpc": "2.0", "method": "ping", "id": 1}`), nil).Maybe()
	mockTransport.On("Send", mock.Anything, mock.Anything).Return(nil).Maybe()
	mockTransport.On("Close").Return(nil).Maybe()

	// Handle connection
	server.HandleConnection(mockTransport)

	// Verify that a new session was created
	assert.Equal(t, 1, len(server.sessions))

	// Get the created session
	var session *protocol.Session
	for _, s := range server.sessions {
		session = s
		break
	}

	// Verify session properties
	assert.NotNil(t, session)
	assert.Equal(t, server.ID, session.ServerID)
	assert.Equal(t, server.Info, session.ServerInfo)
}

// TestServer_Start tests server startup with default options
func TestServer_Start(t *testing.T) {
	// Mock the transport registry to avoid actual network operations
	origRegistry := protocol.DefaultTransportRegistry
	mockRegistry := protocol.NewTransportRegistry()
	protocol.DefaultTransportRegistry = mockRegistry

	// Restore the original registry after test
	defer func() {
		protocol.DefaultTransportRegistry = origRegistry
	}()

	// Register a mock transport creator
	mockRegistry.Register(protocol.TransportTypeStdio, func(ctx context.Context, options map[string]interface{}) (protocol.Transport, error) {
		mockTransport := new(MockTransport)
		mockTransport.On("Start").Return(nil)
		mockTransport.On("Receive", mock.Anything).Return([]byte(`{"jsonrpc": "2.0", "method": "ping", "id": 1}`), nil).Maybe()
		mockTransport.On("Send", mock.Anything, mock.Anything).Return(nil).Maybe()
		mockTransport.On("Close").Return(nil).Maybe()
		return mockTransport, nil
	})

	// Create server
	server := NewServer(WithLogger(slog.LevelDebug))

	// Start server
	err := server.Start(context.Background())
	assert.NoError(t, err)
}

// TestServer_StartWithOptions tests server startup with custom options
func TestServer_StartWithOptions(t *testing.T) {
	// Mock the transport registry to avoid actual network operations
	origRegistry := protocol.DefaultTransportRegistry
	mockRegistry := protocol.NewTransportRegistry()
	protocol.DefaultTransportRegistry = mockRegistry

	// Restore the original registry after test
	defer func() {
		protocol.DefaultTransportRegistry = origRegistry
	}()

	// Register mock transport creators
	mockRegistry.Register(protocol.TransportTypeStdio, func(ctx context.Context, options map[string]interface{}) (protocol.Transport, error) {
		mockTransport := new(MockTransport)
		mockTransport.On("Start").Return(nil)
		mockTransport.On("Receive", mock.Anything).Return([]byte(`{"jsonrpc": "2.0", "method": "ping", "id": 1}`), nil).Maybe()
		mockTransport.On("Send", mock.Anything, mock.Anything).Return(nil).Maybe()
		mockTransport.On("Close").Return(nil).Maybe()
		return mockTransport, nil
	})

	mockRegistry.Register(protocol.TransportTypeHTTP, func(ctx context.Context, options map[string]interface{}) (protocol.Transport, error) {
		// Verify custom options are passed
		listenAddress, ok := options["listenAddress"]
		assert.True(t, ok)
		assert.Equal(t, ":9000", listenAddress)

		mockTransport := new(MockTransport)
		mockTransport.On("StartServer").Return(nil)
		mockTransport.On("Receive", mock.Anything).Return([]byte(`{"jsonrpc": "2.0", "method": "ping", "id": 1}`), nil).Maybe()
		mockTransport.On("Send", mock.Anything, mock.Anything).Return(nil).Maybe()
		mockTransport.On("Close").Return(nil).Maybe()
		return mockTransport, nil
	})

	// Create server
	server := NewServer(WithLogger(slog.LevelDebug))

	// Custom options for transports
	transportOptions := map[string]map[string]interface{}{
		protocol.TransportTypeStdio: {},
		protocol.TransportTypeHTTP: {
			"listenAddress": ":9000",
		},
	}

	// Start server with options
	err := server.StartWithOptions(context.Background(), transportOptions)
	assert.NoError(t, err)
}

// TestServer_Error_Handling tests error handling in various server methods
func TestServer_Error_Handling(t *testing.T) {
	t.Run("TestHandleInitializeWithInvalidParams", func(t *testing.T) {
		server := NewServer(WithLogger(slog.LevelDebug))

		// Create test session
		sessionID := uuid.New().String()
		dispatcher := protocol.NewJSONRPCDispatcher(&MockTransport{}, server)
		session := protocol.NewSession(dispatcher)
		session.ID = sessionID

		// Store session in server
		server.sessionsMutex.Lock()
		server.sessions[sessionID] = session
		server.sessionsMutex.Unlock()

		// Create context with session ID
		ctx := contextWithSessionID(context.Background(), sessionID)

		// Handle initialize request with invalid params
		result, err := server.handleInitialize(ctx, json.RawMessage(`{invalid_json`))

		// Verify error
		assert.Error(t, err)
		assert.Nil(t, result)
		jsonRPCErr, ok := err.(*protocol.JSONRPCError)
		assert.True(t, ok)
		assert.Equal(t, -32700, jsonRPCErr.Code) // Parse error code
	})

	t.Run("TestHandleInitializeWithMissingSession", func(t *testing.T) {
		server := NewServer(WithLogger(slog.LevelDebug))

		// Create context with non-existent session ID
		ctx := contextWithSessionID(context.Background(), "non-existent-session")

		// Handle initialize request
		result, err := server.handleInitialize(ctx, json.RawMessage(`{}`))

		// Verify error
		assert.Error(t, err)
		assert.Nil(t, result)
		jsonRPCErr, ok := err.(*protocol.JSONRPCError)
		assert.True(t, ok)
		assert.Equal(t, -32603, jsonRPCErr.Code) // Internal error code
	})

	t.Run("TestHandleInitializedWithInvalidSessionState", func(t *testing.T) {
		server := NewServer(WithLogger(slog.LevelDebug))

		// Create test session with an invalid state for initialized notification
		sessionID := uuid.New().String()
		dispatcher := protocol.NewJSONRPCDispatcher(&MockTransport{}, server)
		session := protocol.NewSession(dispatcher)
		session.ID = sessionID
		session.State = protocol.SessionStateActive // Already active, should be initializing

		// Store session in server
		server.sessionsMutex.Lock()
		server.sessions[sessionID] = session
		server.sessionsMutex.Unlock()

		// Create context with session ID
		ctx := contextWithSessionID(context.Background(), sessionID)

		// Handle initialized notification
		result, err := server.handleInitialized(ctx, json.RawMessage(`{}`))

		// Verify error
		assert.Error(t, err)
		assert.Nil(t, result)
		jsonRPCErr, ok := err.(*protocol.JSONRPCError)
		assert.True(t, ok)
		assert.Equal(t, -32603, jsonRPCErr.Code) // Internal error code
	})
}
