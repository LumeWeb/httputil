// Package httputil provides HTTP utility functions for building robust web APIs in Go.
// It implements patterns for type-safe request/response handling with validation, structured error reporting,
// and OpenAPI documentation generation.
//
// Key features:
// - DTO (Data Transfer Object) pattern for request/response marshaling
// - Declarative validation using struct schemas
// - Option pattern configuration for request processing
// - Automatic error classification and JSON error responses
// - Context-aware request handling
// - Integrated OpenAPI/Swagger documentation generation
// - TUS protocol support for file uploads
// - Route registration with built-in middleware support
//
// The package emphasizes:
// - Type safety through generics
// - Clear separation of concerns
// - Customizable error handling
// - Consistent JSON formatting
// - Middleware-ready components
// - Self-documenting APIs
//
// Router Features:
// - Unified route registration with access control
// - Automatic OpenAPI spec generation
// - Built-in support for common patterns:
//   - Pagination (_start, _end params)
//   - Sorting (_sort, _order params)
//   - Filtering (custom field filters)
//   - Global search (q param)
// - Predefined Swagger definitions for:
//   - Authenticated endpoints
//   - Public endpoints
//   - File uploads (multipart/form-data)
//   - TUS protocol endpoints
//   - Paginated responses
package httputil
