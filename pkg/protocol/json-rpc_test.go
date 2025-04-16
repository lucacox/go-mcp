package protocol

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockTransport is a mock implementation of the Transport interface for testing
type MockTransport struct {
	mock.Mock
	receiveCh chan []byte
	mu        sync.Mutex
}

// NewMockTransport creates a new mock transport with buffer capacity
func NewMockTransport(bufferSize int) *MockTransport {
	return &MockTransport{
		receiveCh: make(chan []byte, bufferSize),
	}
}

func (m *MockTransport) Send(ctx context.Context, data []byte) error {
	args := m.Called(ctx, data)
	// fmt.Printf("Send called with args: %v\n", args)
	return args.Error(0)
}

func (m *MockTransport) Receive(ctx context.Context) ([]byte, error) {
	args := m.Called(ctx)
	// fmt.Printf("Receive called with args: %v\n", args)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case data := <-m.receiveCh:
		// fmt.Printf("Received data: %s\n", string(data))
		return data, nil
	}
}

func (m *MockTransport) Close() error {
	args := m.Called()
	close(m.receiveCh)
	return args.Error(0)
}

func (m *MockTransport) QueueReceiveData(data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.receiveCh <- data
}

// MockRPCHandler is a mock implementation of the RPCHandler interface for testing
type MockRPCHandler struct {
	mock.Mock
}

func (m *MockRPCHandler) HandleRequest(ctx context.Context, method string, params json.RawMessage) (interface{}, error) {
	args := m.Called(ctx, method, params)
	return args.Get(0), args.Error(1)
}

// TestJSONRPCError tests the JSONRPCError implementation
func TestJSONRPCError(t *testing.T) {
	err := &JSONRPCError{
		Code:    -32700,
		Message: "Parse error",
		Data:    []byte(`{"details":"Invalid JSON"}`),
	}

	assert.Equal(t, "Parse error", err.Error())

	// Test error creation with NewJSONRPCError
	err2 := NewJSONRPCError(ErrorCodeInvalidRequest, "Invalid request", "Details")
	assert.Equal(t, ErrorCodeInvalidRequest, err2.Code)
	assert.Equal(t, "Invalid request", err2.Message)
	assert.NotNil(t, err2.Data)
}

// TestNewJSONRPCDispatcher tests the creation of a JSON-RPC dispatcher
func TestNewJSONRPCDispatcher(t *testing.T) {
	transport := NewMockTransport(10)
	handler := new(MockRPCHandler)

	dispatcher := NewJSONRPCDispatcher(transport, handler)

	assert.NotNil(t, dispatcher)
	assert.Equal(t, transport, dispatcher.transport)
	assert.Equal(t, handler, dispatcher.handler)
	assert.NotNil(t, dispatcher.pending)
	assert.NotNil(t, dispatcher.shutdown)
}

