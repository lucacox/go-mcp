package protocol

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestSessionContext tests the context utilities for session IDs
func TestSessionContext(t *testing.T) {
	// Test adding session ID to context
	sessionID := "test-session-id"
	ctx := context.Background()

	// Add session ID to context
	ctx = WithSessionID(ctx, sessionID)

	// Retrieve session ID from context
	retrievedID, ok := GetSessionID(ctx)
	assert.True(t, ok)
	assert.Equal(t, sessionID, retrievedID)

	// Test with empty context
	emptyCtx := context.Background()
	retrievedID, ok = GetSessionID(emptyCtx)
	assert.False(t, ok)
	assert.Empty(t, retrievedID)
}

// TestSessionState tests the SessionState string representation
func TestSessionState(t *testing.T) {
	tests := []struct {
		state    SessionState
		expected string
	}{
		{SessionStateUninitialized, "uninitialized"},
		{SessionStateInitializing, "initializing"},
		{SessionStateActive, "active"},
		{SessionStateClosing, "closing"},
		{SessionStateClosed, "closed"},
		{SessionState(999), "unknown"}, // Unknown state
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, test.state.String())
	}
}

// TestNewSession tests the creation of a new session
func TestNewSession(t *testing.T) {
	// Create mock dispatcher
	mockDispatcher := &JSONRPCDispatcher{
		transport: NewMockTransport(10),
		handler:   new(MockRPCHandler),
		pending:   make(map[string]chan *JSONRPCMessage),
		shutdown:  make(chan struct{}),
	}

	// Create a new session
	session := NewSession(mockDispatcher)

	// Verify session properties
	assert.NotEmpty(t, session.ID)
	assert.Equal(t, SessionStateUninitialized, session.State)
	assert.Equal(t, mockDispatcher, session.dispatcher)
	assert.NotZero(t, session.CreatedAt)
	assert.NotZero(t, session.LastActiveAt)
	assert.NotNil(t, session.ClientCapabilities)
	assert.NotNil(t, session.ServerCapabilities)
	assert.NotNil(t, session.ClientInfo)
	assert.NotNil(t, session.ServerInfo)
}

// TestSession_GetState tests getting the session state
func TestSession_GetState(t *testing.T) {
	// Create mock dispatcher
	mockDispatcher := &JSONRPCDispatcher{
		transport: NewMockTransport(10),
		handler:   new(MockRPCHandler),
		pending:   make(map[string]chan *JSONRPCMessage),
		shutdown:  make(chan struct{}),
	}

	// Create a new session
	session := NewSession(mockDispatcher)

	// Set a specific state
	session.State = SessionStateInitializing

	// Get the state
	state := session.GetState()

	// Verify the state
	assert.Equal(t, SessionStateInitializing, state)
}

// TestSession_SetState tests setting the session state
func TestSession_SetState(t *testing.T) {
	// Create mock dispatcher
	mockDispatcher := &JSONRPCDispatcher{
		transport: NewMockTransport(10),
		handler:   new(MockRPCHandler),
		pending:   make(map[string]chan *JSONRPCMessage),
		shutdown:  make(chan struct{}),
	}

	// Create a new session
	session := NewSession(mockDispatcher)

	// Set the state
	session.SetState(SessionStateActive)

	// Verify the state was set
	assert.Equal(t, SessionStateActive, session.State)
}

// TestSession_UpdateLastActiveTime tests updating the last active time
func TestSession_UpdateLastActiveTime(t *testing.T) {
	// Create mock dispatcher
	mockDispatcher := &JSONRPCDispatcher{
		transport: NewMockTransport(10),
		handler:   new(MockRPCHandler),
		pending:   make(map[string]chan *JSONRPCMessage),
		shutdown:  make(chan struct{}),
	}

	// Create a new session
	session := NewSession(mockDispatcher)

	// Store the initial last active time
	initialLastActive := session.LastActiveAt

	// Wait a short period to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Update the last active time
	session.UpdateLastActiveTime()

	// Verify the last active time was updated
	assert.True(t, session.LastActiveAt.After(initialLastActive))
}

