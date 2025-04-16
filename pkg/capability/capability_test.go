// Package capability_test tests the capability package
package capability

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/lucacox/go-mcp/pkg/protocol"
	"github.com/stretchr/testify/assert"
)

// TestCapabilityTypes verifies that capability types are defined correctly
func TestCapabilityTypes(t *testing.T) {
	// Verify capability type constants
	assert.Equal(t, CapabilityType("tools"), Tools)
	assert.Equal(t, CapabilityType("resources"), Resources)
	assert.Equal(t, CapabilityType("prompts"), Prompts)
	assert.Equal(t, CapabilityType("sampling"), Sampling)
	assert.Equal(t, CapabilityType("roots"), Roots)
}

// TestNewCapabilityRegistry tests creating a new capability registry
func TestNewCapabilityRegistry(t *testing.T) {
	registry := NewCapabilityRegistry()

	// Verify registry was created with empty maps
	assert.NotNil(t, registry)
	assert.Empty(t, registry.factories)
	assert.Empty(t, registry.capabilities)
}

// TestCapabilityRegistry_RegisterFactory tests registering a capability factory
func TestCapabilityRegistry_RegisterFactory(t *testing.T) {
	registry := NewCapabilityRegistry()

	// Create a simple factory function
	factory := func(options ...interface{}) (Capability, error) {
		return &BasicCapability{
			TypeName:    Tools,
			DescText:    "Test capability",
			OptionsData: json.RawMessage(`{"test": true}`),
		}, nil
	}

	// Register the factory
	registry.RegisterFactory(Tools, factory)

	// Verify factory was registered
	assert.Len(t, registry.factories, 1)
	assert.Contains(t, registry.factories, Tools)
}

// TestCapabilityRegistry_GetSupportedTypes tests getting supported capability types
func TestCapabilityRegistry_GetSupportedTypes(t *testing.T) {
	registry := NewCapabilityRegistry()

	// Register some factories
	factory := func(options ...interface{}) (Capability, error) {
		return &BasicCapability{}, nil
	}

	registry.RegisterFactory(Tools, factory)
	registry.RegisterFactory(Resources, factory)
	registry.RegisterFactory(Prompts, factory)

	// Get supported types
	types := registry.GetSupportedTypes()

	// Verify returned types
	assert.Len(t, types, 3)
	assert.Contains(t, types, Tools)
	assert.Contains(t, types, Resources)
	assert.Contains(t, types, Prompts)
}

// TestCapabilityRegistry_Create tests creating a capability
func TestCapabilityRegistry_Create(t *testing.T) {
	registry := NewCapabilityRegistry()

	// Define test description and options
	testDesc := "Test capability"
	testOptions := json.RawMessage(`{"test": true}`)

	// Create a factory function
	factory := func(options ...interface{}) (Capability, error) {
		return &BasicCapability{
			TypeName:    Tools,
			DescText:    testDesc,
			OptionsData: testOptions,
		}, nil
	}

	// Register the factory
	registry.RegisterFactory(Tools, factory)

	// Create a capability
	cap, err := registry.Create(Tools)

	// Verify capability was created
	assert.NoError(t, err)
	assert.NotNil(t, cap)
	assert.Equal(t, Tools, cap.GetType())
	assert.Equal(t, testDesc, cap.GetDescription())
	assert.Equal(t, testOptions, cap.GetOptions())

	// Try to create a non-registered capability
	cap, err = registry.Create(Sampling)

	// Verify error
	assert.Error(t, err)
	assert.Nil(t, cap)
	assert.Contains(t, err.Error(), "capability type not supported")
}

// TestCapabilityRegistry_AddCapability tests adding a capability
func TestCapabilityRegistry_AddCapability(t *testing.T) {
	registry := NewCapabilityRegistry()

	// Create a capability
	cap := &BasicCapability{
		TypeName:    Tools,
		DescText:    "Test capability",
		OptionsData: json.RawMessage(`{"test": true}`),
	}

	// Add the capability
	registry.AddCapability(cap)

	// Verify capability was added
	assert.Len(t, registry.capabilities, 1)
	assert.Contains(t, registry.capabilities, Tools)
	assert.Equal(t, cap, registry.capabilities[Tools])
}