// TestJSONRPCDispatcher_Call tests the Call method of the JSON-RPC dispatcher
func TestJSONRPCDispatcher_Call(t *testing.T) {
	t.Run("SuccessfulCall", func(t *testing.T) {
		transport := NewMockTransport(10)
		handler := new(MockRPCHandler)

		// Configure mock transport to simulate a response
		transport.On("Send", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			// Extract the sent request
			var request JSONRPCMessage
			err := json.Unmarshal(args.Get(1).([]byte), &request)
			require.NoError(t, err)

			// Create and queue a response
			response := JSONRPCMessage{
				JSONRPC: JSONRPCVersion,
				ID:      request.ID,
				Result:  []byte(`"response result"`),
			}
			responseData, err := json.Marshal(response)
			require.NoError(t, err)
			transport.QueueReceiveData(responseData)
		})

		transport.On("Receive", mock.Anything).Return([]byte{}, nil)

		dispatcher := NewJSONRPCDispatcher(transport, handler)
		dispatcher.Start()

		// Make a call
		ctx, cancel := context.WithCancel(context.Background())
		result, err := dispatcher.Call(ctx, "test_method", map[string]string{"param": "value"})

		// Verify
		assert.NoError(t, err)
		assert.Equal(t, []byte(`"response result"`), []byte(result))
		transport.AssertExpectations(t)
		cancel()
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		transport := NewMockTransport(10)
		handler := new(MockRPCHandler)

		// Configure mock transport to simulate an error response
		transport.On("Send", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			// Extract the sent request
			var request JSONRPCMessage
			err := json.Unmarshal(args.Get(1).([]byte), &request)
			require.NoError(t, err)

			// Create and queue an error response
			response := JSONRPCMessage{
				JSONRPC: JSONRPCVersion,
				ID:      request.ID,
				Error: &JSONRPCError{
					Code:    ErrorCodeMethodNotFound,
					Message: "Method not found",
				},
			}
			responseData, err := json.Marshal(response)
			require.NoError(t, err)
			transport.QueueReceiveData(responseData)
		})

		transport.On("Receive", mock.Anything).Return([]byte{}, nil)

		dispatcher := NewJSONRPCDispatcher(transport, handler)
		dispatcher.Start()
		defer dispatcher.Stop()

		// Make a call
		ctx, cancel := context.WithCancel(context.Background())
		result, err := dispatcher.Call(ctx, "nonexistent_method", nil)

		// Verify
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "Method not found", err.Error())
		transport.AssertExpectations(t)
		cancel()
	})

	t.Run("TransportError", func(t *testing.T) {
		transport := NewMockTransport(10)
		handler := new(MockRPCHandler)

		// Configure mock transport to return an error
		transport.On("Send", mock.Anything, mock.Anything).Return(errors.New("network error"))
		transport.On("Receive", mock.Anything).Return([]byte{}, nil).Maybe()

		dispatcher := NewJSONRPCDispatcher(transport, handler)
		dispatcher.Start()
		defer dispatcher.Stop()

		// Make a call
		ctx, cancel := context.WithCancel(context.Background())
		result, err := dispatcher.Call(ctx, "test_method", nil)

		// Verify
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "error sending request")
		transport.AssertExpectations(t)
		cancel()
		// transport.QueueReceiveData([]byte(`{"jsonrpc":"2.0"}`))
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		transport := NewMockTransport(10)
		handler := new(MockRPCHandler)

		// Configure mock transport
		transport.On("Send", mock.Anything, mock.Anything).Return(nil)
		transport.On("Receive", mock.Anything).Return([]byte{}, nil)

		dispatcher := NewJSONRPCDispatcher(transport, handler)
		dispatcher.Start()
		defer dispatcher.Stop()

		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Make a call that should time out
		result, err := dispatcher.Call(ctx, "test_method", nil)

		// Verify
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		transport.AssertExpectations(t)
	})
}

// TestJSONRPCDispatcher_Notify tests the Notify method of the JSON-RPC dispatcher
func TestJSONRPCDispatcher_Notify(t *testing.T) {
	t.Run("SuccessfulNotification", func(t *testing.T) {
		transport := NewMockTransport(10)
		handler := new(MockRPCHandler)

		// Configure mock transport
		transport.On("Send", mock.Anything, mock.MatchedBy(func(data []byte) bool {
			// Verify the sent notification has no ID
			var notification JSONRPCMessage
			err := json.Unmarshal(data, &notification)
			return err == nil && notification.Method == "test_notification" && notification.ID == nil
		})).Return(nil)

		dispatcher := NewJSONRPCDispatcher(transport, handler)

		// Send notification
		err := dispatcher.Notify(context.Background(), "test_notification", map[string]string{"event": "test"})

		// Verify
		assert.NoError(t, err)
		transport.AssertExpectations(t)
	})

	t.Run("TransportError", func(t *testing.T) {
		transport := NewMockTransport(10)
		handler := new(MockRPCHandler)

		// Configure mock transport to return an error
		transport.On("Send", mock.Anything, mock.Anything).Return(errors.New("network error"))

		dispatcher := NewJSONRPCDispatcher(transport, handler)

		// Send notification
		err := dispatcher.Notify(context.Background(), "test_notification", nil)

		// Verify
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error sending notification")
		transport.AssertExpectations(t)
	})
}

