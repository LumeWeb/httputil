package httputil

import (
	"testing"

	"github.com/labstack/echo/v4"
	"go.lumeweb.com/httputil/internal/mocks"

	"io"
	"net/http"
	"net/http/httptest"

	z "github.com/Oudwins/zog"
)

// newChunkedGetRequest creates a GET request with chunked transfer encoding
// and a closed body pipe, simulating the production bug where Echo's BindBody
// hits 415/EOF on an empty chunked body.
func newChunkedGetRequest(target string, contentType string) *http.Request {
	pr, pw := io.Pipe()
	pw.Close()

	req, _ := http.NewRequest("GET", target, pr)
	req.Header.Set("Content-Type", contentType)
	req.ContentLength = -1
	req.TransferEncoding = []string{"chunked"}
	return req
}

// setupPathContext creates an Echo context with path parameters set up
// from the last path segment of the request URL, and returns a RequestContext
// wrapping it.
func setupPathContext(req *http.Request) RequestContext {
	w := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, w)
	c.SetPath("/dag/:cid")
	c.SetParamNames("cid")
	// Use the last path segment as the CID value
	path := req.URL.Path
	if idx := len(path) - 1; idx >= 0 {
		for i := idx; i >= 0; i-- {
			if path[i] == '/' {
				c.SetParamValues(path[i+1:])
				break
			}
		}
	}
	return Context(c)
}

// newMinLengthValidator creates a mock DTOValidator with a zog schema that
// requires the given field to be a string with at least minLen characters.
func newMinLengthValidator(t *testing.T, field string, minLen int) *mocks.MockDTOValidator {
	v := mocks.NewMockDTOValidator(t)
	schema := z.Struct(z.Schema{
		field: z.String().Min(minLen).Required(),
	})
	v.EXPECT().Schema().Return(schema)
	return v
}
