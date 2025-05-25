package httputil

import (
	"github.com/labstack/echo/v4"
)

// RequestContext carries HTTP request-scoped values and provides utility methods
// for handling the complete request/response lifecycle. It combines:
// - Validation and encoding utilities
// - Embedded Echo Context for framework features
type RequestContext struct {
	echo.Context // Embed the Echo Context interface
}

// Context creates a new RequestContext from an Echo context.
// This should be the primary way to initialize the request context for handlers.
func Context(c echo.Context) RequestContext {
	if c == nil {
		panic("nil echo.Context provided to httputil.Context()")
	}
	if c.Request() == nil {
		panic("echo.Context with nil Request provided to httputil.Context()")
	}
	if c.Request().Context() == nil {
		panic("http.Request with nil Context provided to httputil.Context()")
	}
	return RequestContext{
		Context: c,
	}
}