// TestJSONRPCDispatcher_HandleRequest tests the request handling of the JSON-RPC dispatcher
func TestJSONRPCDispatcher_HandleRequest(t *testing.T) {
	t.Run("SuccessfulRequest", func(t *testing.T) {
		transport := NewMockTransport(10)
		handler := new(MockRPCHandler)

		// Configure handler to return a successful result
		handler.On("HandleRequest", mock.Anything, "test_method", mock.Anything).Return("success result", nil)

		// Configure transport to expect a successful response
		transport.On("Send", mock.Anything, mock.MatchedBy(func(data []byte) bool {
			var response JSONRPCMessage
			err := json.Unmarshal(data, &response)
			return err == nil && response.Error == nil && string(response.Result) == `"success result"`
		})).Return(nil)

		dispatcher := NewJSONRPCDispatcher(transport, handler)

		// Simulate receiving a request
		request := JSONRPCMessage{
			JSONRPC: JSONRPCVersion,
			ID:      []byte(`"test-id"`),
			Method:  "test_method",
			Params:  []byte(`{"param":"value"}`),
		}

		ctx := context.Background()
		dispatcher.handleRequest(ctx, &request)

		// Verify
		handler.AssertExpectations(t)
		transport.AssertExpectations(t)
	})

	t.Run("ErrorInHandler", func(t *testing.T) {
		transport := NewMockTransport(10)
		handler := new(MockRPCHandler)

		// Configure handler to return an error
		handler.On("HandleRequest", mock.Anything, "test_method", mock.Anything).Return(nil, errors.New("handler error"))

		// Configure transport to expect an error response
		transport.On("Send", mock.Anything, mock.MatchedBy(func(data []byte) bool {
			var response JSONRPCMessage
			err := json.Unmarshal(data, &response)
			return err == nil && response.Error != nil && response.Error.Message == "handler error"
		})).Return(nil)

		dispatcher := NewJSONRPCDispatcher(transport, handler)

		// Simulate receiving a request
		request := JSONRPCMessage{
			JSONRPC: JSONRPCVersion,
			ID:      []byte(`"test-id"`),
			Method:  "test_method",
			Params:  []byte(`{"param":"value"}`),
		}

		ctx := context.Background()
		dispatcher.handleRequest(ctx, &request)

		// Verify
		handler.AssertExpectations(t)
		transport.AssertExpectations(t)
	})

	t.Run("RpcErrorInHandler", func(t *testing.T) {
		transport := NewMockTransport(10)
		handler := new(MockRPCHandler)

		// Configure handler to return a specific JSON-RPC error
		jsonRpcErr := &JSONRPCError{
			Code:    ErrorCodeInvalidParams,
			Message: "Invalid parameters",
		}
		handler.On("HandleRequest", mock.Anything, "test_method", mock.Anything).Return(nil, jsonRpcErr)

		// Configure transport to expect a matching error response
		transport.On("Send", mock.Anything, mock.MatchedBy(func(data []byte) bool {
			var response JSONRPCMessage
			err := json.Unmarshal(data, &response)
			return err == nil &&
				response.Error != nil &&
				response.Error.Code == ErrorCodeInvalidParams &&
				response.Error.Message == "Invalid parameters"
		})).Return(nil)

		dispatcher := NewJSONRPCDispatcher(transport, handler)

		// Simulate receiving a request
		request := JSONRPCMessage{
			JSONRPC: JSONRPCVersion,
			ID:      []byte(`"test-id"`),
			Method:  "test_method",
			Params:  []byte(`{"param":"value"}`),
		}

		ctx := context.Background()
		dispatcher.handleRequest(ctx, &request)

		// Verify
		handler.AssertExpectations(t)
		transport.AssertExpectations(t)
	})
}

// TestJSONRPCDispatcher_HandleNotification tests the notification handling of the JSON-RPC dispatcher
func TestJSONRPCDispatcher_HandleNotification(t *testing.T) {
	transport := NewMockTransport(10)
	handler := new(MockRPCHandler)

	// Configure handler to handle notification
	handler.On("HandleRequest", mock.Anything, "test_notification", mock.Anything).Return(nil, nil)

	dispatcher := NewJSONRPCDispatcher(transport, handler)

	// Should not expect any response to be sent for notifications

	// Simulate receiving a notification
	notification := JSONRPCMessage{
		JSONRPC: JSONRPCVersion,
		Method:  "test_notification",
		Params:  []byte(`{"event":"test"}`),
	}

	ctx := context.Background()
	dispatcher.handleNotification(ctx, &notification)

	// Verify
	handler.AssertExpectations(t)
	// No expectations on transport.Send as notifications don't send responses
}

// TestJSONRPCDispatcher_HandleResponse tests the response handling of the JSON-RPC dispatcher
func TestJSONRPCDispatcher_HandleResponse(t *testing.T) {
	transport := NewMockTransport(10)
	handler := new(MockRPCHandler)

	dispatcher := NewJSONRPCDispatcher(transport, handler)

	// Create a pending request channel
	responseID := uuid.New().String()
	responseCh := make(chan *JSONRPCMessage, 1)

	dispatcher.pendingMux.Lock()
	dispatcher.pending[responseID] = responseCh
	dispatcher.pendingMux.Unlock()

	// Simulate receiving a response
	response := &JSONRPCMessage{
		JSONRPC: JSONRPCVersion,
		ID:      []byte(`"` + responseID + `"`),
		Result:  []byte(`"response data"`),
	}

	// Handle the response
	dispatcher.handleResponse(response)

	// Verify that the response was delivered to the channel
	select {
	case received := <-responseCh:
		assert.Equal(t, response, received)
	case <-time.After(time.Second):
		t.Fatal("Timed out waiting for response")
	}

	// Verify the pending request was removed
	dispatcher.pendingMux.Lock()
	_, exists := dispatcher.pending[responseID]
	dispatcher.pendingMux.Unlock()
	assert.False(t, exists, "Pending request should be removed after handling")
}

