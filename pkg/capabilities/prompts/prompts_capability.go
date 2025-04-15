// Package prompts provides the prompts capability implementation
package prompts

import (
	"context"
	"encoding/json"

	"github.com/lucacox/go-mcp/pkg/capability"
)

// PromptsCapability represents the Prompts server capability
type PromptsCapability struct {
	capability.BaseServerCapability
	ListChanged bool
}

func PromptsCapabilityFactory(options ...interface{}) (capability.Capability, error) {
	listChanged := false
	ok := false
	if len(options) > 0 {
		listChanged, ok = options[0].(bool)
		if !ok {
			listChanged = false
		}
	}
	return NewPromptsCapability(listChanged), nil
}

// NewPromptsCapability creates a new instance of PromptsCapability
func NewPromptsCapability(listChanged bool) *PromptsCapability {
	pc := &PromptsCapability{
		ListChanged: listChanged,
	}
	pc.TypeName = capability.Prompts
	pc.DescText = "Prompts capability implementation"

	options := map[string]interface{}{
		"listChanged": listChanged,
	}
	optionsJSON, _ := json.Marshal(options)
	pc.OptionsData = optionsJSON

	return pc
}

// SupportsListChangedNotifications indicates if the capability supports list change notifications
func (p *PromptsCapability) SupportsListChangedNotifications() bool {
	return p.ListChanged
}

// NotifyListChanged sends a notification that the prompts list has changed
func (p *PromptsCapability) NotifyListChanged(ctx context.Context) error {
	// Actual implementation to be handled via events or callbacks
	return nil
}

// Initialize initializes the capability with the provided options
func (p *PromptsCapability) Initialize(ctx context.Context, options json.RawMessage) error {
	if len(options) == 0 {
		return nil
	}

	var opts struct {
		ListChanged bool `json:"listChanged"`
	}

	if err := json.Unmarshal(options, &opts); err != nil {
		return &capability.CapabilityError{
			Message: "failed to parse prompts capability options",
			Cause:   err,
		}
	}

	p.ListChanged = opts.ListChanged
	return nil
}

// Shutdown performs any necessary cleanup when the capability is no longer needed
func (p *PromptsCapability) Shutdown(ctx context.Context) error {
	// No special cleanup needed for this capability
	return nil
}
