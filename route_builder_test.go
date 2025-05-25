package httputil

import (
	swagger "go.lumeweb.com/gswagger"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"go.lumeweb.com/portal/core"
)

func TestNewRoute(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {}

	route := NewRoute("GET", "/test", handler)

	assert.Equal(t, "GET", route.Method)
	assert.Equal(t, "/test", route.Path)
	assert.NotNil(t, route.Handler)
	assert.Equal(t, core.ACCESS_USER_ROLE, route.Access)
	assert.NotNil(t, route.Swagger)
	assert.Empty(t, route.Middlewares)
}

func TestWithAccess(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {}

	route := NewRoute("GET", "/test", handler, WithAccess(core.ACCESS_ADMIN_ROLE))

	assert.Equal(t, core.ACCESS_ADMIN_ROLE, route.Access)
}

func TestWithSwagger(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {}
	swaggerDef := swagger.Definitions{
		Summary: "Test Summary",
	}

	route := NewRoute("GET", "/test", handler, WithSwagger(swaggerDef))

	assert.Equal(t, "Test Summary", route.Swagger.Summary)
}

func TestRouteExecution(t *testing.T) {
	called := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}

	route := NewRoute("GET", "/test", handler)

	router := mux.NewRouter()
	router.HandleFunc(route.Path, route.Handler).Methods(route.Method)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rr.Code)
}
