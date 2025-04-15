package tools

import (
	"context"
	"encoding/json"

	"github.com/lucacox/go-mcp/pkg/protocol"
)

// ToolsEndpoint represents an endpoint that handles tools capability methods
type ToolsEndpoint struct {
	protocol.BaseEndpoint
	capability *ToolsCapability
}

// NewToolsEndpoint creates a new tools capability endpoint
func NewToolsEndpoint(capability *ToolsCapability) *ToolsEndpoint {
	endpoint := &ToolsEndpoint{
		BaseEndpoint: *protocol.NewBaseEndpoint(protocol.ToolsNamespace),
		capability:   capability,
	}

	// Register methods
	endpoint.RegisterMethod("list", endpoint.handleList)
	endpoint.RegisterMethod("call", endpoint.handleCall)

	// Register notifications
	if capability.SupportsListChangedNotifications() {
		endpoint.RegisterNotification("list_changed", endpoint.handleListChanged)
	}

	return endpoint
}

// handleList handles the tools/list request
func (e *ToolsEndpoint) handleList(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var listParams protocol.ToolsListParams
	if err := json.Unmarshal(params, &listParams); err != nil {
		return nil, &protocol.JSONRPCError{
			Code:    protocol.ErrorCodeInvalidParams,
			Message: "Invalid parameters: " + err.Error(),
		}
	}

	return e.capability.HandleToolsList(ctx, listParams)
}

// handleCall handles the tools/call request
func (e *ToolsEndpoint) handleCall(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var callParams protocol.ToolsCallParams
	if err := json.Unmarshal(params, &callParams); err != nil {
		return nil, &protocol.JSONRPCError{
			Code:    protocol.ErrorCodeInvalidParams,
			Message: "Invalid parameters: " + err.Error(),
		}
	}

	return e.capability.HandleToolsCall(ctx, callParams)
}

// handleListChanged handles the notifications/tools/list_changed notification
func (e *ToolsEndpoint) handleListChanged(ctx context.Context, params json.RawMessage) (interface{}, error) {
	// No-op for notifications (server won't receive this notification)
	return nil, nil
}

// SendListChangedNotification sends a notification that the tools list has changed
func (e *ToolsEndpoint) SendListChangedNotification(ctx context.Context, session *protocol.Session) error {
	// Create empty parameters for the notification
	params := protocol.ToolsListChangedParams{}

	// Send the notification
	return session.Notify(ctx, "notifications/tools/list_changed", params)
}
