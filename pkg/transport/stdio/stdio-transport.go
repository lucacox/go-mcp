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

	for {
		select {
		case <-t.done:
			return
		default:
			// Read the input line
			line, err := t.reader.ReadBytes('\n')
			line = bytes.TrimSpace(line)
			if err != nil {
				if err == io.EOF {
					// With a named pipe (FIFO), EOF only indicates that the writer has closed
					// but doesn't mean we should terminate the transport
					if len(line) > 0 {
						// If we read some data before the EOF, process it
						t.incomingMessages <- line
					}
					// Wait a short interval before trying to read again
					time.Sleep(100 * time.Millisecond)
					continue
				} else {
					// Other reading errors
					logging.Error(t.logger, "error reading from stdin", "error", err)
					continue
				}
			}

			if len(line) > 0 {
				// Add the message to the incoming messages channel
				t.incomingMessages <- line
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
	_, err := t.writer.Write(data)
	return err
}

// Receive receives a message from the sender
func (t *STDIOTransport) Receive(ctx context.Context) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case data := <-t.incomingMessages:
		return data, nil
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
