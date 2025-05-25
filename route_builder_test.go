package httputil

import (
	swagger "go.lumeweb.com/gswagger"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"go.lumeweb.com/portal/core"
	coreTesting "go.lumeweb.com/portal/core/testing"
	"go.lumeweb.com/portal/core/testing/mocks"
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

func TestWithVerification(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {}
	ctx := coreTesting.NewTestContext(t)

	// Setup mock expectations
	mockUser := mocks.NewMockUserService(t)
	ctx.RegisterService(core.USER_SERVICE, mockUser)

	route := NewRoute("GET", "/test", handler, WithVerification(ctx))

	assert.Len(t, route.Middlewares, 1)
	mockUser.AssertExpectations(t)
}

func TestWith2FA(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {}
	ctx := mocks.NewMockContext(t)

	route := NewRoute("GET", "/test", handler, With2FA(ctx))

	assert.Len(t, route.Middlewares, 1)
}

func TestWithMiddleware(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {}
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	}

	route := NewRoute("GET", "/test", handler, WithMiddleware(mw))

	assert.Len(t, route.Middlewares, 1)
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

func TestMultipleOptions(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {}
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	}
	ctx := coreTesting.NewTestContext(t)

	// Setup mock expectations
	mockUser := mocks.NewMockUserService(t)
	ctx.RegisterService(core.USER_SERVICE, mockUser)

	route := NewRoute(
		"GET",
		"/test",
		handler,
		WithAccess(core.ACCESS_ADMIN_ROLE),
		WithVerification(ctx),
		WithMiddleware(mw),
	)

	assert.Equal(t, core.ACCESS_ADMIN_ROLE, route.Access)
	assert.Len(t, route.Middlewares, 2)
	mockUser.AssertExpectations(t)
}
