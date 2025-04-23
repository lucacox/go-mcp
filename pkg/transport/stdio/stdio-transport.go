// Package stdio provides an stdio transport implementation for MCP
package stdio

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/lucacox/go-mcp/internal/logging"
	"github.com/lucacox/go-mcp/pkg/protocol"
)

// Auto-register the STDIO transport in the default registry
func init() {
	// RegisterSTDIOTransport(protocol.DefaultTransportRegistry)
}

// STDIOTransport implements an MCP transport via stdin/stdout
type STDIOTransport struct {
	// Reader for standard input
	reader *bufio.Reader

	// Writer for standard output
	writer io.Writer

	// Incoming messages buffer
	incomingMessages chan []byte

	logger *slog.Logger

	// Mutex for synchronization
	mutex sync.Mutex

	// Reading goroutine
	wg sync.WaitGroup

	// Close channel
	done chan struct{}

	// Closed state
	closed bool
}

// NewSTDIOTransport creates a new stdio transport
func NewSTDIOTransport() *STDIOTransport {
	loggerFactory := logging.NewLoggerFactory()
	loggerFactory.SetLevel(slog.LevelDebug)

	return &STDIOTransport{
		reader:           bufio.NewReader(os.Stdin),
		writer:           os.Stdout,
		incomingMessages: make(chan []byte, 100),
		done:             make(chan struct{}),
		logger:           loggerFactory.CreateLogger("stdio-transport"),
	}
}

// NewSTDIOTransportWithIO creates a new stdio transport with custom reader and writer
func NewSTDIOTransportWithIO(reader io.Reader, writer io.Writer) *STDIOTransport {
	loggerFactory := logging.NewLoggerFactory()
	loggerFactory.SetLevel(slog.LevelDebug)

	return &STDIOTransport{
		reader:           bufio.NewReader(reader),
		writer:           writer,
		incomingMessages: make(chan []byte, 100),
		done:             make(chan struct{}),
		logger:           loggerFactory.CreateLogger("stdio-transport"),
	}
}

// Start starts the transport
func (t *STDIOTransport) Start() {
	t.wg.Add(1)
	go t.readLoop()
}

// readLoop is the request reading loop
func (t *STDIOTransport) readLoop() {
	defer t.wg.Done()

	// Use a single channel for data with a buffer to reduce potential blocking
	dataChannel := make(chan []byte, 100)

	// Start a goroutine that continuously reads from stdin
	go func() {
		for !t.closed {
			readData, readErr := t.reader.ReadBytes('\n')

			// logging.Debug(t.logger, "readLoop: read from stdin", "data", string(readData), "error", readErr)

			// Process data if we have any
			if len(readData) > 0 {
				select {
				case dataChannel <- readData:
					// Successfully sent data
				case <-t.done:
					// Transport is closing, exit
					return
				}
			}

			// Handle errors
			if readErr != nil {
				if readErr == io.EOF {
					// EOF is normal when writer closes, just log at debug level
					// logging.Debug(t.logger, "readLoop: EOF reached")
				} else {
					// Log other errors
					logging.Error(t.logger, "error reading from stdin", "error", readErr)
				}

				// Small backoff on error to prevent CPU spinning
				select {
				case <-time.After(100 * time.Millisecond):
					// Continue after backoff
				case <-t.done:
					// Transport is closing, exit
					return
				}
			}
		}
	}()

	// Main loop: process data and check for termination
	for {
		select {
		case <-t.done:
			logging.Debug(t.logger, "readLoop: done")
			return
		case data := <-dataChannel:
			// Process each message as it arrives
			trimmedData := bytes.TrimSpace(data)
			if len(trimmedData) > 0 {
				t.incomingMessages <- trimmedData
			}
		}
	}
}

// Send sends a message to the recipient
func (t *STDIOTransport) Send(ctx context.Context, data []byte) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.closed {
		return fmt.Errorf("transport closed")
	}

	// Add a newline to the message
	data = append(data, '\n')

	// Write the message to standard output
	// logging.Debug(t.logger, "Send: writing to stdout", "data", string(data))
	_, err := t.writer.Write(data)
	return err
}

// Receive receives a message from the sender
func (t *STDIOTransport) Receive(ctx context.Context) ([]byte, error) {
	// Add a timeout if not already present in the context
	var cancel context.CancelFunc = func() {}
	deadline, ok := ctx.Deadline()
	if !ok {
		// If no deadline is set, use a reasonable default (500ms)
		ctx, cancel = context.WithTimeout(ctx, 500*time.Millisecond)
		defer cancel()
	} else {
		// If deadline is very far in the future, add a more reasonable timeout
		if time.Until(deadline) > 30*time.Second {
			ctx, cancel = context.WithTimeout(ctx, 500*time.Millisecond)
			defer cancel()
		}
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case data, ok := <-t.incomingMessages:
		if !ok {
			return nil, fmt.Errorf("channel closed")
		}
		return data, nil
	case <-time.After(50 * time.Millisecond):
		// Add a shorter timeout to prevent blocking indefinitely
		// This will allow us to check ctx.Done() more frequently
		return nil, fmt.Errorf("no message available")
	}
}

// Close closes the transport connection
func (t *STDIOTransport) Close() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.closed {
		return nil
	}

	t.closed = true
	close(t.done)
	t.wg.Wait()

	return nil
}

// Reader implements the BiDirectionalTransport interface
func (t *STDIOTransport) Reader() io.Reader {
	return t.reader
}

// Writer implements the BiDirectionalTransport interface
func (t *STDIOTransport) Writer() io.Writer {
	return t.writer
}

// STDIOTransportCreator is a factory for creating stdio transports
func STDIOTransportCreator(ctx context.Context, options map[string]interface{}) (protocol.Transport, error) {
	return NewSTDIOTransport(), nil
}

// RegisterSTDIOTransport registers the stdio transport in the transport registry
func RegisterSTDIOTransport(registry *protocol.TransportRegistry) {
	registry.Register("stdio", STDIOTransportCreator)
}

// MessageFormat represents the format of an stdio message
type MessageFormat struct {
	// JSON-RPC message content
	Content json.RawMessage `json:"content"`

	// Additional metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// FormatMessage formats a message for stdio transport
func FormatMessage(content []byte, metadata map[string]interface{}) ([]byte, error) {
	msg := MessageFormat{
		Content:  content,
		Metadata: metadata,
	}

	return json.Marshal(msg)
}

// ParseMessage extracts the content from a formatted message
func ParseMessage(data []byte) ([]byte, map[string]interface{}, error) {
	var msg MessageFormat
	if err := json.Unmarshal(data, &msg); err != nil {
		// If not a valid format, we assume it's the content directly
		return data, nil, nil
	}

	return msg.Content, msg.Metadata, nil
}
