package protocol

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTransportConstants tests the transport type constants
func TestTransportConstants(t *testing.T) {
	// Verify transport type constants
	assert.Equal(t, "stdio", TransportTypeStdio)
	assert.Equal(t, "http", TransportTypeHTTP)
}

// TestDefaultTransportRegistry verifies that the default registry exists
func TestDefaultTransportRegistry(t *testing.T) {
	assert.NotNil(t, DefaultTransportRegistry)
}

// TestNewTransportRegistry tests creating a new transport registry
func TestNewTransportRegistry(t *testing.T) {
	registry := NewTransportRegistry()

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.creators)
	assert.Empty(t, registry.creators)
}

// TestTransportRegistry_Register tests registering a transport creator
func TestTransportRegistry_Register(t *testing.T) {
	registry := NewTransportRegistry()

	// Create a mock creator function
	creatorFunc := func(ctx context.Context, options map[string]interface{}) (Transport, error) {
		return NewMockTransport(10), nil
	}

	// Register the creator
	registry.Register("test-transport", creatorFunc)

	// Verify it was registered
	assert.Len(t, registry.creators, 1)
	assert.Contains(t, registry.creators, "test-transport")
}

// TestTransportRegistry_Create tests creating a transport instance
func TestTransportRegistry_Create(t *testing.T) {
	registry := NewTransportRegistry()
	mockTransport := NewMockTransport(10)

	// Create a mock creator function
	creatorFunc := func(ctx context.Context, options map[string]interface{}) (Transport, error) {
		// Verify options are passed correctly
		if value, ok := options["test"]; ok && value == "value" {
			return mockTransport, nil
		}
		return nil, errors.New("invalid options")
	}

	// Register the creator
	registry.Register("test-transport", creatorFunc)

	t.Run("SuccessfulCreate", func(t *testing.T) {
		// Create a transport with valid options
		transport, err := registry.Create(context.Background(), "test-transport", map[string]interface{}{
			"test": "value",
		})

		// Verify
		assert.NoError(t, err)
		assert.Equal(t, mockTransport, transport)
	})

	t.Run("InvalidTransportType", func(t *testing.T) {
		// Try to create a transport with an unregistered type
		transport, err := registry.Create(context.Background(), "unknown-transport", nil)

		// Verify
		assert.Error(t, err)
		assert.Nil(t, transport)

		// Check error type
		var transportErr *TransportError
		assert.ErrorAs(t, err, &transportErr)
		assert.Contains(t, transportErr.Message, "transport type not supported")
	})

	t.Run("CreatorError", func(t *testing.T) {
		// Try to create a transport with invalid options
		transport, err := registry.Create(context.Background(), "test-transport", map[string]interface{}{
			"test": "wrong-value",
		})

		// Verify
		assert.Error(t, err)
		assert.Nil(t, transport)
		assert.Contains(t, err.Error(), "invalid options")
	})
}

// TestTransportRegistry_HasTransport tests checking if a transport type is supported
func TestTransportRegistry_HasTransport(t *testing.T) {
	registry := NewTransportRegistry()

	// Register a test transport
	registry.Register("test-transport", func(ctx context.Context, options map[string]interface{}) (Transport, error) {
		return NewMockTransport(10), nil
	})

	// Check for registered and unregistered transports
	assert.True(t, registry.HasTransport("test-transport"))
	assert.False(t, registry.HasTransport("unknown-transport"))
}

// TestTransportRegistry_GetSupportedTransports tests getting the list of supported transport types
func TestTransportRegistry_GetSupportedTransports(t *testing.T) {
	registry := NewTransportRegistry()

	// Initially no transports
	transports := registry.GetSupportedTransports()
	assert.Empty(t, transports)

	// Register some transports
	registry.Register("transport1", func(ctx context.Context, options map[string]interface{}) (Transport, error) {
		return NewMockTransport(10), nil
	})
	registry.Register("transport2", func(ctx context.Context, options map[string]interface{}) (Transport, error) {
		return NewMockTransport(10), nil
	})

	// Get supported transports
	transports = registry.GetSupportedTransports()

	// Verify
	assert.Len(t, transports, 2)
	assert.Contains(t, transports, "transport1")
	assert.Contains(t, transports, "transport2")
}

