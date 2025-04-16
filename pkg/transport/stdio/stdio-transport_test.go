package stdio_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lucacox/go-mcp/pkg/protocol"
	"github.com/lucacox/go-mcp/pkg/transport/stdio"
)

// TestNewSTDIOTransportWithIO tests the creation of a new STDIO transport with custom reader and writer
func TestNewSTDIOTransportWithIO(t *testing.T) {
	reader := strings.NewReader("")
	writer := &bytes.Buffer{}

	transport := stdio.NewSTDIOTransportWithIO(reader, writer)

	if transport == nil {
		t.Fatal("Expected transport to be created, got nil")
	}

	if transport.Reader() == nil {
		t.Error("Expected Reader() to return a reader, got nil")
	}

	if transport.Writer() != writer {
		t.Error("Expected Writer() to return the provided writer")
	}
}

// TestSendReceive tests sending and receiving messages through the transport
func TestSendReceive(t *testing.T) {
	// Create a pipe for testing
	r, w := io.Pipe()

	// Create the transport with the pipe
	transport := stdio.NewSTDIOTransportWithIO(r, w)

	// Start the transport
	transport.Start()
	defer transport.Close()

	// Test message
	testMessage := []byte(`{"method": "test"}`)

	// Create a WaitGroup to synchronize the test
	var wg sync.WaitGroup
	wg.Add(1)

	// Start a goroutine to receive the message
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Receive the message
		received, err := transport.Receive(ctx)
		if err != nil {
			t.Errorf("Failed to receive message: %v", err)
			return
		}

		// Compare the received message with the sent one
		if !bytes.Equal(received, testMessage) {
			t.Errorf("Expected to receive %q, got %q", testMessage, received)
		}

		w.Close()
		r.Close()
	}()

	// Give time for the goroutine to start
	time.Sleep(100 * time.Millisecond)

	// Write the test message to the pipe (simulating input)
	go func() {
		_, err := w.Write(append(testMessage, '\n'))
		if err != nil {
			t.Errorf("Failed to write test message: %v", err)
		}
	}()

	// Wait for the receive operation to complete
	wg.Wait()
}

// TestSend tests the Send method
func TestSend(t *testing.T) {
	// Create a buffer to capture the output
	writer := &bytes.Buffer{}

	// Create the transport with a dummy reader and the buffer as writer
	transport := stdio.NewSTDIOTransportWithIO(strings.NewReader(""), writer)

	// Test message
	testMessage := []byte(`{"method": "test"}`)

	// Send the message
	err := transport.Send(context.Background(), testMessage)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Check that the message was written to the buffer with a newline
	expected := append(testMessage, '\n')
	if !bytes.Equal(writer.Bytes(), expected) {
		t.Errorf("Expected to write %q, wrote %q", expected, writer.Bytes())
	}
}

// TestSendAfterClose tests sending a message after the transport is closed
func TestSendAfterClose(t *testing.T) {
	// Create the transport
	transport := stdio.NewSTDIOTransportWithIO(strings.NewReader(""), &bytes.Buffer{})

	// Close the transport
	err := transport.Close()
	if err != nil {
		t.Fatalf("Failed to close transport: %v", err)
	}

	// Try to send a message after closing
	err = transport.Send(context.Background(), []byte(`{"method": "test"}`))
	if err == nil {
		t.Fatal("Expected error when sending after close, got nil")
	}
}

// TestReceiveContextCancellation tests the cancellation of a Receive operation
func TestReceiveContextCancellation(t *testing.T) {
	// Create the transport
	transport := stdio.NewSTDIOTransportWithIO(strings.NewReader(""), &bytes.Buffer{})

	// Create a context with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Try to receive a message (should time out)
	_, err := transport.Receive(ctx)
	if err == nil {
		t.Fatal("Expected context cancellation error, got nil")
	}
}

