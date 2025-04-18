// Package tools provides the tools capability implementation
package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/lucacox/go-mcp/pkg/capability"
	"github.com/lucacox/go-mcp/pkg/protocol"
)

// ToolsCapability represents the Tools server capability
type ToolsCapability struct {
	capability.BaseServerCapability
	ListChanged bool
	registry    *ToolRegistry
	callbacks   ToolsCapabilityCallbacks
	// Endpoint associated with this capability
	endpoint *ToolsEndpoint
	// Active Session that this capability is associated with
	sessions   map[string]*protocol.Session
	sessionsMu sync.RWMutex
}

// ToolsCapabilityCallbacks defines the callbacks for the tools capability
type ToolsCapabilityCallbacks struct {
	// OnListChanged is called when the tools list changes
	OnListChanged func(ctx context.Context) error
	// OnToolsRequest is called when a tools/list request is received
	OnToolsRequest func(ctx context.Context, params protocol.ToolsListParams) (*protocol.ToolsListResult, error)
	// OnToolsCall is called when a tools/call request is received
	OnToolsCall func(ctx context.Context, params protocol.ToolsCallParams) (*protocol.ToolsCallResult, error)
}

// ToolsCapabilityOption defines a function that configures a ToolsCapability
type ToolsCapabilityOption func(*ToolsCapability)

// WithListChangedNotifications configures whether the capability supports list change notifications
func WithListChangedNotifications(listChanged bool) ToolsCapabilityOption {
	return func(tc *ToolsCapability) {
		tc.ListChanged = listChanged
	}
}

// WithToolsCapabilityCallbacks configures the callbacks for the tools capability
func WithToolsCapabilityCallbacks(callbacks ToolsCapabilityCallbacks) ToolsCapabilityOption {
	return func(tc *ToolsCapability) {
		tc.callbacks = callbacks
	}
}

// WithSessionRegistry configura il registro delle sessioni per la capability
func WithSessionRegistry(server interface{}) ToolsCapabilityOption {
	return func(tc *ToolsCapability) {
		// Questo verrà impostato quando la capability viene attivata dal server
		tc.sessions = make(map[string]*protocol.Session)
	}
}

// ToolsCapabilityFactory creates a new instance of ToolsCapability
// with the provided options:
// 1. listChanged: indicates if the capability supports list change notifications
func ToolsCapabilityFactory(options ...interface{}) (capability.Capability, error) {
	listChanged := false
	ok := false
	if len(options) > 0 {
		listChanged, ok = options[0].(bool)
		if !ok {
			listChanged = false
		}
	}
	return NewToolsCapability(WithListChangedNotifications(listChanged)), nil
}

// NewToolsCapability creates a new instance of ToolsCapability with the provided options
func NewToolsCapability(options ...ToolsCapabilityOption) *ToolsCapability {
	tc := &ToolsCapability{
		registry: NewToolRegistry(),
	}
	tc.TypeName = capability.Tools
	tc.DescText = "Tools capability implementation"
	tc.endpoint = NewToolsEndpoint(tc)

	// Apply options
	for _, option := range options {
		option(tc)
	}

	// Create a proper JSON object with listChanged as a field
	optionsMap := map[string]interface{}{
		"listChanged": tc.ListChanged,
	}
	optionsJSON, _ := json.Marshal(optionsMap)
	tc.OptionsData = optionsJSON

	return tc
}

func (t *ToolsCapability) GetEndpoint() protocol.Endpoint {
	return t.endpoint
}

// SupportsListChangedNotifications indicates if the capability supports list change notifications
func (t *ToolsCapability) SupportsListChangedNotifications() bool {
	return t.ListChanged
}

