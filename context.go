package httputil

import (
	"context"
	"net/http"
)

// RequestContext carries HTTP request-scoped values and provides utility methods.
// It wraps:
// - Standard context.Context
// - HTTP request object
// - HTTP response writer
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
