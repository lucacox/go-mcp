package server

import (
	"log/slog"
	"slices"

	"github.com/lucacox/go-mcp/internal/logging"
	"github.com/lucacox/go-mcp/pkg/capabilities/tools"
	"github.com/lucacox/go-mcp/pkg/capability"
	"github.com/lucacox/go-mcp/pkg/protocol"
	httptransport "github.com/lucacox/go-mcp/pkg/transport/http"
	stdiotransport "github.com/lucacox/go-mcp/pkg/transport/stdio"
)

// ServerOption is a function that configures a server
type ServerOption func(*Server)

// WithServerID sets the server ID
func WithServerID(id string) ServerOption {
	return func(s *Server) {
		s.ID = id
	}
}

// WithServerInfo sets the server information
func WithServerInfo(info map[string]string) ServerOption {
	return func(s *Server) {
		for k, v := range info {
			s.Info[k] = v
		}
	}
}

// WithServerName sets the server name
func WithServerName(name string) ServerOption {
	return func(s *Server) {
		s.Name = name
	}
}

// WithServerVersion sets the server version
func WithServerVersion(version string) ServerOption {
	return func(s *Server) {
		s.Version = version
	}
}

// WithLogger sets the logger for the server
func WithLogger(level slog.Level) ServerOption {
	lf := logging.NewLoggerFactory()
	lf.SetLevel(level)
	return func(s *Server) {
		s.loggerFactory = lf
	}
}

// WithTransportRegistry specifies a custom transport registry to use
func WithTransportRegistry(registry *protocol.TransportRegistry) ServerOption {
	return func(s *Server) {
		s.transportRegistry = registry
	}
}

// WithTransports specifies which transports the server should support
// If no transports are specified, all available transports from the DefaultTransportRegistry will be used
func WithTransports(transportTypes ...string) ServerOption {
	return func(s *Server) {
		if s.transportRegistry == nil {
			s.transportRegistry = protocol.DefaultTransportRegistry
		}

		for _, trsType := range transportTypes {
			if trsType == protocol.TransportTypeStdio {
				s.transportRegistry.Register(protocol.TransportTypeStdio, stdiotransport.STDIOTransportCreator)
			}
			if trsType == protocol.TransportTypeHTTP {
				s.transportRegistry.Register(protocol.TransportTypeHTTP, httptransport.HTTPTransportCreator)
			}
		}
	}
}

// WithCapability adds a capability to the server
// options are optional and can be used to pass additional parameters to the capability constructor:
//   - Tools: options[0] = listChanged
//   - Resources: options[0] = listChanged, options[1] = subscribe
//   - Prompts: options[0] = listChanged
func WithCapability(capType capability.CapabilityType, options ...interface{}) ServerOption {
	return func(s *Server) {
		cap, err := s.capabilityRegistry.Create(capType, options...)
		if err == nil {
			s.capabilityRegistry.AddCapability(cap)
			if cap.GetEndpoint() != nil {
				s.endpointRegistry.RegisterEndpoint(cap.GetEndpoint())
			}
		} else {
			logging.Error(s.logger, "Failed to create capability", "capType", capType, "error", err)
		}
	}
}

func WithProtocolVersion(version protocol.ProtocolVersion) ServerOption {
	return func(s *Server) {
		idx := slices.Index(s.SupportedVersions, version)
		if idx == -1 {
			s.SupportedVersions = append(s.SupportedVersions, version)
		}
	}
}

func WithTool(tool *tools.ToolWithHandler) ServerOption {
	return func(s *Server) {
		cap := s.capabilityRegistry.GetCapability(capability.Tools)
		if cap == nil {
			WithCapability(capability.Tools)(s)
			cap = s.capabilityRegistry.GetCapability(capability.Tools)
		}
		if toolsCap, ok := cap.(*tools.ToolsCapability); ok {
			toolsCap.RegisterTool(tool)
		} else {
			logging.Error(s.logger, "Failed to cast capability", "capType", capability.Tools)
		}
	}
}