// NotifyListChanged sends a notification that the tools list has changed
func (t *ToolsCapability) NotifyListChanged(ctx context.Context) error {
	// Se abbiamo un callback personalizzato, utilizzalo
	if t.callbacks.OnListChanged != nil {
		if err := t.callbacks.OnListChanged(ctx); err != nil {
			return err
		}
	}

	// Se non abbiamo un endpoint, non possiamo inviare notifiche
	if t.endpoint == nil {
		return nil
	}

	// Invia una notifica a tutte le sessioni attive
	var lastErr error

	for _, session := range t.GetActiveSessions() {
		if err := t.endpoint.SendListChangedNotification(ctx, session); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// Initialize initializes the capability with the provided options
func (t *ToolsCapability) Initialize(ctx context.Context, options json.RawMessage) error {
	if len(options) == 0 {
		return nil
	}

	var opts struct {
		ListChanged bool `json:"listChanged"`
	}

	if err := json.Unmarshal(options, &opts); err != nil {
		return &capability.CapabilityError{
			Message: "failed to parse tools capability options",
			Cause:   err,
		}
	}

	t.ListChanged = opts.ListChanged
	return nil
}

// Shutdown performs any necessary cleanup when the capability is no longer needed
func (t *ToolsCapability) Shutdown(ctx context.Context) error {
	// No special cleanup needed for this capability
	return nil
}

// RegisterTool registers a tool with the capability
func (t *ToolsCapability) RegisterTool(tool *ToolWithHandler) error {
	err := t.registry.RegisterTool(tool)
	if err != nil {
		return err
	}

	// Notify listeners that the tools list has changed
	if t.ListChanged {
		// Execute in a goroutine to avoid blocking
		go func() {
			ctx := context.Background()
			_ = t.NotifyListChanged(ctx)
		}()
	}

	return nil
}

// UnregisterTool removes a tool from the capability
func (t *ToolsCapability) UnregisterTool(name string) error {
	err := t.registry.UnregisterTool(name)
	if err != nil {
		return err
	}

	// Notify listeners that the tools list has changed
	if t.ListChanged {
		// Execute in a goroutine to avoid blocking
		go func() {
			ctx := context.Background()
			_ = t.NotifyListChanged(ctx)
		}()
	}

	return nil
}

// GetTool returns a tool by name
func (t *ToolsCapability) GetTool(name string) (*ToolWithHandler, error) {
	return t.registry.GetTool(name)
}

// ListTools returns all registered tools
func (t *ToolsCapability) ListTools() []*ToolWithHandler {
	return t.registry.ListTools()
}

// CountTools returns the number of registered tools
func (t *ToolsCapability) CountTools() int {
	return t.registry.Count()
}

// HandleToolsList handles the tools/list request
func (t *ToolsCapability) HandleToolsList(ctx context.Context, params protocol.ToolsListParams) (*protocol.ToolsListResult, error) {
	// If a custom handler is provided, use that
	if t.callbacks.OnToolsRequest != nil {
		return t.callbacks.OnToolsRequest(ctx, params)
	}

	// Otherwise use the default implementation
	tools := t.ListTools()

	// Convert to the format expected by the protocol
	resultTools := make([]protocol.Tool, 0, len(tools))
	for _, tool := range tools {
		// Just copy the embedded Tool struct directly
		resultTools = append(resultTools, tool.Tool)
	}

	// Simple implementation without pagination
	result := &protocol.ToolsListResult{
		Tools: resultTools,
		// NextCursor is empty as we're not implementing pagination in this version
	}

	return result, nil
}

// HandleToolsCall handles the tools/call request
func (t *ToolsCapability) HandleToolsCall(ctx context.Context, params protocol.ToolsCallParams) (*protocol.ToolsCallResult, error) {
	// If a custom handler is provided, use that
	if t.callbacks.OnToolsCall != nil {
		return t.callbacks.OnToolsCall(ctx, params)
	}

	// Otherwise use the default implementation
	tool, err := t.GetTool(params.Name)
	if err != nil {
		return nil, &protocol.JSONRPCError{
			Code:    protocol.ErrorCodeInvalidRequest,
			Message: fmt.Sprintf("Unknown tool: %s", params.Name),
		}
	}

	// Check if the tool has a handler
	if tool.Handler == nil {
		return nil, &protocol.JSONRPCError{
			Code:    protocol.ErrorCodeInternalError,
			Message: fmt.Sprintf("Tool %s has no handler", params.Name),
		}
	}

	// Execute the tool
	arguments, err := json.Marshal(params.Arguments)
	if err != nil {
		return nil, protocol.NewJSONRPCError(protocol.ErrorCodeInvalidParams, "Invalid arguments format", err)
	}

	result, err := tool.Handler(ctx, arguments)
	if err != nil {
		// Handle protocol errors
		var jsonrpcErr *protocol.JSONRPCError
		if errors.As(err, &jsonrpcErr) {
			return nil, jsonrpcErr
		}

		// Return a generic error
		return nil, protocol.NewJSONRPCError(protocol.ErrorCodeInternalError, "Tool execution failed", err)
	}

	if result == nil {
		result = NewSuccessToolResult("")
	}

	// Convert to protocol format
	protocolContent := make([]protocol.ToolResultContent, 0, len(result.Content))
	for _, content := range result.Content {
		var protocolContentItem protocol.ToolResultContent

		switch content.Type {
		case ContentTypeText:
			protocolContentItem = protocol.ToolResultContent{
				Type: string(content.Type),
				Text: content.Text,
			}
		case ContentTypeImage, ContentTypeAudio:
			protocolContentItem = protocol.ToolResultContent{
				Type:     string(content.Type),
				Data:     content.Data,
				MimeType: content.MimeType,
			}
		case ContentTypeResource:
			if content.Resource != nil {
				protocolContentItem = protocol.ToolResultContent{
					Type: string(content.Type),
					Resource: &protocol.ResourceContent{
						URI:      content.Resource.URI,
						MimeType: content.Resource.MimeType,
						Text:     content.Resource.Text,
					},
				}
			}
		}

		protocolContent = append(protocolContent, protocolContentItem)
	}

	return &protocol.ToolsCallResult{
		Content: protocolContent,
		IsError: result.IsError,
	}, nil
}

// RegisterSession registra una sessione con questa capability
func (t *ToolsCapability) RegisterSession(session *protocol.Session) {
	t.sessionsMu.Lock()
	defer t.sessionsMu.Unlock()

	t.sessions[session.ID] = session
}

// UnregisterSession rimuove una sessione da questa capability
func (t *ToolsCapability) UnregisterSession(sessionID string) {
	t.sessionsMu.Lock()
	defer t.sessionsMu.Unlock()

	delete(t.sessions, sessionID)
}

// GetActiveSessions restituisce tutte le sessioni attive
func (t *ToolsCapability) GetActiveSessions() []*protocol.Session {
	t.sessionsMu.RLock()
	defer t.sessionsMu.RUnlock()

	activeSessions := make([]*protocol.Session, 0, len(t.sessions))
	for _, session := range t.sessions {
		if session.IsActive() {
			activeSessions = append(activeSessions, session)
		}
	}

	return activeSessions
}