// TestJSONRPCDispatcher_ReceiveLoop tests the message receiving loop
func TestJSONRPCDispatcher_ReceiveLoop(t *testing.T) {
	// This test is complex due to the goroutine nature of receiveLoop

	transport := NewMockTransport(10)
	handler := new(MockRPCHandler)

	// Configure handler for when requests might be passed to it
	handler.On("HandleRequest", mock.Anything, mock.Anything, mock.Anything).Return("success", nil)

	// Set up the dispatcher
	dispatcher := NewJSONRPCDispatcher(transport, handler)

	// Prepare test messages
	requestJSON := `{"jsonrpc":"2.0","id":"req1","method":"test_method","params":{"key":"value"}}`
	notificationJSON := `{"jsonrpc":"2.0","method":"test_notify","params":{"event":"happened"}}`
	responseJSON := `{"jsonrpc":"2.0","id":"resp1","result":"success"}`

	// Configure transport.Receive to actually use the MockTransport's queue
	// This is critical for the test to work properly
	transport.On("Receive", mock.Anything).Run(func(args mock.Arguments) {
		// Messages will be read from receiveCh when QueueReceiveData is called
	}).Return([]byte{}, nil)

	// Configure transport.Send for response handling
	transport.On("Send", mock.Anything, mock.Anything).Return(nil)

	// Start the receive loop
	dispatcher.Start()

	// Add a fake pending request to handle the response
	responseCh := make(chan *JSONRPCMessage, 1)
	dispatcher.pendingMux.Lock()
	dispatcher.pending["resp1"] = responseCh
	dispatcher.pendingMux.Unlock()

	// Queue messages with delays to ensure they're processed one at a time
	transport.QueueReceiveData([]byte(requestJSON))
	time.Sleep(50 * time.Millisecond)

	transport.QueueReceiveData([]byte(notificationJSON))
	time.Sleep(50 * time.Millisecond)

	transport.QueueReceiveData([]byte(responseJSON))

	// Wait a reasonable time for processing (but not too long)
	time.Sleep(100 * time.Millisecond)

	// Shutdown the dispatcher
	dispatcher.Stop()

	// Verify calls - should have reasonable expectations
	handler.AssertNumberOfCalls(t, "HandleRequest", 2) // One for request and one for notification

	// Verify the response was delivered (processed by handleResponse)
	select {
	case received := <-responseCh:
		assert.Equal(t, "resp1", string(received.ID)[1:len(string(received.ID))-1])
		assert.Equal(t, `"success"`, string(received.Result))
	default:
		t.Fatal("Response not delivered to the pending channel")
	}
}

// TestJSONRPCDispatcher_SetSessionID tests setting the session ID
func TestJSONRPCDispatcher_SetSessionID(t *testing.T) {
	transport := NewMockTransport(10)
	handler := new(MockRPCHandler)

	dispatcher := NewJSONRPCDispatcher(transport, handler)

	// Set session ID
	sessionID := "test-session-123"
	dispatcher.SetSessionID(sessionID)

	// Verify it was set
	assert.Equal(t, sessionID, dispatcher.sessionID)

	// The real test is whether the session ID is properly added to context when handling requests
	// Configure handler to verify context has session ID
	handler.On("HandleRequest", mock.MatchedBy(func(ctx context.Context) bool {
		id, ok := GetSessionID(ctx)
		return ok && id == sessionID
	}), mock.Anything, mock.Anything).Return("success", nil)

	// Configure transport for response
	transport.On("Send", mock.Anything, mock.Anything).Return(nil)

	// Simulate a request
	request := JSONRPCMessage{
		JSONRPC: JSONRPCVersion,
		ID:      []byte(`"test-id"`),
		Method:  "test_method",
		Params:  []byte(`{}`),
	}

	// Handle the request
	dispatcher.handleRequest(context.Background(), &request)

	// Verify the handler was called with context containing session ID
	handler.AssertExpectations(t)
}
