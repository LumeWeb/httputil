// Package httputil provides HTTP utility functions for building robust web APIs in Go.
// It implements patterns for type-safe request/response handling with validation and structured error reporting.
//
// Key features:
// - DTO (Data Transfer Object) pattern for request/response marshaling
// - Declarative validation using struct schemas
// - Option pattern configuration for request processing
// - Automatic error classification and JSON error responses
// - Context-aware request handling
//
// The package emphasizes:
// - Type safety through generics
// - Clear separation of concerns
// - Customizable error handling
// - Consistent JSON formatting
// - Middleware-ready components
package httputil