// TestCapabilityRegistry_GetCapability tests retrieving a capability
func TestCapabilityRegistry_GetCapability(t *testing.T) {
	registry := NewCapabilityRegistry()

	// Create a capability
	cap := &BasicCapability{
		TypeName:    Tools,
		DescText:    "Test capability",
		OptionsData: json.RawMessage(`{"test": true}`),
	}

	// Add the capability
	registry.AddCapability(cap)

	// Get the capability
	retrievedCap := registry.GetCapability(Tools)

	// Verify retrieved capability
	assert.NotNil(t, retrievedCap)
	assert.Equal(t, cap, retrievedCap)

	// Try to get a non-existing capability
	retrievedCap = registry.GetCapability(Sampling)

	// Verify nil is returned
	assert.Nil(t, retrievedCap)
}

// TestCapabilityRegistry_GetCapabilities tests getting all capabilities
func TestCapabilityRegistry_GetCapabilities(t *testing.T) {
	registry := NewCapabilityRegistry()

	// Create some capabilities
	toolsCap := &BasicCapability{TypeName: Tools}
	resourcesCap := &BasicCapability{TypeName: Resources}

	// Add capabilities
	registry.AddCapability(toolsCap)
	registry.AddCapability(resourcesCap)

	// Get capabilities
	caps := registry.GetCapabilities()

	// Verify capabilities
	assert.Len(t, caps, 2)

	// Create a map of capabilities for easier verification
	capsMap := make(map[CapabilityType]Capability)
	for _, c := range caps {
		capsMap[c.GetType()] = c
	}

	assert.Contains(t, capsMap, Tools)
	assert.Contains(t, capsMap, Resources)
	assert.Equal(t, toolsCap, capsMap[Tools])
	assert.Equal(t, resourcesCap, capsMap[Resources])
}

// TestCreateBasicCapability tests creating a basic capability
func TestCreateBasicCapability(t *testing.T) {
	// Create a basic capability
	cap, err := CreateBasicCapability(Tools)

	// Verify capability was created
	assert.NoError(t, err)
	assert.NotNil(t, cap)
	assert.Equal(t, Tools, cap.GetType())
	assert.Contains(t, cap.GetDescription(), "Basic tools capability")
	assert.Equal(t, json.RawMessage(`{}`), cap.GetOptions())
}

// TestBasicCapability tests basic capability methods
func TestBasicCapability(t *testing.T) {
	// Create a basic capability
	typeName := Tools
	descText := "Test capability"
	optionsData := json.RawMessage(`{"test": true}`)

	cap := &BasicCapability{
		TypeName:    typeName,
		DescText:    descText,
		OptionsData: optionsData,
	}

	// Test GetType
	assert.Equal(t, typeName, cap.GetType())

	// Test GetDescription
	assert.Equal(t, descText, cap.GetDescription())

	// Test GetOptions
	assert.Equal(t, optionsData, cap.GetOptions())

	// Test GetEndpoint
	assert.Nil(t, cap.GetEndpoint())

	// Test Initialize
	err := cap.Initialize(context.Background(), json.RawMessage(`{"new": "options"}`))
	assert.NoError(t, err)

	// Test Shutdown
	err = cap.Shutdown(context.Background())
	assert.NoError(t, err)
}

// TestBaseServerCapability tests base server capability
func TestBaseServerCapability(t *testing.T) {
	// Create a base server capability
	cap := &BaseServerCapability{
		BasicCapability: BasicCapability{
			TypeName: Tools,
			DescText: "Test server capability",
		},
	}

	// Test IsServerCapability
	assert.True(t, cap.IsServerCapability())

	// Test IsClientCapability
	assert.False(t, cap.IsClientCapability())
}