// TestSession_Initialize tests session initialization
func TestSession_Initialize(t *testing.T) {
	// Create mock dispatcher
	mockDispatcher := &JSONRPCDispatcher{
		transport: NewMockTransport(10),
		handler:   new(MockRPCHandler),
		pending:   make(map[string]chan *JSONRPCMessage),
		shutdown:  make(chan struct{}),
	}

	// Create a new session
	session := NewSession(mockDispatcher)

	// Set server properties (normally these would be set by the server)
	session.ServerID = "test-server-id"
	session.ServerInfo = map[string]string{"name": "Test Server"}
	session.ServerCapabilities = map[string]CapabilityDefinition{
		"tools": {
			Options: json.RawMessage(`{"list_changed":true}`),
		},
	}

	// Create initialization parameters
	initParams := &InitializeParams{
		ProtocolVersion: "2025-03-26",
		ClientID:        "test-client-id",
		ClientInfo:      map[string]string{"name": "Test Client"},
		Capabilities: map[string]CapabilityDefinition{
			"tools": {
				Options: json.RawMessage(`{"use_list_changed":true}`),
			},
		},
	}

	// Initialize the session
	result, err := session.Initialize(context.Background(), initParams, ProtocolVersion20250326)

	// Verify no error
	assert.NoError(t, err)

	// Verify result
	assert.NotNil(t, result)
	assert.Equal(t, string(ProtocolVersion20250326), result.ProtocolVersion)
	assert.Equal(t, session.ServerID, result.ServerID)
	assert.Equal(t, session.ServerInfo, result.ServerInfo)
	assert.Equal(t, session.ServerCapabilities, result.Capabilities)

	// Verify session state
	assert.Equal(t, SessionStateInitializing, session.State)

	// Verify client information was stored in session
	assert.Equal(t, initParams.ClientID, session.ClientID)
	assert.Equal(t, initParams.ClientInfo, session.ClientInfo)
	assert.Equal(t, initParams.Capabilities, session.ClientCapabilities)
}

// TestSession_Initialize_AlreadyInitialized tests initialization of an already initialized session
func TestSession_Initialize_AlreadyInitialized(t *testing.T) {
	// Create mock dispatcher
	mockDispatcher := &JSONRPCDispatcher{
		transport: NewMockTransport(10),
		handler:   new(MockRPCHandler),
		pending:   make(map[string]chan *JSONRPCMessage),
		shutdown:  make(chan struct{}),
	}

	// Create a new session
	session := NewSession(mockDispatcher)

	// Set the state to active (not uninitialized)
	session.State = SessionStateActive

	// Create initialization parameters
	initParams := &InitializeParams{
		ProtocolVersion: "2025-03-26",
		ClientID:        "test-client-id",
		ClientInfo:      map[string]string{"name": "Test Client"},
		Capabilities:    map[string]CapabilityDefinition{},
	}

	// Try to initialize the session
	result, err := session.Initialize(context.Background(), initParams, ProtocolVersion20250326)

	// Verify error
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "already initialized")

	// Verify session state was not changed
	assert.Equal(t, SessionStateActive, session.State)
}

// TestSession_Close tests session closing
func TestSession_Close(t *testing.T) {
	// Create mock dispatcher
	mockDispatcher := &JSONRPCDispatcher{
		transport: NewMockTransport(10),
		handler:   new(MockRPCHandler),
		pending:   make(map[string]chan *JSONRPCMessage),
		shutdown:  make(chan struct{}),
	}

	// Create a new session
	session := NewSession(mockDispatcher)

	// Set the state to active
	session.State = SessionStateActive

	// Close the session
	err := session.Close()

	// Verify no error
	assert.NoError(t, err)

	// Verify session state
	assert.Equal(t, SessionStateClosed, session.State)

	// Test closing an already closed session
	err = session.Close()
	assert.NoError(t, err) // Should be idempotent
}

// TestSession_IsActive tests checking if a session is active
func TestSession_IsActive(t *testing.T) {
	// Create mock dispatcher
	mockDispatcher := &JSONRPCDispatcher{
		transport: NewMockTransport(10),
		handler:   new(MockRPCHandler),
		pending:   make(map[string]chan *JSONRPCMessage),
		shutdown:  make(chan struct{}),
	}

	// Create a new session
	session := NewSession(mockDispatcher)

	// Test with different states
	testCases := []struct {
		state    SessionState
		isActive bool
	}{
		{SessionStateUninitialized, false},
		{SessionStateInitializing, false},
		{SessionStateActive, true},
		{SessionStateClosing, false},
		{SessionStateClosed, false},
	}

	for _, tc := range testCases {
		session.State = tc.state
		assert.Equal(t, tc.isActive, session.IsActive())
	}
}

