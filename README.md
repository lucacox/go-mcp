# Go-MCP: Model Context Protocol Implementation in Go

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Reference](https://pkg.go.dev/badge/github.com/lucacox/go-mcp.svg)](https://pkg.go.dev/github.com/lucacox/go-mcp)
[![Go Report Card](https://goreportcard.com/badge/github.com/lucacox/go-mcp)](https://goreportcard.com/report/github.com/lucacox/go-mcp)

Go-MCP is a Go implementation of the [Model Context Protocol (MCP)](https://github.com/microsoft/model-context-protocol), a standardized protocol for communication between AI models and external context providers. This implementation enables developers to create MCP-compatible servers and clients in Go, allowing integration with various AI model providers.

## Table of Contents
- [Project Structure](#project-structure)
- [Architecture](#architecture)
- [Implementation Status](#implementation-status)
- [Getting Started](#getting-started)
- [Advanced Usage](#advanced-usage)
- [Contributing](#contributing)
- [License](#license)

## Project Structure

```
go-mcp/
├── cmd/                  # Command-line applications
│   └── examples/         # Example applications
│       └── main.go       # Main example server
├── docs/                 # Documentation
├── internal/             # Internal packages (not exported)
│   ├── errors/           # Error utilities
│   └── logging/          # Logging utilities
├── pkg/                  # Public packages
│   ├── capabilities/     # MCP capability implementations
│   │   ├── prompts/      # Prompts capability
│   │   ├── resources/    # Resources capability
│   │   ├── roots/        # Roots capability
│   │   ├── sampling/     # Sampling capability
│   │   └── tools/        # Tools capability
│   ├── capability/       # Base capability abstractions
│   ├── client/           # MCP client implementation
│   ├── protocol/         # Protocol definitions and logic
│   ├── server/           # Server implementation
│   └── transport/        # Transport layer implementations
│       ├── http/         # HTTP transport
│       └── stdio/        # Standard I/O transport
└── test/                 # Test resources
    ├── initialize.json   # Sample initialize request
    ├── initialized.json  # Sample initialized response
    ├── ping.json         # Sample ping request
    ├── tools_call.json   # Sample tool call request
    └── tools_list.json   # Sample tools listing request
```

## Architecture

Go-MCP follows a layered architecture with clean separation of concerns:

### 1. Transport Layer

The transport layer is responsible for handling the communication mechanism between clients and the server:

- `StdioTransport`: For standard input/output communication
- `HTTPTransport`: For HTTP-based communication with Server-Sent Events (SSE) for notifications

Each transport implements the `Transport` interface, allowing for easy addition of new transport types.

### 2. Protocol Layer

The protocol layer processes JSON-RPC messages according to the MCP specification:

- Request/response handling
- Notification processing
- Error handling and propagation
- Batch message processing
- Session management

### 3. Server Layer

The server layer provides the core MCP server functionality:

- Client state management (initialization, shutdown)
- Message routing to appropriate handlers
- Capability advertisement and negotiation
- Server configuration via builder pattern or options

### 4. Capability Layer

Capabilities are modular components that implement specific MCP functionality:

- `Tools`: Registers and executes tools that extend the model's capabilities
- `Resources`: Provides additional context from external resources
- `Prompts`: Manages prompt templates
- `Roots`: Manages root items for large content sections
- `Sampling`: Captures and reports model behavior statistics

### 5. Client Layer

Provides a client implementation for connecting to MCP servers:

- Connection management
- Request/response handling
- Notification handling
- Error handling

## Implementation Status

### Implemented Features ✅

- Core JSON-RPC message handling (serialization/deserialization)
- Server architecture with option-based configuration
- Protocol support for MCP versions:
  - 2024-11-05
  - 2025-03-26
- Transport layers:
  - Standard I/O (stdio) - fully functional
  - HTTP - basic functionality
- Tool capability:
  - Tool registration and execution
  - JSON schema generation
  - Tool result handling
- Server lifecycle management
- Client state management
- Command support:
  - `ping` - Simple connectivity check
  - `initialize` - Client initialization and capability negotiation
  - `tools/list` - List available tools
  - `tools/call` - Execute a specific tool
- Batch message processing (JSON-RPC batch)
- Error handling and propagation
- Logging infrastructure

### In Development 🚧

- HTTP transport enhancements:
  - SSE support for notifications
  - Multiple client connections
  - Request validation
- Resources capability implementation
- Prompts capability implementation
- Roots capability implementation
- Sampling capability implementation
- Notification handling
- Client implementation improvements

### Planned Features 📋

- Completions capability
- Advanced authentication and security features
- Advanced client implementation with automatic reconnection
- Extended examples for all capabilities
- Comprehensive documentation
- Test coverage expansion
- Performance optimizations

## Getting Started

### Installation

```bash
go get github.com/lucacox/go-mcp
```

### Basic Example

Here's a current example of creating an MCP server with tool capabilities:

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/lucacox/go-mcp/internal/logging"
	"github.com/lucacox/go-mcp/pkg/capabilities/tools"
	"github.com/lucacox/go-mcp/pkg/protocol"
	"github.com/lucacox/go-mcp/pkg/server"
)

var exampleTool = tools.NewTool("example_tool", "this is an example tool", protocol.ObjectSchema(
	map[string]*protocol.JSONSchema{
		"param1": protocol.StringSchema("example param 1"),
		"param2": protocol.NumberSchema("example param 2"),
	},
	[]string{"param1", "param2"},
), nil, func(ctx context.Context, arguments json.RawMessage) (*tools.ToolResult, error) {
	var params map[string]interface{}
	if err := json.Unmarshal(arguments, &params); err != nil {
		return nil, err
	}
	result := tools.NewToolResult([]tools.ContentItem{
		tools.NewTextContent(fmt.Sprintf("this is the example_tool result. Param1: %s, Param2: %f\n", params["param1"], params["param2"])),
	}, false)
	return result, nil
})

var systemUserName = tools.NewTool("system_user_name", "returns the username of the current logged-in user", protocol.ObjectSchema(nil, nil), nil, func(ctx context.Context, arguments json.RawMessage) (*tools.ToolResult, error) {
	username, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	result := tools.NewToolResult([]tools.ContentItem{
		tools.NewTextContent(fmt.Sprintf("Current user: %s\n", username)),
	}, false)
	return result, nil
})

func main() {
	loggerFactory := logging.NewLoggerFactory()
	loggerFactory.SetLevel(logging.LevelDebug)
	logger := loggerFactory.CreateLogger("main")

	// Create a new MCP server instance
	srv := server.NewServer(
		server.WithServerName("ExampleServer"),
		server.WithServerVersion("1.0.0"),
		server.WithLogger(slog.LevelDebug),
		server.WithTransports(protocol.TransportTypeStdio),
		server.WithProtocolVersion(protocol.ProtocolVersion20241105),
		server.WithProtocolVersion(protocol.ProtocolVersion20250326),
		server.WithTool(exampleTool),
		server.WithTool(systemUserName),
	)

	// Create a context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the MCP server
	if err := srv.Start(ctx); err != nil {
		logging.Debug(logger, "Error starting server", "error", err)
		os.Exit(1)
	}

	logging.Debug(logger, "MCP Server started", "version", srv.Version, "name", srv.Name)

	// Wait for interruption signal
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	// Gracefully shut down the server
	logging.Debug(logger, "Shutting down server...")
	srv.Shutdown()
	logging.Debug(logger, "Server successfully stopped")
}
```

## Advanced Usage

### Using HTTP Transport

TODO

### Batch Message Handling

Go-MCP supports batch message processing:

TODO

## Contributing

Contributions to the Go-MCP project are welcome and encouraged! Here's how you can contribute:

### Setting Up Development Environment

1. Clone the repository:
   ```bash
   git clone https://github.com/lucacox/go-mcp.git
   cd go-mcp
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Run tests:
   ```bash
   go test ./...
   ```

### Testing MCP Server

You can test the MCP server using named pipes:

1. Create a named pipe:
   ```bash
   mkfifo MYPIPE
   ```

2. Run the server using the pipe as input:
   ```bash
   go run ./cmd/examples/main.go < MYPIPE
   ```

3. Send test requests to the server:
   ```bash
   cat test/initialize.json > MYPIPE
   cat test/tools_list.json > MYPIPE
   cat test/tools_call.json > MYPIPE
   ```

### Priority Areas for Contribution

- Completing the HTTP transport implementation with SSE support
- Implementing the remaining capabilities (Resources, Prompts, etc.)
- Enhancing error handling and validation
- Adding more comprehensive examples
- Improving documentation
- Writing more tests
- Performance optimizations


## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgements

- [Model Context Protocol](https://modelcontextprotocol.io/specification/2025-03-26) - The original protocol specification
- [JSON-RPC 2.0](https://www.jsonrpc.org/specification) - The underlying RPC protocol