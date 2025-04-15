// Package logging provides logging functionality for MCP using slog
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
)

// Custom logging levels (compatible with slog)
const (
	LevelTrace = slog.Level(-8)
	LevelDebug = slog.LevelDebug // -4
	LevelInfo  = slog.LevelInfo  // 0
	LevelWarn  = slog.LevelWarn  // 4
	LevelError = slog.LevelError // 8
	LevelFatal = slog.Level(12)
)

// Constants for level names
var levelNames = map[slog.Level]string{
	LevelTrace: "TRACE",
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
	LevelFatal: "FATAL",
}

// LoggerFactory is a factory for creating logger instances
type LoggerFactory struct {
	handler slog.Handler
}

// NewLoggerFactory creates a new factory with default settings
func NewLoggerFactory() *LoggerFactory {
	options := &slog.HandlerOptions{
		Level:       LevelInfo,
		ReplaceAttr: customizeLogLevels,
	}
	handler := slog.NewTextHandler(os.Stderr, options)

	return &LoggerFactory{
		handler: handler,
	}
}

// NewLoggerFactoryWithConfig creates a new factory with custom configuration
func NewLoggerFactoryWithConfig(w io.Writer, level slog.Level) *LoggerFactory {
	options := &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: customizeLogLevels,
	}

	// If w is nil, use stderr
	if w == nil {
		w = os.Stderr
	}

	handler := slog.NewTextHandler(w, options)

	return &LoggerFactory{
		handler: handler,
	}
}

// SetLevel sets the logging level for the factory
func (f *LoggerFactory) SetLevel(level slog.Level) {
	// In slog it's not possible to modify the level of an existing handler,
	// so we need to create a new one with the same writer
	// Extract the writer from the current handler
	var writer io.Writer

	// Assume handler is a TextHandler (most common case)
	switch f.handler.(type) {
	case *slog.TextHandler:
		// In Go 1.21+ we could use a method to get the writer
		// But for now we assume it's stderr if not specified otherwise
		writer = os.Stderr
	default:
		writer = os.Stderr
	}

	options := &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: customizeLogLevels,
	}

	f.handler = slog.NewTextHandler(writer, options)
}

// CreateLogger creates a new logger with the specified context
func (f *LoggerFactory) CreateLogger(name string) *slog.Logger {
	return slog.New(f.handler).With("component", name)
}

// Helper functions

// customizeLogLevels customizes log level names
func customizeLogLevels(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.LevelKey {
		level := a.Value.Any().(slog.Level)
		if name, ok := levelNames[level]; ok {
			return slog.Attr{Key: a.Key, Value: slog.StringValue(name)}
		}
	}
	return a
}

// Trace logs at trace level
func Trace(logger *slog.Logger, msg string, args ...any) {
	logger.Log(context.TODO(), LevelTrace, msg, args...)
}

// Debug logs at debug level
func Debug(logger *slog.Logger, msg string, args ...any) {
	logger.Debug(msg, args...)
}

// Info logs at info level
func Info(logger *slog.Logger, msg string, args ...any) {
	logger.Info(msg, args...)
}

// Warn logs at warn level
func Warn(logger *slog.Logger, msg string, args ...any) {
	logger.Warn(msg, args...)
}

// Error logs at error level
func Error(logger *slog.Logger, msg string, args ...any) {
	logger.Error(msg, args...)
}

// Fatal logs at fatal level and exits
func Fatal(logger *slog.Logger, msg string, args ...any) {
	logger.Log(context.TODO(), LevelFatal, msg, args...)
	os.Exit(1)
}