// TestSession_HasCapability tests checking if a capability is supported
func TestSession_HasCapability(t *testing.T) {
	// Create mock dispatcher
	mockDispatcher := &JSONRPCDispatcher{
		transport: NewMockTransport(10),
		handler:   new(MockRPCHandler),
		pending:   make(map[string]chan *JSONRPCMessage),
		shutdown:  make(chan struct{}),
	}

	// Create a new session
	session := NewSession(mockDispatcher)

	// Add some capabilities
	session.ClientCapabilities = map[string]CapabilityDefinition{
		"tools": {
			Options: json.RawMessage(`{"list_changed":true}`),
		},
		"resources": {
			Options: json.RawMessage(`{}`),
		},
	}

	// Test with various capabilities
	assert.True(t, session.HasCapability("tools"))
	assert.True(t, session.HasCapability("resources"))
	assert.False(t, session.HasCapability("prompts"))
	assert.False(t, session.HasCapability("unknown"))
}

// mockDispatcherForSessionTests creates a mock dispatcher that can be used for session tests
func mockDispatcherForSessionTests(t *testing.T) *JSONRPCDispatcher {
	transport := NewMockTransport(10)
	handler := new(MockRPCHandler)
	return NewJSONRPCDispatcher(transport, handler)
}

// TestSession_Call tests making a call through a session
func TestSession_Call(t *testing.T) {
	mockTransport := NewMockTransport(10)
	handler := new(MockRPCHandler)
	mockDispatcher := NewJSONRPCDispatcher(mockTransport, handler)

	// Configure transport to handle the call
	mockTransport.On("Send", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		// Parse the sent request
		var request JSONRPCMessage
		err := json.Unmarshal(args.Get(1).([]byte), &request)
		require.NoError(t, err)

		// Prepare and send a response
		response := JSONRPCMessage{
			JSONRPC: JSONRPCVersion,
			ID:      request.ID,
			Result:  []byte(`"test result"`),
		}

		responseData, err := json.Marshal(response)
		require.NoError(t, err)

		mockTransport.QueueReceiveData(responseData)
	})

	mockTransport.On("Receive", mock.Anything).Return([]byte{}, nil)

	// Create a new session
	session := NewSession(mockDispatcher)
	mockDispatcher.Start()
	defer mockDispatcher.Stop()

	// Test call with inactive session
	result, err := session.Call(context.Background(), "test/method", map[string]string{"param": "value"})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "session not active")

	// Set session to active
	session.State = SessionStateActive

	// Make a call
	result, err = session.Call(context.Background(), "test/method", map[string]string{"param": "value"})

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, []byte(`"test result"`), []byte(result))
	mockTransport.AssertExpectations(t)
}

// TestSession_Notify tests sending a notification through a session
func TestSession_Notify(t *testing.T) {
	mockTransport := NewMockTransport(10)
	handler := new(MockRPCHandler)
	mockDispatcher := NewJSONRPCDispatcher(mockTransport, handler)

	// Configure transport to expect a notification
	mockTransport.On("Send", mock.Anything, mock.MatchedBy(func(data []byte) bool {
		var notification JSONRPCMessage
		err := json.Unmarshal(data, &notification)
		return err == nil && notification.Method == "notifications/test_method" && notification.ID == nil
	})).Return(nil)

	// Create a new session
	session := NewSession(mockDispatcher)

	// Test notify with inactive session
	err := session.Notify(context.Background(), "notifications/test_method", map[string]string{"event": "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not active")

	// Set session to active
	session.State = SessionStateActive

	// Send a notification
	err = session.Notify(context.Background(), "notifications/test_method", map[string]string{"event": "test"})

	// Verify
	assert.NoError(t, err)
	mockTransport.AssertExpectations(t)

	// Verify last active time was updated (indirectly)
	assert.WithinDuration(t, time.Now(), session.LastActiveAt, time.Second)
}
