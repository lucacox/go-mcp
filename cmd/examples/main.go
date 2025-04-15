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
