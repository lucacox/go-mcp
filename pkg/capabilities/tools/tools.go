// Package tools provides the tools capability implementation
package tools

import (
	"github.com/lucacox/go-mcp/pkg/protocol"
)

// ToolWithHandler extends the protocol.Tool definition with a handler function
type ToolWithHandler struct {
	// Embed the protocol.Tool to inherit all its fields
	protocol.Tool

	// Handler is the function that executes the tool
	Handler ToolHandler `json:"-"` // Not serialized
}

func NewTool(name, description string, inputSchema *protocol.JSONSchema, annotations map[string]interface{}, handler ToolHandler) *ToolWithHandler {
	return &ToolWithHandler{
		Tool: protocol.Tool{
			Name:        name,
			Description: description,
			InputSchema: inputSchema,
			Annotations: annotations,
		},
		Handler: handler,
	}
}

// ContentType represents the type of content in a tool result
type ContentType string

const (
	// ContentTypeText represents text content in a tool result
	ContentTypeText ContentType = "text"

	// ContentTypeImage represents image content in a tool result
	ContentTypeImage ContentType = "image"

	// ContentTypeAudio represents audio content in a tool result
	ContentTypeAudio ContentType = "audio"

	// ContentTypeResource represents resource content in a tool result
	ContentTypeResource ContentType = "resource"
)

// ContentItem represents a content item in a tool result
type ContentItem struct {
	// Type is the type of content
	Type ContentType `json:"type"`

	// Text is the text content (for ContentTypeText)
	Text string `json:"text,omitempty"`

	// Data is the base64-encoded data (for ContentTypeImage and ContentTypeAudio)
	Data string `json:"data,omitempty"`

	// MimeType is the MIME type of the content
	MimeType string `json:"mimeType,omitempty"`

	// Resource is the embedded resource (for ContentTypeResource)
	Resource *Resource `json:"resource,omitempty"`
}

// Resource represents an embedded resource in a tool result
type Resource struct {
	// URI is the unique identifier for the resource
	URI string `json:"uri"`

	// MimeType is the MIME type of the resource
	MimeType string `json:"mimeType"`

	// Text is the text content of the resource
	Text string `json:"text,omitempty"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	// Content contains the content items returned by the tool
	Content []ContentItem `json:"content"`

	// IsError indicates whether the tool execution resulted in an error
	IsError bool `json:"isError"`
}

// NewTextContent creates a new text content item
func NewTextContent(text string) ContentItem {
	return ContentItem{
		Type: ContentTypeText,
		Text: text,
	}
}

// NewImageContent creates a new image content item
func NewImageContent(data string, mimeType string) ContentItem {
	return ContentItem{
		Type:     ContentTypeImage,
		Data:     data,
		MimeType: mimeType,
	}
}

// NewAudioContent creates a new audio content item
func NewAudioContent(data string, mimeType string) ContentItem {
	return ContentItem{
		Type:     ContentTypeAudio,
		Data:     data,
		MimeType: mimeType,
	}
}

// NewResourceContent creates a new resource content item
func NewResourceContent(uri string, mimeType string, text string) ContentItem {
	return ContentItem{
		Type: ContentTypeResource,
		Resource: &Resource{
			URI:      uri,
			MimeType: mimeType,
			Text:     text,
		},
	}
}

// NewToolResult creates a new tool result
func NewToolResult(content []ContentItem, isError bool) *ToolResult {
	return &ToolResult{
		Content: content,
		IsError: isError,
	}
}

// NewErrorToolResult creates a new error tool result with a text message
func NewErrorToolResult(errorMessage string) *ToolResult {
	return &ToolResult{
		Content: []ContentItem{
			NewTextContent(errorMessage),
		},
		IsError: true,
	}
}

// NewSuccessToolResult creates a new success tool result with a text message
func NewSuccessToolResult(message string) *ToolResult {
	return &ToolResult{
		Content: []ContentItem{
			NewTextContent(message),
		},
		IsError: false,
	}
}
