// Package protocol provides types and utilities for JSON-RPC communication
package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lucacox/go-mcp/internal/logging"
)

// JSON-RPC 2.0 error codes
const (
	ErrorCodeParseError     int = -32700
	ErrorCodeInvalidRequest int = -32600
	ErrorCodeMethodNotFound int = -32601
	ErrorCodeInvalidParams  int = -32602
	ErrorCodeInternalError  int = -32603
)

// JSONRPCVersion is the supported JSON-RPC protocol version
const JSONRPCVersion = "2.0"

// JSONRPCMessage represents a generic JSON-RPC message
type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCRequest represents a JSON-RPC request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCNotification represents a JSON-RPC notification
type JSONRPCNotification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC response
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC error
type JSONRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Error implements the error interface
func (e *JSONRPCError) Error() string {
	return e.Message
}

func NewJSONRPCError(code int, message string, data interface{}) *JSONRPCError {
	var dataJSON json.RawMessage
	if data != nil {
		bytes, err := json.Marshal(data)
		if err == nil {
			dataJSON = bytes
		}
	}
	return &JSONRPCError{
		Code:    code,
		Message: message,
		Data:    dataJSON,
	}
}

// RPCHandler is an interface for handling RPC requests
type RPCHandler interface {
	// HandleRequest handles an RPC request
	HandleRequest(ctx context.Context, method string, params json.RawMessage) (interface{}, error)
}

// RPCClient is an interface for sending RPC requests
type RPCClient interface {
	// Call sends an RPC request and waits for a response
	Call(ctx context.Context, method string, params interface{}) (json.RawMessage, error)

	// Notify sends an RPC notification (without waiting for a response)
	Notify(ctx context.Context, method string, params interface{}) error
}

// JSONRPCDispatcher manages sending and receiving JSON-RPC messages
type JSONRPCDispatcher struct {
	transport  Transport
	handler    RPCHandler
	pending    map[string]chan *JSONRPCMessage
	pendingMux sync.Mutex
	shutdown   chan struct{}
	wg         sync.WaitGroup
	sessionID  string       // ID della sessione associata a questo dispatcher
	logger     *slog.Logger // Optional logger for error reporting
}

// NewJSONRPCDispatcher creates a new JSON-RPC dispatcher
func NewJSONRPCDispatcher(transport Transport, handler RPCHandler) *JSONRPCDispatcher {
	factory := logging.NewLoggerFactory()
	dispatcher := &JSONRPCDispatcher{
		transport:  transport,
		handler:    handler,
		pending:    make(map[string]chan *JSONRPCMessage),
		pendingMux: sync.Mutex{},
		shutdown:   make(chan struct{}),
		logger:     factory.CreateLogger("dispatcher"),
	}

	return dispatcher
}

// Start starts the dispatcher to receive messages
func (d *JSONRPCDispatcher) Start() {
	d.wg.Add(1)
	go d.receiveLoop()
}

// Stop stops the dispatcher
func (d *JSONRPCDispatcher) Stop() {
	close(d.shutdown)
	d.wg.Wait()
}

// receiveLoop handles the message receiving loop
func (d *JSONRPCDispatcher) receiveLoop() {
	defer d.wg.Done()

	for {
		select {
		case <-d.shutdown:
			return
		default:
			// Create a new context with timeout for each receive attempt
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			data, err := d.transport.Receive(ctx)
			cancel() // Cancel the context immediately after use

			if err != nil {
				// Add a small sleep to avoid CPU spinning when no messages are available
				time.Sleep(10 * time.Millisecond)
				continue
			}

			var message JSONRPCMessage
			if err := json.Unmarshal(data, &message); err != nil {
				// Log the error but don't consume 100% CPU
				if d.logger != nil {
					logging.Warn(d.logger, "error unmarshaling message", "error", err)
				}
				continue
			}

			// Handle different message types
			if message.Method != "" && message.ID != nil {
				// It's a request
				go d.handleRequest(ctx, &message)
			} else if message.Method != "" && message.ID == nil {
				// It's a notification
				go d.handleNotification(ctx, &message)
			} else {
				// It's a response
				d.handleResponse(&message)
			}
		}
	}
}

// handleRequest handles an RPC request
func (d *JSONRPCDispatcher) handleRequest(ctx context.Context, msg *JSONRPCMessage) {
	if d.handler == nil {
		// Send a method not supported error
		d.sendErrorResponse(ctx, msg.ID, -32601, "Method not found", nil)
		return
	}

	// Aggiungi l'ID della sessione al contesto, se presente
	if d.sessionID != "" {
		ctx = WithSessionID(ctx, d.sessionID)
	}

	result, err := d.handler.HandleRequest(ctx, msg.Method, msg.Params)
	if err != nil {
		// Handle the error
		code := -32603 // Internal error
		message := err.Error()

		if rpcErr, ok := err.(*JSONRPCError); ok {
			code = rpcErr.Code
			message = rpcErr.Message
		}

		d.sendErrorResponse(ctx, msg.ID, code, message, nil)
		return
	}

	// Send the response
	d.sendResponse(ctx, msg.ID, result)
}

