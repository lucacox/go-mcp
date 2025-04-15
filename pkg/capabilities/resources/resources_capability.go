// Package resources provides the resources capability implementation
package resources

import (
	"context"
	"encoding/json"

	"github.com/lucacox/go-mcp/pkg/capability"
)

// ResourcesCapability represents the Resources server capability
type ResourcesCapability struct {
	capability.BaseServerCapability
	ListChanged bool
	Subscribe   bool
}

// ResourcesCapabilityFactory creates a new instance of ResourcesCapability
// with the provided options:
// 1. listChanged: indicates if the capability supports list change notifications
// 2. subscribe: indicates if the capability supports subscription
func ResourcesCapabilityFactory(options ...interface{}) (capability.Capability, error) {
	listChanged := false
	subscribe := false
	ok := false
	if len(options) > 0 {
		listChanged, ok = options[0].(bool)
		if !ok {
			listChanged = false
		}
	}
	if len(options) > 1 {
		subscribe, ok = options[1].(bool)
		if !ok {
			subscribe = false
		}
	}
	return NewResourcesCapability(listChanged, subscribe), nil
}

// NewResourcesCapability creates a new instance of ResourcesCapability
func NewResourcesCapability(listChanged bool, subscribe bool) *ResourcesCapability {
	rc := &ResourcesCapability{
		ListChanged: listChanged,
		Subscribe:   subscribe,
	}
	rc.TypeName = capability.Resources
	rc.DescText = "Resources capability implementation"

	options := map[string]interface{}{
		"listChanged": listChanged,
		"subscribe":   subscribe,
	}
	optionsJSON, _ := json.Marshal(options)
	rc.OptionsData = optionsJSON

	return rc
}

// SupportsListChangedNotifications indicates if the capability supports list change notifications
func (r *ResourcesCapability) SupportsListChangedNotifications() bool {
	return r.ListChanged
}

// SupportsSubscription indicates if the capability supports subscription
func (r *ResourcesCapability) SupportsSubscription() bool {
	return r.Subscribe
}

// NotifyListChanged sends a notification that the resources list has changed
func (r *ResourcesCapability) NotifyListChanged(ctx context.Context) error {
	// Actual implementation to be handled via events or callbacks
	return nil
}

// Initialize initializes the capability with the provided options
func (r *ResourcesCapability) Initialize(ctx context.Context, options json.RawMessage) error {
	if len(options) == 0 {
		return nil
	}

	var opts struct {
		ListChanged bool `json:"listChanged"`
		Subscribe   bool `json:"subscribe"`
	}

	if err := json.Unmarshal(options, &opts); err != nil {
		return &capability.CapabilityError{
			Message: "failed to parse resources capability options",
			Cause:   err,
		}
	}

	r.ListChanged = opts.ListChanged
	r.Subscribe = opts.Subscribe
	return nil
}

// Shutdown performs any necessary cleanup when the capability is no longer needed
func (r *ResourcesCapability) Shutdown(ctx context.Context) error {
	// No special cleanup needed for this capability
	return nil
}