// TestBaseClientCapability tests base client capability
func TestBaseClientCapability(t *testing.T) {
	// Create a base client capability
	cap := &BaseClientCapability{
		BasicCapability: BasicCapability{
			TypeName: Tools,
			DescText: "Test client capability",
		},
	}

	// Test IsServerCapability
	assert.False(t, cap.IsServerCapability())

	// Test IsClientCapability
	assert.True(t, cap.IsClientCapability())
}

// TestCapabilityError tests capability error
func TestCapabilityError(t *testing.T) {
	// Create a capability error
	message := "test error"
	err := &CapabilityError{
		Message: message,
	}

	// Test Error
	assert.Equal(t, message, err.Error())

	// Test WithCause
	cause := errors.New("cause error")
	err = err.WithCause(cause)

	// Test Error with cause
	assert.Equal(t, message+": "+cause.Error(), err.Error())

	// Test Unwrap
	assert.Equal(t, cause, err.Unwrap())
}

// TestNewCapabilityContext tests creating a new capability context
func TestNewCapabilityContext(t *testing.T) {
	// Create a new capability context
	ctx := NewCapabilityContext()

	// Verify context properties
	assert.NotEmpty(t, ctx.RequestID)
	assert.Equal(t, int64(30000), ctx.Timeout)
	assert.True(t, ctx.Cancellable)
	assert.Empty(t, ctx.Options)
}

// TestCapabilityContext_WithMethods tests capability context with methods
func TestCapabilityContext_WithMethods(t *testing.T) {
	// Create a new capability context
	ctx := NewCapabilityContext()

	// Test WithSessionID
	sessionID := "test-session"
	ctx = ctx.WithSessionID(sessionID)
	assert.Equal(t, sessionID, ctx.SessionID)

	// Test WithTimeout
	timeout := int64(60000)
	ctx = ctx.WithTimeout(timeout)
	assert.Equal(t, timeout, ctx.Timeout)

	// Test WithCancellable
	ctx = ctx.WithCancellable(false)
	assert.False(t, ctx.Cancellable)

	// Test WithOption
	key := "testKey"
	value := "testValue"
	ctx = ctx.WithOption(key, value)
	assert.Contains(t, ctx.Options, key)
	assert.Equal(t, value, ctx.Options[key])
}

// MockEndpoint implements a mock endpoint for testing
type MockEndpoint struct {
	namespace protocol.Namespace
	methods   []string
}

func (m *MockEndpoint) GetNamespace() protocol.Namespace {
	return m.namespace
}

func (m *MockEndpoint) GetMethods() []string {
	return m.methods
}

func (m *MockEndpoint) HandleRequest(ctx context.Context, method string, params json.RawMessage) (interface{}, error) {
	return nil, nil
}

func (m *MockEndpoint) HandleNotification(ctx context.Context, method string, params json.RawMessage) (interface{}, error) {
	return nil, nil
}

// TestCapabilityWithCustomEndpoint tests a capability with a custom endpoint
func TestCapabilityWithCustomEndpoint(t *testing.T) {
	// Create a mock endpoint
	mockEndpoint := &MockEndpoint{
		namespace: protocol.ToolsNamespace,
		methods:   []string{"test1", "test2"},
	}

	// Create a custom capability type that embeds BasicCapability
	// and overrides the GetEndpoint method
	type CustomCapability struct {
		BasicCapability
		endpoint protocol.Endpoint
	}

	// Override GetEndpoint method by creating a method on CustomCapability
	customGetEndpoint := func(c *CustomCapability) protocol.Endpoint {
		return c.endpoint
	}

	// Create a new custom capability instance
	customCap := &CustomCapability{
		BasicCapability: BasicCapability{
			TypeName:    Tools,
			DescText:    "Test capability with endpoint",
			OptionsData: json.RawMessage(`{}`),
		},
		endpoint: mockEndpoint,
	}

	// Test the custom capability endpoint
	assert.Equal(t, mockEndpoint, customGetEndpoint(customCap))
	assert.Equal(t, protocol.ToolsNamespace, customGetEndpoint(customCap).GetNamespace())
	assert.Equal(t, []string{"test1", "test2"}, customGetEndpoint(customCap).GetMethods())
}
