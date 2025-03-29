package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	swagger "go.lumeweb.com/gswagger"
	"go.lumeweb.com/portal-middleware/auth/jwt"
	coreTesting "go.lumeweb.com/portal/core/testing"
	coreMocks "go.lumeweb.com/portal/core/testing/mocks"
)

func TestRegisterRoutes(t *testing.T) {
	tests := []struct {
		name              string
		routes           []RouteDefinition
		accessSvcErr     error
		wantRegisterErr  bool
		wantSwaggerGenErr bool
		wantAccessReg    bool
	}{
		{
			name: "successful registration",
			routes: []RouteDefinition{
				{
					Path:    "/test",
					Method:  "GET",
					Handler: func(w http.ResponseWriter, r *http.Request) {},
				},
			},
			wantRegisterErr:  false,
			wantSwaggerGenErr: false,
			wantAccessReg:    false,
		},
		{
			name: "with access control",
			routes: []RouteDefinition{
				{
					Path:    "/secure",
					Method:  "GET",
					Handler: func(w http.ResponseWriter, r *http.Request) {},
					Access:  "admin",
				},
			},
			wantRegisterErr:  false,
			wantSwaggerGenErr: false,
			wantAccessReg:    true,
		},
		{
			name: "access service error",
			routes: []RouteDefinition{
				{
					Path:    "/secure",
					Method:  "GET",
					Handler: func(w http.ResponseWriter, r *http.Request) {},
					Access:  "admin",
				},
			},
			accessSvcErr:     assert.AnError,
			wantRegisterErr:  true,
			wantSwaggerGenErr: false,
			wantAccessReg:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			muxRouter := mux.NewRouter()
			gRouter, err := NewSwaggerRouter(muxRouter, APIInfo().
				Title("Test API").
				Version("1.0.0"))
			require.NoError(t, err)
			accessSvc := coreMocks.NewMockAccessService(t)
			ctx := coreTesting.NewTestContext(t)

			if tt.wantAccessReg || tt.accessSvcErr != nil {
				// If access registration is expected OR an access service error is expected,
				// set up the mock to be called and return the specified error.
				accessSvc.On("RegisterRoute", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(tt.accessSvcErr).Once()
			} else {
				// Otherwise, assert that RegisterRoute is not called.
				accessSvc.AssertNotCalled(t, "RegisterRoute", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			}

			err = RegisterRoutes(ctx, muxRouter, gRouter, accessSvc, "test", tt.routes)

			if tt.wantRegisterErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			accessSvc.AssertExpectations(t)

			// Generate OpenAPI spec after routes are registered
			err = gRouter.GenerateAndExposeOpenapi()
			if tt.wantSwaggerGenErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify routes are registered by making test requests
			for _, route := range tt.routes {
				req := httptest.NewRequest(route.Method, route.Path, nil)
				rr := httptest.NewRecorder()
				muxRouter.ServeHTTP(rr, req)
				assert.NotEqual(t, http.StatusNotFound, rr.Code)
			}
		})
	}
}

func TestDefineRoutes(t *testing.T) {
	r1 := RouteDefinition{Path: "/one"}
	r2 := RouteDefinition{Path: "/two"}

	routes := DefineRoutes(r1, r2)

	assert.Len(t, routes, 2)
	assert.Equal(t, "/one", routes[0].Path)
	assert.Equal(t, "/two", routes[1].Path)
}

func TestAuthSwagger(t *testing.T) {
	def := AuthSwagger(
		"Test Summary",
		"Test Description",
		jwt.PurposeLogin,
		nil,
		nil,
		nil,
	)

	assert.Equal(t, "Test Summary", def.Summary)
	assert.Equal(t, "Test Description", def.Description)
	assert.Contains(t, def.Tags, "Authenticated")
	assert.NotEmpty(t, def.Security)
	assert.Contains(t, def.Responses, http.StatusOK)
	assert.Contains(t, def.Responses, http.StatusUnauthorized)
	assert.Contains(t, def.Responses, http.StatusForbidden)
}

func TestBasicSwagger(t *testing.T) {
	def := BasicSwagger(
		"Test Summary",
		"Test Description",
		nil,
		nil,
		nil,
	)

	assert.Equal(t, "Test Summary", def.Summary)
	assert.Equal(t, "Test Description", def.Description)
	assert.Contains(t, def.Tags, "Public")
	assert.Contains(t, def.Responses, http.StatusOK)
	assert.NotContains(t, def.Security, "bearerAuth")
}

func TestWithPathParam(t *testing.T) {
	def := swagger.Definitions{}
	def = WithPathParam(def, "id", "Test ID", "string")

	assert.NotNil(t, def.PathParams)
	assert.Equal(t, "Test ID", def.PathParams["id"].Description)
}

func TestWithQueryParam(t *testing.T) {
	def := swagger.Definitions{}
	def = WithQueryParam(def, "filter", "Test filter", "string")

	assert.NotNil(t, def.Querystring)
	assert.Equal(t, "Test filter", def.Querystring["filter"].Description)
}

func TestWithPaginationParams(t *testing.T) {
	def := swagger.Definitions{}
	def = WithPaginationParams(def)

	assert.NotNil(t, def.Querystring)
	assert.Contains(t, def.Querystring, "_start")
	assert.Contains(t, def.Querystring, "_end")
}

func TestNewSwaggerRouter(t *testing.T) {
	muxRouter := mux.NewRouter()
	gRouter, err := NewSwaggerRouter(muxRouter, APIInfo().
		Title("Test API").
		Version("1.0.0"))

	assert.NoError(t, err)
	assert.NotNil(t, gRouter)
}

func TestSwaggerDocsServed(t *testing.T) {
	muxRouter := mux.NewRouter()
	gRouter, err := NewSwaggerRouter(muxRouter, APIInfo().
		Title("Test API").
		Version("1.0.0"))
	require.NoError(t, err)
	require.NotNil(t, gRouter)

	// Register a test route first
	routes := DefineRoutes(
		RouteDefinition{
			Path:    "/test",
			Method:  "GET",
			Handler: func(w http.ResponseWriter, r *http.Request) {},
		},
	)
	err = RegisterRoutes(coreTesting.NewTestContext(t), muxRouter, gRouter, nil, "", routes)
	require.NoError(t, err)

	// Now generate the OpenAPI spec
	err = gRouter.GenerateAndExposeOpenapi()
	require.NoError(t, err)

	tests := []struct {
		name     string
		path     string
		wantCode int
	}{
		{
			name:     "JSON docs",
			path:     SwaggerJSONPath,
			wantCode: http.StatusOK,
		},
		{
			name:     "YAML docs",
			path:     SwaggerYAMLPath,
			wantCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rr := httptest.NewRecorder()
			muxRouter.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantCode, rr.Code)
			assert.NotEmpty(t, rr.Body.String())
		})
	}
}

func TestWithSortParams(t *testing.T) {
	def := swagger.Definitions{}
	def = WithSortParams(def, []string{"name", "date"})

	assert.NotNil(t, def.Querystring)
	assert.Contains(t, def.Querystring, "_sort")
	assert.Contains(t, def.Querystring, "_order")
	assert.Contains(t, def.Querystring["_sort"].Description, "name, date")
}
