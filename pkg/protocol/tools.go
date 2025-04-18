package protocol

// Tool represents a tool definition as defined in the MCP specification
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema *JSONSchema            `json:"inputSchema,omitempty"`
	Annotations map[string]interface{} `json:"annotations,omitempty"`
}

// ToolsListParams represents the parameters for a tools/list request
type ToolsListParams struct {
	Cursor string `json:"cursor,omitempty"`
}

// ToolsListResult represents the result of a tools/list request
type ToolsListResult struct {
	Tools      []Tool `json:"tools"`
	NextCursor string `json:"nextCursor,omitempty"`
}

// ToolsCallParams represents the parameters for a tools/call request
type ToolsCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResultContent represents a content item in a tool result
type ToolResultContent struct {
	Type     string           `json:"type"`
	Text     string           `json:"text,omitempty"`
	Data     string           `json:"data,omitempty"`
	MimeType string           `json:"mimeType,omitempty"`
	Resource *ResourceContent `json:"resource,omitempty"`
}

// ResourceContent represents an embedded resource in a tool result
type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text,omitempty"`
}

// ToolsCallResult represents the result of a tools/call request
type ToolsCallResult struct {
	Content []ToolResultContent `json:"content"`
	IsError bool                `json:"isError"`
}

// ToolsListChangedParams represents the parameters for a tools/list_changed notification
type ToolsListChangedParams struct {
	// Empty in the current specification
}
