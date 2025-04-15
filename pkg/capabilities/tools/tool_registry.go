package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// ToolHandler is a function that handles tool execution
type ToolHandler func(ctx context.Context, arguments json.RawMessage) (*ToolResult, error)

// ToolRegistry manages the registration and lookup of tools
type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]*ToolWithHandler
}

// NewToolRegistry creates a new ToolRegistry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]*ToolWithHandler),
	}
}

// RegisterTool registers a tool with the registry
func (r *ToolRegistry) RegisterTool(tool *ToolWithHandler) error {
	if tool == nil {
		return fmt.Errorf("tool cannot be nil")
	}

	if tool.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	if tool.Handler == nil {
		return fmt.Errorf("tool handler cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[tool.Name]; exists {
		return fmt.Errorf("tool with name %s already exists", tool.Name)
	}

	r.tools[tool.Name] = tool
	return nil
}

// UnregisterTool removes a tool from the registry
func (r *ToolRegistry) UnregisterTool(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; !exists {
		return fmt.Errorf("tool with name %s does not exist", name)
	}

	delete(r.tools, name)
	return nil
}

// GetTool returns a tool by name
func (r *ToolRegistry) GetTool(name string) (*ToolWithHandler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool with name %s does not exist", name)
	}

	return tool, nil
}

// ListTools returns all registered tools
func (r *ToolRegistry) ListTools() []*ToolWithHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]*ToolWithHandler, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}

	return tools
}

// Count returns the number of registered tools
func (r *ToolRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tools)
}