// handleNotification handles an RPC notification (no response)
func (d *JSONRPCDispatcher) handleNotification(ctx context.Context, msg *JSONRPCMessage) {
	if d.handler == nil {
		return
	}

	// Aggiungi l'ID della sessione al contesto, se presente
	if d.sessionID != "" {
		ctx = WithSessionID(ctx, d.sessionID)
	}

	_, _ = d.handler.HandleRequest(ctx, msg.Method, msg.Params)
	// We don't send a response for notifications
}

// handleResponse handles a received RPC response
func (d *JSONRPCDispatcher) handleResponse(msg *JSONRPCMessage) {
	var idStr string
	json.Unmarshal(msg.ID, &idStr)

	d.pendingMux.Lock()
	defer d.pendingMux.Unlock()

	ch, exists := d.pending[idStr]
	if exists {
		ch <- msg
		delete(d.pending, idStr)
	}
}

// Call sends an RPC request and waits for a response
func (d *JSONRPCDispatcher) Call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	// Generate a unique ID for the request
	id := uuid.New().String()

	// Prepare the request message
	request := &JSONRPCMessage{
		JSONRPC: JSONRPCVersion,
		ID:      []byte(`"` + id + `"`),
		Method:  method,
	}

	// Serialize the parameters
	if params != nil {
		paramsJSON, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("error marshaling parameters: %w", err)
		}
		request.Params = paramsJSON
	}

	// Create a channel to receive the response
	responseCh := make(chan *JSONRPCMessage, 1)

	// Register the channel for this request
	d.pendingMux.Lock()
	d.pending[id] = responseCh
	d.pendingMux.Unlock()

	// Send the request
	requestJSON, err := json.Marshal(request)
	if err != nil {
		d.pendingMux.Lock()
		delete(d.pending, id)
		d.pendingMux.Unlock()
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	if err := d.transport.Send(ctx, requestJSON); err != nil {
		d.pendingMux.Lock()
		delete(d.pending, id)
		d.pendingMux.Unlock()
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	// Wait for the response or timeout
	select {
	case <-ctx.Done():
		d.pendingMux.Lock()
		delete(d.pending, id)
		d.pendingMux.Unlock()
		return nil, ctx.Err()
	case response := <-responseCh:
		if response.Error != nil {
			return nil, response.Error
		}
		return response.Result, nil
	}
}

// Notify sends an RPC notification (without waiting for a response)
func (d *JSONRPCDispatcher) Notify(ctx context.Context, method string, params interface{}) error {
	// Prepare the notification message (without ID)
	notification := &JSONRPCMessage{
		JSONRPC: JSONRPCVersion,
		Method:  method,
	}

	// Serialize the parameters
	if params != nil {
		paramsJSON, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("error marshaling parameters: %w", err)
		}
		notification.Params = paramsJSON
	}

	// Send the notification
	notificationJSON, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("error marshaling notification: %w", err)
	}

	if err := d.transport.Send(ctx, notificationJSON); err != nil {
		return fmt.Errorf("error sending notification: %w", err)
	}

	return nil
}

// sendErrorResponse sends an error response
func (d *JSONRPCDispatcher) sendErrorResponse(ctx context.Context, id json.RawMessage, code int, message string, data interface{}) {
	var dataJSON json.RawMessage
	if data != nil {
		bytes, err := json.Marshal(data)
		if err == nil {
			dataJSON = bytes
		}
	}

	response := &JSONRPCMessage{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    dataJSON,
		},
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		// We can't do much if serialization fails
		return
	}

	_ = d.transport.Send(ctx, responseJSON)
}

// sendResponse sends a response with a result
func (d *JSONRPCDispatcher) sendResponse(ctx context.Context, id json.RawMessage, result interface{}) {
	response := &JSONRPCMessage{
		JSONRPC: JSONRPCVersion,
		ID:      id,
	}

	if result != nil {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			d.sendErrorResponse(ctx, id, -32603, "Internal error", nil)
			return
		}
		response.Result = resultJSON
	} else {
		response.Result = []byte("null")
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		// We can't do much if serialization fails
		return
	}

	_ = d.transport.Send(ctx, responseJSON)
}

// SetSessionID sets the session ID for this dispatcher
func (d *JSONRPCDispatcher) SetSessionID(sessionID string) {
	d.sessionID = sessionID
}
