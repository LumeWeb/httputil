# HTTP Utility Package

A lightweight Go package providing HTTP request/response utilities with:
- Request context management
- JSON encoding/decoding
- Form data parsing
- Data validation
- DTO pattern implementation
- Structured error handling with option pattern
- Customizable error handlers
- Validation middleware support

## Features

### Request Processing Pipeline
```go
// Create context and handle request
r := httputil.Context(req, w)

// Decode, validate and convert to domain model
model, ok := httputil.DecodeAndValidateRequest(r, &requestDTO)
if !ok {
    return // Error already handled
}

// Encode response from domain model
_ = httputil.EncodeResponse(r, model, &responseDTO)
```

### Form Data Parsing
```go
// Parse form values into different types
err := r.DecodeForm("field", &target)
// Supports: string, int, bool, time.Time, slices, and custom types
```

### Validation
```go
// Validate request using schema
err := r.Validate(validator)

// Or get validation errors as map
errors, err := r.ValidateRequest(validator)
```

### Option Pattern Configuration
```go
// Custom error handler that logs errors
errorHandler := func(ctx httputil.RequestContext, err error) {
    log.Printf("Request error: %v", err)
    httputil.DefaultErrorHandler{}.HandleError(ctx, err)
}

// Use custom handler for a request
model, ok := httputil.DecodeAndValidateRequest(
    r, 
    &requestDTO,
    httputil.WithErrorHandler(errorHandler),
)
```

## Installation
```bash
go get go.lumeweb.com/httputil
```

## Error Handling

The package provides three levels of error handling:

1. **Automatic Errors** (400/422/500 status codes):
```go
// Errors automatically handled with appropriate status codes
model, ok := httputil.DecodeAndValidateRequest(r, &dto)
```

2. **Custom Error Handler**:
```go
// Create custom error handler
type LoggingHandler struct{}

func (h LoggingHandler) HandleError(ctx httputil.RequestContext, err error) {
    log.Printf("Error processing request: %v", err)
    httputil.DefaultErrorHandler{}.HandleError(ctx, err)
}

// Use custom handler
model, ok := httputil.DecodeAndValidateRequest(
    r,
    &dto,
    httputil.WithErrorHandler(LoggingHandler{}),
)
```

3. **Manual Error Handling**:
```go
// Full manual control
cfg := httputil.vdConfig{
    errorHandler: httputil.ErrorHandlerFunc(func(ctx httputil.RequestContext, err error) {
        // Custom error handling logic
    }),
}

model, ok := httputil.DecodeAndValidateRequest(r, &dto, httputil.WithErrorHandler(cfg.errorHandler))
```

### Complete Example

#### Request/Response Flow
```go
func createUserHandler(w http.ResponseWriter, req *http.Request) {
    // Initialize context
    r := httputil.Context(req, w)
    
    // Decode and validate request
    var createReq CreateUserRequest
    user, ok := httputil.DecodeAndValidateRequest(r, &createReq)
    if !ok {
        return // Error response already handled
    }
    
    // Business logic
    if err := userService.Create(user); err != nil {
        _ = r.Error(fmt.Errorf("creation failed: %w", err), http.StatusConflict)
        return
    }
    
    // Create and send response
    resp := UserResponse{}
    _ = httputil.EncodeResponse(r, user, &resp)
}
```

#### Custom Error Handling
```go
type MetricsErrorHandler struct {
    metricsClient metrics.Client
}

func (h MetricsErrorHandler) HandleError(ctx httputil.RequestContext, err error) {
    // Track error metrics
    h.metricsClient.Increment("request.errors", 1)
    
    // Fallback to default handling
    httputil.DefaultErrorHandler{}.HandleError(ctx, err)
}

// Usage
func metricsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := httputil.Context(r, w)
        handler := MetricsErrorHandler{metricsClient: globalMetrics}
        
        // Process request with metrics tracking
        var createReq CreateUserRequest
        user, ok := httputil.DecodeAndValidateRequest(
            ctx, 
            &createReq,
            httputil.WithErrorHandler(handler),
        )
        // ... rest of handler logic
    })
}
```

## Error Handling
All errors are automatically formatted and returned with appropriate HTTP status codes.

## Testing
The package includes comprehensive tests and mock implementations for easy testing.

## Credits
Originally adapted from [jape](https://github.com/SiaFoundation/jape) but extended with additional features.
