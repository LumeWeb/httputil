package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

// echoContextWrapper is a minimal echo.Context implementation for testing nil cases
type echoContextWrapper struct {
	echo.Context
}

func (c *echoContextWrapper) Request() *http.Request {
	return nil
}

func TestContext(t *testing.T) {
	e := echo.New()

	t.Run("basic properties", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		echoCtx := e.NewContext(req, w)

		ctx := Context(echoCtx)

		if ctx.Request() != echoCtx.Request() {
			t.Errorf("RequestContext.Request() != echoCtx.Request()")
		}
		if ctx.Response() != echoCtx.Response() {
			t.Errorf("RequestContext.Response() != echoCtx.Response()")
		}
		if ctx.Request().Context() != req.Context() {
			t.Errorf("RequestContext.Request().Context() != req.Context()")
		}
	})

	t.Run("nil echo.Context panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic with nil echo.Context")
			}
		}()
		Context(nil)
	})

	t.Run("nil Request panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic with nil Request")
			}
		}()
		Context(&echoContextWrapper{})
	})

	t.Run("nil Request.Context panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic with nil Request.Context")
			}
		}()
		req := httptest.NewRequest("GET", "/test", nil)
		req = req.WithContext(nil)
		echoCtx := e.NewContext(req, httptest.NewRecorder())
		Context(echoCtx)
	})

	t.Run("embedded echo methods", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		echoCtx := e.NewContext(req, w)

		ctx := Context(echoCtx)

		// Test embedded Echo Context methods
		if ctx.Request().Method != "GET" {
			t.Errorf("Expected method 'GET', got '%s'", ctx.Request().Method)
		}

		// Path() is empty until set by router
		if ctx.Path() != "" {
			t.Errorf("Expected empty path before routing, got '%s'", ctx.Path())
		}

		// Status is 0 until WriteHeader is called
		if ctx.Response().Status != 0 {
			t.Errorf("Expected status 0 before writing, got %d", ctx.Response().Status)
		}
	})

	t.Run("response writer access", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		echoCtx := e.NewContext(req, w)

		ctx := Context(echoCtx)
		_, err := ctx.Response().Write([]byte("test"))
		if err != nil {
			t.Errorf("Unexpected write error: %v", err)
		}

		if w.Body.String() != "test" {
			t.Errorf("Expected body 'test', got '%s'", w.Body.String())
		}
	})
}
