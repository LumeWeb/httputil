package httputil

import (
	swagger "go.lumeweb.com/gswagger"
	"go.lumeweb.com/portal/core"
	"net/http"
)

type RouteOption func(*RouteDefinition)

// Core route builder
func NewRoute(method, path string, handler http.HandlerFunc, opts ...RouteOption) RouteDefinition {
	def := RouteDefinition{
		Method:  method,
		Path:    path,
		Handler: handler,
		// Defaults
		Access:  core.ACCESS_USER_ROLE,
		Swagger: swagger.Definitions{},
	}

	for _, opt := range opts {
		opt(&def)
	}

	return def
}

// Option setters
func WithAccess(accessRole string) RouteOption {
	return func(d *RouteDefinition) {
		d.Access = accessRole
	}
}

func WithSwagger(def swagger.Definitions) RouteOption {
	return func(d *RouteDefinition) {
		d.Swagger = def
	}
}
