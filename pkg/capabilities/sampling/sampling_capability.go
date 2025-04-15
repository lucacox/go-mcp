// Package sampling provides the sampling capability implementation
package sampling

import (
	"context"
	"encoding/json"

	"github.com/lucacox/go-mcp/pkg/capability"
)

// SamplingCapability represents the Sampling client capability
type SamplingCapability struct {
	capability.BaseClientCapability
}

// SamplingCapabilityFactory creates a new instance of SamplingCapability
// with no options
func SamplingCapabilityFactory(options ...interface{}) (capability.Capability, error) {
	return NewSamplingCapability(), nil
}

// NewSamplingCapability creates a new instance of SamplingCapability
func NewSamplingCapability() *SamplingCapability {
	sc := &SamplingCapability{}
	sc.TypeName = capability.Sampling
	sc.DescText = "Sampling capability implementation"
	sc.OptionsData = json.RawMessage(`{}`)
	return sc
}

// Initialize initializes the capability with the provided options
func (s *SamplingCapability) Initialize(ctx context.Context, options json.RawMessage) error {
	// No configuration options for this capability
	return nil
}

// Shutdown performs any necessary cleanup when the capability is no longer needed
func (s *SamplingCapability) Shutdown(ctx context.Context) error {
	// No special cleanup needed for this capability
	return nil
}
