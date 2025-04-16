package protocol

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProtocolVersion tests the protocol version constants
func TestProtocolVersion(t *testing.T) {
	// Verify protocol versions are correct
	assert.Equal(t, ProtocolVersion("2024-11-05"), ProtocolVersion20241105)
	assert.Equal(t, ProtocolVersion("2025-03-26"), ProtocolVersion20250326)
}

// TestNamespace tests the namespace constants
func TestNamespace(t *testing.T) {
	// Verify namespace constants
	assert.Equal(t, Namespace(""), EmptyNamespace)
	assert.Equal(t, Namespace("notifications"), NotifficationsNamespace)
	assert.Equal(t, Namespace("tools"), ToolsNamespace)
	assert.Equal(t, Namespace("roots"), RootsNamespace)
	assert.Equal(t, Namespace("sampling"), SamplingNamespace)
	assert.Equal(t, Namespace("prompts"), PromptsNamespace)
	assert.Equal(t, Namespace("resources"), ResourcesNamespace)
	assert.Equal(t, Namespace("completion"), CompletionNamespace)
	assert.Equal(t, Namespace("logging"), LoggingNamespace)
}

// TestBuildMethod tests building method names with namespace
func TestBuildMethod(t *testing.T) {
	tests := []struct {
		method         string
		namespace      Namespace
		expected       string
		isNotification bool
	}{
		{"method", EmptyNamespace, "method", false},
		{"method", ToolsNamespace, "tools/method", false},
		{"method", PromptsNamespace, "prompts/method", false},

		// Notification methods
		{"method", EmptyNamespace, "notifications/method", true},
		{"method", ToolsNamespace, "notifications/tools/method", true},
	}

	for _, test := range tests {
		if !test.isNotification {
			result := BuildMethod(test.method, test.namespace)
			assert.Equal(t, test.expected, result)
		} else {
			result := BuildNotificationsMethod(test.method, test.namespace)
			assert.Equal(t, test.expected, result)
		}
	}
}

// TestNewEndpointRegistry tests creating a new endpoint registry
func TestNewEndpointRegistry(t *testing.T) {
	registry := NewEndpointRegistry()
	assert.NotNil(t, registry)
	assert.NotNil(t, registry.endpoints)
}

// TestEndpointRegistry_RegisterEndpoint tests registering an endpoint
func TestEndpointRegistry_RegisterEndpoint(t *testing.T) {
	registry := NewEndpointRegistry()

	// Create an endpoint
	endpoint := NewBaseEndpoint(ToolsNamespace)

	// Register the endpoint
	registry.RegisterEndpoint(endpoint)

	// Verify endpoint was registered
	assert.Len(t, registry.endpoints, 1)
	assert.Contains(t, registry.endpoints, ToolsNamespace)
}

// TestEndpointRegistry_UnregisterEndpoint tests unregistering an endpoint
func TestEndpointRegistry_UnregisterEndpoint(t *testing.T) {
	registry := NewEndpointRegistry()

	// Create an endpoint
	endpoint := NewBaseEndpoint(ToolsNamespace)

	// Register the endpoint
	registry.RegisterEndpoint(endpoint)

	// Verify endpoint was registered
	assert.Len(t, registry.endpoints, 1)

	// Unregister the endpoint
	registry.UnregisterEndpoint(ToolsNamespace)

	// Verify endpoint was unregistered
	assert.Len(t, registry.endpoints, 0)
	assert.NotContains(t, registry.endpoints, ToolsNamespace)

	// Unregister a non-existing endpoint (shouldn't cause issues)
	registry.UnregisterEndpoint(PromptsNamespace)
	assert.Len(t, registry.endpoints, 0)
}

// TestEndpointRegistry_GetEndpoint tests retrieving an endpoint
func TestEndpointRegistry_GetEndpoint(t *testing.T) {
	registry := NewEndpointRegistry()

	// Create an endpoint
	endpoint := NewBaseEndpoint(ToolsNamespace)

	// Register the endpoint
	registry.RegisterEndpoint(endpoint)

	// Get a registered endpoint
	retrievedEndpoint, exists := registry.GetEndpoint(ToolsNamespace)
	assert.True(t, exists)
	assert.Equal(t, endpoint, retrievedEndpoint)

	// Try to get a non-existing endpoint
	retrievedEndpoint, exists = registry.GetEndpoint(PromptsNamespace)
	assert.False(t, exists)
	assert.Nil(t, retrievedEndpoint)
}

// TestEndpointRegistry_HandleRequest tests handling a request through the endpoint registry
func TestEndpointRegistry_HandleRequest(t *testing.T) {
	registry := NewEndpointRegistry()

	// Define a test handler for the endpoint
	testHandler := func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var input map[string]string
		if params != nil && len(params) > 0 {
			_ = json.Unmarshal(params, &input)
		}
		return map[string]string{"status": "success", "echo": input["value"]}, nil
	}

	// Create an endpoint
	endpoint := NewBaseEndpoint(ToolsNamespace)
	endpoint.RegisterMethod("test", testHandler)
	endpoint.RegisterNotification("event", testHandler)

	// Register the endpoint
	registry.RegisterEndpoint(endpoint)

	// Test cases for various request formats
	tests := []struct {
		name          string
		method        string
		params        string
		expectSuccess bool
		expectedError bool
	}{
		{"ValidMethodRequest", "tools/test", `{"value":"hello"}`, true, false},
		{"ValidNotification", "notifications/tools/event", `{"value":"notify"}`, true, false},
		{"NotificationOnEmptyNamespace", "notifications/ping", `{}`, false, true},
		{"UnknownNamespace", "unknown/test", `{}`, false, true},
		{"UnknownMethod", "tools/unknown", `{}`, false, true},
		{"InvalidMethodFormat", "", `{}`, false, true},
		{"ValidEmptyNamespaceMethod", "ping", `{}`, false, true}, // Will fail because no default endpoint
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			params := json.RawMessage(test.params)
			result, err := registry.HandleRequest(context.Background(), test.method, params)

			if test.expectSuccess {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if response, ok := result.(map[string]string); ok {
					assert.Equal(t, "success", response["status"])
				}
			} else if test.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			}
		})
	}
}

