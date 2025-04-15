// Package http provides an HTTP transport implementation for MCP
package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/lucacox/go-mcp/pkg/protocol"
)

// Auto-registro il trasporto HTTP nel registro predefinito
func init() {
	// RegisterHTTPTransport(protocol.DefaultTransportRegistry)
}

// HTTPTransport implements an MCP transport over HTTP
type HTTPTransport struct {
	// Server or client URL
	url string

	// HTTP client
	client *http.Client

	// HTTP server (server side only)
	server *http.Server

	// Incoming message buffer
	incomingMessages chan []byte

	// Mutex for synchronization
	mutex sync.Mutex

	// Closed state
	closed bool
}

// HTTPTransportOptions represents the options for HTTP transport
type HTTPTransportOptions struct {
	// MCP server URL (client side)
	URL string

	// Listening address (server side)
	ListenAddress string

	// Custom HTTP handler (server side)
	Handler http.Handler

	// Request timeout
	Timeout time.Duration
}

// NewHTTPClientTransport creates a new HTTP transport for the client side
func NewHTTPClientTransport(url string, timeout time.Duration) *HTTPTransport {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &HTTPTransport{
		url:              url,
		client:           &http.Client{Timeout: timeout},
		incomingMessages: make(chan []byte, 100),
	}
}

// NewHTTPServerTransport creates a new HTTP transport for the server side
func NewHTTPServerTransport(listenAddr string, handler http.Handler) (*HTTPTransport, error) {
	transport := &HTTPTransport{
		incomingMessages: make(chan []byte, 100),
	}

	// Configure the HTTP server
	transport.server = &http.Server{
		Addr:    listenAddr,
		Handler: handler,
	}

	return transport, nil
}

// StartServer starts the HTTP server
func (t *HTTPTransport) StartServer() error {
	if t.server == nil {
		return fmt.Errorf("server not configured")
	}

	// Start the server in a goroutine
	go func() {
		err := t.server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			// Log the error
		}
	}()

	return nil
}

// Send sends a message to the recipient
func (t *HTTPTransport) Send(ctx context.Context, data []byte) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.closed {
		return fmt.Errorf("transport closed")
	}

	// Client side: send a POST request to the server
	req, err := http.NewRequestWithContext(ctx, "POST", t.url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check the response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read the response and add it to the incoming messages
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Add the message to the incoming messages channel
	t.incomingMessages <- respData

	return nil
}

// Receive receives a message from the sender
func (t *HTTPTransport) Receive(ctx context.Context) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case data := <-t.incomingMessages:
		return data, nil
	}
}

// Close closes the transport connection
func (t *HTTPTransport) Close() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.closed {
		return nil
	}

	t.closed = true

	// Close the server if it has been initialized
	if t.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return t.server.Shutdown(ctx)
	}

	return nil
}

// HTTPTransportCreator is a factory for creating HTTP transports
func HTTPTransportCreator(ctx context.Context, options map[string]interface{}) (protocol.Transport, error) {
	// Extract options
	url, _ := options["url"].(string)
	if url == "" {
		return nil, fmt.Errorf("URL is required")
	}

	timeout := 30 * time.Second
	if timeoutMs, ok := options["timeout"].(float64); ok {
		timeout = time.Duration(timeoutMs) * time.Millisecond
	}

	return NewHTTPClientTransport(url, timeout), nil
}

// RegisterHTTPTransport registers the HTTP transport in the transport registry
func RegisterHTTPTransport(registry *protocol.TransportRegistry) {
	registry.Register(protocol.TransportTypeHTTP, HTTPTransportCreator)
}

// HTTPHandler is an HTTP handler for the MCP server
type HTTPHandler struct {
	// Callback called when a message arrives
	messageCallback func([]byte) []byte
}

// NewHTTPHandler creates a new HTTP handler for the MCP server
func NewHTTPHandler(callback func([]byte) []byte) *HTTPHandler {
	return &HTTPHandler{
		messageCallback: callback,
	}
}

// ServeHTTP implements the http.Handler interface
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check that it is a POST request
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	// Process the message
	var response []byte
	if h.messageCallback != nil {
		response = h.messageCallback(body)
	}

	// Send the response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}
