# HTTP Utility Package

A lightweight Go package providing HTTP request/response utilities with:
- Request context management
- JSON encoding/decoding
- Form data parsing
- Data validation
- DTO pattern implementation
- Consistent error handling

## Features

### Request Context
```go
r := httputil.Context(req, w) // Create request context
```

### JSON Handling
```go
// Decode request body
err := r.Decode(&myStruct)

// Encode response
r.Encode(responseData)
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

### DTO Pattern
```go
// Request processing pipeline
err := r.DecodeAndValidateRequest(dto, &model)

// Response pipeline  
r.EncodeResponse(model, dto)
```

## Installation
```bash
go get go.lumeweb.com/httputil
```

## Examples

### Basic Handler
```go
func handler(w http.ResponseWriter, req *http.Request) {
    r := httputil.Context(req, w)
    
    var input InputDTO
    if err := r.DecodeAndValidateRequest(&input, &model); err != nil {
        return // Error already handled
    }
    
    // Process model...
    
    r.EncodeResponse(model, &OutputDTO{})
}
```

### Complete Example with Structs

#### Request DTO Example
```go
type CreateUserRequest struct {
    Username string `json:"username"`
    Email    string `json:"email"`
    Age      int    `json:"age"`
}

func (r *CreateUserRequest) ToModel(model any) error {
    user, ok := model.(*User) // Assume User is your domain model
    if !ok {
        return fmt.Errorf("expected *User, got %T", model)
    }
    user.Username = r.Username
    user.Email = r.Email
    user.Age = r.Age
    return nil
}

func (r *CreateUserRequest) Schema() *z.StructSchema {
    return z.Struct(z.Schema{
        "username": z.String().Min(3).Max(50),
        "email":    z.String().Email(),
        "age":      z.Number().Min(18).Max(120),
    })
}
```

#### Response DTO Example
```go
type UserResponse struct {
    ID       string `json:"id"`
    Username string `json:"username"`
    Email    string `json:"email"`
}

func (r *UserResponse) FromModel(model any) error {
    user, ok := model.(*User)
    if !ok {
        return fmt.Errorf("expected *User, got %T", model)
    }
    r.ID = user.ID
    r.Username = user.Username
    r.Email = user.Email
    return nil
}
```

#### Full Handler Example
```go
func createUserHandler(w http.ResponseWriter, req *http.Request) {
    r := httputil.Context(req, w)
    
    var input CreateUserRequest
    var user User
    
    if err := r.DecodeAndValidateRequest(&input, &user); err != nil {
        // Error already handled with proper HTTP response
        return
    }
    
    // Save user to database...
    user.ID = generateID()
    
    // Return response
    r.EncodeResponse(&user, &UserResponse{})
}
```

## Error Handling
All errors are automatically formatted and returned with appropriate HTTP status codes.

## Testing
The package includes comprehensive tests and mock implementations for easy testing.

## Credits
Originally adapted from [jape](https://github.com/SiaFoundation/jape) but extended with additional features.
