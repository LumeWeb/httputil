package httputil

import (
	"context"
	"net/http"
)

// RequestContext carries HTTP request-scoped values and provides utility methods
// for handling the complete request/response lifecycle. It combines:
// - Standard context.Context for cancellation and deadlines
// - HTTP request object for reading request data
// - HTTP response writer for sending responses
// - Validation and encoding utilities
type RequestContext struct {
	context.Context
	Request  *http.Request
	Response http.ResponseWriter
}

// Context creates a new RequestContext from an HTTP request and response writer.
// This should be the primary way to initialize the request context for handlers.
func Context(r *http.Request, w http.ResponseWriter) RequestContext {
	return RequestContext{
		Context:  r.Context(),
		Request:  r,
		Response: w,
	}
}
