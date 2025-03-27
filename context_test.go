package httputil

import (
	"net/http/httptest"
	"testing"
)

func TestContext(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	ctx := Context(req, w)

	if ctx.Request != req {
		t.Errorf("RequestContext.Request != req")
	}
	if ctx.Response != w {
		t.Errorf("RequestContext.Response != w")
	}
	if ctx.Context != req.Context() {
		t.Errorf("RequestContext.Context != req.Context()")
	}
}
