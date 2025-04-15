// Package roots provides the roots capability implementation
package roots

import (
	"context"
	"encoding/json"

	"github.com/lucacox/go-mcp/pkg/capability"
)

// RootsCapability represents the Roots client capability
type RootsCapability struct {
	capability.BaseClientCapability
	ListChanged bool
}

// RootsCapabilityFactory creates a new instance of RootsCapability
// with the provided options:
// 1. listChanged: indicates if the capability supports list change notifications
func RootsCapabilityFactory(options ...interface{}) (capability.Capability, error) {
	listChanged := false
	ok := false
	if len(options) > 0 {
		listChanged, ok = options[0].(bool)
		if !ok {
			listChanged = false
		}
	}
	return NewRootsCapability(listChanged), nil
}

// NewRootsCapability creates a new instance of RootsCapability
func NewRootsCapability(listChanged bool) *RootsCapability {
	rc := &RootsCapability{
		ListChanged: listChanged,
	}
	rc.TypeName = capability.Roots
	rc.DescText = "Roots capability implementation"

	options := map[string]interface{}{
		"listChanged": listChanged,
	}
	optionsJSON, _ := json.Marshal(options)
	rc.OptionsData = optionsJSON

	return rc
}

// SupportsListChangedNotifications indicates if the capability supports list change notifications
func (r *RootsCapability) SupportsListChangedNotifications() bool {
	return r.ListChanged
}

// NotifyListChanged sends a notification that the roots list has changed
func (r *RootsCapability) NotifyListChanged(ctx context.Context) error {
	// Actual implementation to be handled via events or callbacks
	return nil
}

// Initialize initializes the capability with the provided options
func (r *RootsCapability) Initialize(ctx context.Context, options json.RawMessage) error {
	if len(options) == 0 {
		return nil
	}

	var opts struct {
		ListChanged bool `json:"listChanged"`
	}

	if err := json.Unmarshal(options, &opts); err != nil {
		return &capability.CapabilityError{
			Message: "failed to parse roots capability options",
			Cause:   err,
		}
	}

	r.ListChanged = opts.ListChanged
	return nil
}

// Shutdown performs any necessary cleanup when the capability is no longer needed
func (r *RootsCapability) Shutdown(ctx context.Context) error {
	// No special cleanup needed for this capability
	return nil
}
