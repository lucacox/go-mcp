package client

import (
	"log/slog"

	"github.com/lucacox/go-mcp/internal/logging"
	"github.com/lucacox/go-mcp/pkg/capability"
	"github.com/lucacox/go-mcp/pkg/protocol"
)

// ClientOption is a function that configures a client
type ClientOption func(*Client)

// WithClientID sets the client ID
func WithClientID(id string) ClientOption {
	return func(c *Client) {
		c.ID = id
	}
}

// WithClientInfo sets the client information
func WithClientInfo(info map[string]string) ClientOption {
	return func(c *Client) {
		for k, v := range info {
			c.Info[k] = v
		}
	}
}

// WithLogger sets the logger for the client
func WithLogger(level slog.Level) ClientOption {
	lf := logging.NewLoggerFactory()
	lf.SetLevel(level)
	return func(c *Client) {
		c.loggerFactory = lf
	}
}

// WithTransportRegistry sets a custom transport registry for the client
func WithTransportRegistry(registry *protocol.TransportRegistry) ClientOption {
	return func(c *Client) {
		c.transportRegistry = registry
	}
}

// WithCapability adds a capability to the client
func WithCapability(capType capability.CapabilityType, options ...interface{}) ClientOption {
	return func(c *Client) {
		cap, err := c.capabilityRegistry.Create(capType, options...)
		if err == nil {
			c.capabilityRegistry.AddCapability(cap)
		} else {
			logging.Error(c.logger, "Error creating capability", "error", err)
		}
	}
}