// TestTransportRegistry_GetCreator tests retrieving a transport creator function
func TestTransportRegistry_GetCreator(t *testing.T) {
	registry := NewTransportRegistry()

	// Create a mock creator function
	creatorFunc := func(ctx context.Context, options map[string]interface{}) (Transport, error) {
		return NewMockTransport(10), nil
	}

	// Register the creator
	registry.Register("test-transport", creatorFunc)

	// Get the creator
	creator, err := registry.GetCreator("test-transport")

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, creator)
	tran, err := creator(context.TODO(), nil)
	assert.NoError(t, err)
	assert.IsType(t, &MockTransport{}, tran)

	// Try to get a non-existing creator
	creator, err = registry.GetCreator("unknown-transport")

	// Verify
	assert.Error(t, err)
	assert.Nil(t, creator)
	var transportErr *TransportError
	assert.ErrorAs(t, err, &transportErr)
	assert.Contains(t, transportErr.Message, "transport type not supported")
}

// TestCreateTransport tests the convenience function for creating a transport
func TestCreateTransport(t *testing.T) {
	// Backup the default registry
	originalRegistry := DefaultTransportRegistry

	// Create a new registry for testing
	testRegistry := NewTransportRegistry()

	// Replace the default registry
	DefaultTransportRegistry = testRegistry

	// Restore the original registry after the test
	defer func() {
		DefaultTransportRegistry = originalRegistry
	}()

	// Register a test transport
	mockTransport := NewMockTransport(10)
	testRegistry.Register("test-transport", func(ctx context.Context, options map[string]interface{}) (Transport, error) {
		return mockTransport, nil
	})

	// Create a transport using the convenience function
	transport, err := CreateTransport(context.Background(), "test-transport", nil)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, mockTransport, transport)
}

// TestTransportError tests the transport error implementation
func TestTransportError(t *testing.T) {
	// Test creating a transport error
	err := &TransportError{
		Message: "test error",
	}

	// Test error method
	assert.Equal(t, "test error", err.Error())

	// Test with cause
	cause := errors.New("underlying error")
	err.Cause = cause

	// Test error method with cause
	assert.Equal(t, "test error: underlying error", err.Error())

	// Test unwrap
	assert.Equal(t, cause, err.Unwrap())

	// Test with cause method
	err2 := &TransportError{Message: "another error"}
	err2WithCause := err2.WithCause(cause)

	// Verify
	assert.Equal(t, err2, err2WithCause) // Should return the same instance
	assert.Equal(t, cause, err2.Cause)
	assert.Equal(t, "another error: underlying error", err2.Error())
}

// TestBiDirectionalTransport_Interface ensures that the BiDirectionalTransport interface extends Transport
func TestBiDirectionalTransport_Interface(t *testing.T) {
	// Create a proper mock that implements BiDirectionalTransport
	mockBiDiTransport := &MockBiDirectionalTransport{}

	// Configure the mock to return non-nil values
	mockBiDiTransport.On("Reader").Return(io.NopCloser(nil)).Maybe()
	mockBiDiTransport.On("Writer").Return(io.Discard).Maybe()

	// Verify it implements both interfaces
	var _ Transport = mockBiDiTransport
	var _ BiDirectionalTransport = mockBiDiTransport
}

// MockBiDirectionalTransport is a mock implementation for testing BiDirectionalTransport
type MockBiDirectionalTransport struct {
	MockTransport
}

func (m *MockBiDirectionalTransport) Reader() io.Reader {
	args := m.Called()
	return args.Get(0).(io.Reader)
}

func (m *MockBiDirectionalTransport) Writer() io.Writer {
	args := m.Called()
	return args.Get(0).(io.Writer)
}