// TestNewBaseEndpoint tests creating a base endpoint
func TestNewBaseEndpoint(t *testing.T) {
	namespace := ToolsNamespace
	endpoint := NewBaseEndpoint(namespace)

	assert.NotNil(t, endpoint)
	assert.Equal(t, namespace, endpoint.namespace)
	assert.NotNil(t, endpoint.methods)
	assert.NotNil(t, endpoint.notifications)
}

// TestBaseEndpoint_GetNamespace tests getting the namespace of an endpoint
func TestBaseEndpoint_GetNamespace(t *testing.T) {
	namespace := ToolsNamespace
	endpoint := NewBaseEndpoint(namespace)

	assert.Equal(t, namespace, endpoint.GetNamespace())
}

// TestBaseEndpoint_GetMethods tests getting the methods of an endpoint
func TestBaseEndpoint_GetMethods(t *testing.T) {
	endpoint := NewBaseEndpoint(ToolsNamespace)

	// Initially no methods
	assert.Empty(t, endpoint.GetMethods())

	// Register some methods
	endpoint.RegisterMethod("method1", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		return nil, nil
	})
	endpoint.RegisterMethod("method2", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		return nil, nil
	})

	// Get methods
	methods := endpoint.GetMethods()

	// Verify methods
	assert.Len(t, methods, 2)
	assert.Contains(t, methods, "method1")
	assert.Contains(t, methods, "method2")
}

// TestBaseEndpoint_RegisterMethod tests registering a method on an endpoint
func TestBaseEndpoint_RegisterMethod(t *testing.T) {
	endpoint := NewBaseEndpoint(ToolsNamespace)

	// Register a method
	handler := func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		return "result", nil
	}
	endpoint.RegisterMethod("test", handler)

	// Verify method was registered
	assert.Contains(t, endpoint.methods, "test")
	assert.NotNil(t, endpoint.methods["test"])
}

// TestBaseEndpoint_RegisterNotification tests registering a notification on an endpoint
func TestBaseEndpoint_RegisterNotification(t *testing.T) {
	endpoint := NewBaseEndpoint(ToolsNamespace)

	// Register a notification
	handler := func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		return nil, nil
	}
	endpoint.RegisterNotification("event", handler)
	// Verify notification was registered
	assert.Contains(t, endpoint.notifications, "event")
	assert.NotNil(t, endpoint.notifications["event"])
}

// TestBaseEndpoint_HandleRequest tests handling a request through an endpoint
func TestBaseEndpoint_HandleRequest(t *testing.T) {
	endpoint := NewBaseEndpoint(ToolsNamespace)

	// Register a test method
	endpoint.RegisterMethod("test", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var input map[string]string
		if err := json.Unmarshal(params, &input); err == nil {
			return map[string]string{"echo": input["value"]}, nil
		}
		return "default result", nil
	})

	// Test with valid method and parameters
	params := json.RawMessage(`{"value":"hello"}`)
	result, err := endpoint.HandleRequest(context.Background(), "test", params)

	// Verify result
	require.NoError(t, err)
	assert.NotNil(t, result)
	response, ok := result.(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "hello", response["echo"])

	// Test with unknown method
	result, err = endpoint.HandleRequest(context.Background(), "unknown", params)

	// Verify error
	assert.Error(t, err)
	assert.Nil(t, result)
	jsonRPCErr, ok := err.(*JSONRPCError)
	require.True(t, ok)
	assert.Equal(t, -32601, jsonRPCErr.Code)
}

// TestBaseEndpoint_HandleNotification tests handling a notification through an endpoint
func TestBaseEndpoint_HandleNotification(t *testing.T) {
	endpoint := NewBaseEndpoint(ToolsNamespace)

	// Keep track of which notifications were handled
	handledNotifications := make(map[string]bool)

	// Register a test notification
	endpoint.RegisterNotification("event", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		handledNotifications["event"] = true
		return nil, nil
	})

	// Test with valid notification and parameters
	params := json.RawMessage(`{"value":"notify"}`)
	result, err := endpoint.HandleNotification(context.Background(), "event", params)

	// Verify result
	assert.NoError(t, err)
	assert.Nil(t, result)
	assert.True(t, handledNotifications["event"])

	// Test with unknown notification
	result, err = endpoint.HandleNotification(context.Background(), "unknown", params)

	// Verify error
	assert.Error(t, err)
	assert.Nil(t, result)
	jsonRPCErr, ok := err.(*JSONRPCError)
	require.True(t, ok)
	assert.Equal(t, -32601, jsonRPCErr.Code)
}
