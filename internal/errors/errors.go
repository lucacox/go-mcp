// Package errors provides a specific error handling system for MCP
package errors

import (
	"errors"
	"fmt"
)

// ErrorCode represents an MCP error code
type ErrorCode int

// Definition of standard error codes for MCP
const (
	// Generic errors
	Unknown ErrorCode = -1
	None    ErrorCode = 0

	// Protocol errors (1-999)
	ParseError         ErrorCode = 1
	InvalidRequest     ErrorCode = 2
	MethodNotFound     ErrorCode = 3
	InvalidParams      ErrorCode = 4
	InternalError      ErrorCode = 5
	ServerError        ErrorCode = 6
	TransportError     ErrorCode = 7
	TimeoutError       ErrorCode = 8
	CancelledError     ErrorCode = 9
	AuthorizationError ErrorCode = 10

	// Capability errors (1000-1999)
	CapabilityNotSupported ErrorCode = 1000
	CapabilityError        ErrorCode = 1001

	// Resource errors (2000-2999)
	ResourceNotFound      ErrorCode = 2000
	ResourceAccessDenied  ErrorCode = 2001
	ResourceInvalidFormat ErrorCode = 2002

	// Tool errors (3000-3999)
	ToolNotFound       ErrorCode = 3000
	ToolExecutionError ErrorCode = 3001

	// Prompt errors (4000-4999)
	PromptNotFound      ErrorCode = 4000
	PromptInvalidFormat ErrorCode = 4001

	// Sampling errors (5000-5999)
	SamplingError  ErrorCode = 5000
	SamplingDenied ErrorCode = 5001
)

// Error represents an MCP error with code and message
type Error struct {
	Code    ErrorCode
	Message string
	Details interface{}
	cause   error
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Details != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap implements the unwrapping interface
func (e *Error) Unwrap() error {
	return e.cause
}

// NewError creates a new MCP error
func NewError(code ErrorCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// NewErrorf creates a new MCP error with format
func NewErrorf(code ErrorCode, format string, args ...interface{}) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

// WithDetails adds details to the error
func (e *Error) WithDetails(details interface{}) *Error {
	e.Details = details
	return e
}

// WithCause adds a causal error
func (e *Error) WithCause(err error) *Error {
	e.cause = err
	return e
}

// Is checks if an error is of a certain type
func Is(err error, target error) bool {
	return errors.Is(err, target)
}

// As checks if an error is of a certain type and converts it
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// Unwrap extracts the causal error
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

// IsMCPError checks if an error is an MCP error and has the specified code
func IsMCPError(err error, code ErrorCode) bool {
	var mcpErr *Error
	if errors.As(err, &mcpErr) {
		return mcpErr.Code == code
	}
	return false
}

// GetMCPErrorCode extracts the error code from an error if it is of MCP type
func GetMCPErrorCode(err error) ErrorCode {
	var mcpErr *Error
	if errors.As(err, &mcpErr) {
		return mcpErr.Code
	}
	return Unknown
}