// TestFormatAndParseMessage tests the message formatting and parsing functions
func TestFormatAndParseMessage(t *testing.T) {
	// Test content
	content := []byte(`{"method":"test","params":{"key":"value"}}`)

	// Test metadata
	metadata := map[string]interface{}{
		"id":   "123",
		"type": "request",
	}

	// Format the message
	formatted, err := stdio.FormatMessage(content, metadata)
	if err != nil {
		t.Fatalf("Failed to format message: %v", err)
	}

	// Parse the formatted message
	parsedContent, parsedMetadata, err := stdio.ParseMessage(formatted)
	if err != nil {
		t.Fatalf("Failed to parse message: %v", err)
	}

	// Check the parsed content
	if !bytes.Equal(parsedContent, content) {
		t.Errorf("Expected content %q, got %q", content, parsedContent)
	}

	// Check the parsed metadata
	if parsedMetadata["id"] != metadata["id"] || parsedMetadata["type"] != metadata["type"] {
		t.Errorf("Expected metadata %v, got %v", metadata, parsedMetadata)
	}
}

// TestParseInvalidMessage tests parsing an invalid formatted message
func TestParseInvalidMessage(t *testing.T) {
	// Invalid JSON message
	invalidMessage := []byte(`invalid json`)

	// Parse the invalid message
	content, metadata, err := stdio.ParseMessage(invalidMessage)
	if err != nil {
		t.Fatalf("Expected no error for invalid format, got: %v", err)
	}

	// In case of invalid format, the original message should be returned as content
	if !bytes.Equal(content, invalidMessage) {
		t.Errorf("Expected content %q, got %q", invalidMessage, content)
	}

	// Metadata should be nil
	if metadata != nil {
		t.Errorf("Expected nil metadata, got %v", metadata)
	}
}

// TestRegisterSTDIOTransport tests registering the transport in the registry
func TestRegisterSTDIOTransport(t *testing.T) {
	// Create a new transport registry
	registry := protocol.NewTransportRegistry()

	// Register the STDIO transport
	stdio.RegisterSTDIOTransport(registry)

	// Check if the transport is registered
	creator, err := registry.GetCreator("stdio")
	if err != nil {
		t.Fatalf("Expected stdio transport to be registered, got error: %v", err)
	}

	// Create a transport using the creator
	transport, err := creator(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create transport with registered creator: %v", err)
	}

	// Check if the created transport is of the correct type
	_, ok := transport.(*stdio.STDIOTransport)
	if !ok {
		t.Error("Created transport is not of type *stdio.STDIOTransport")
	}
}

// TestSTDIOTransportCreator tests the factory function for creating transports
func TestSTDIOTransportCreator(t *testing.T) {
	transport, err := stdio.STDIOTransportCreator(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to create transport with creator: %v", err)
	}

	if transport == nil {
		t.Fatal("Expected transport to be created, got nil")
	}

	_, ok := transport.(*stdio.STDIOTransport)
	if !ok {
		t.Error("Created transport is not of type *stdio.STDIOTransport")
	}
}

// TestCloseMultipleTimes tests closing the transport multiple times
func TestCloseMultipleTimes(t *testing.T) {
	transport := stdio.NewSTDIOTransportWithIO(strings.NewReader(""), &bytes.Buffer{})

	// Close the first time
	err := transport.Close()
	if err != nil {
		t.Fatalf("First close failed: %v", err)
	}

	// Close the second time (should be a no-op)
	err = transport.Close()
	if err != nil {
		t.Fatalf("Second close failed: %v", err)
	}
}

// TestReadEOF tests the behavior of readLoop when an EOF is encountered
func TestReadEOF(t *testing.T) {
	// Create a reader that will return EOF after some data
	r, w := io.Pipe()

	// Create the transport
	transport := stdio.NewSTDIOTransportWithIO(r, &bytes.Buffer{})

	// Start the transport
	transport.Start()
	defer transport.Close()

	// Write some data followed by closing the writer (which will cause EOF)
	testMessage := []byte(`{"method": "test"}`)
	go func() {
		_, err := w.Write(append(testMessage, '\n'))
		if err != nil {
			t.Errorf("Failed to write test message: %v", err)
		}
		// Close the writer to cause EOF
		w.Close()
	}()

	// Give time for the readLoop to process the message
	time.Sleep(100 * time.Millisecond)

	// Try to receive the message
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	received, err := transport.Receive(ctx)
	if err != nil {
		t.Fatalf("Failed to receive message: %v", err)
	}

	// Check the received message
	if !bytes.Equal(received, testMessage) {
		t.Errorf("Expected to receive %q, got %q", testMessage, received)
	}
}
