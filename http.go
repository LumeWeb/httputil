package httputil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Encode writes a JSON response to the client with consistent formatting:
// - Empty slices/maps render as []/{} instead of null
// - Sets application/json Content-Type header
// - Uses indented JSON for human readability
// - Handles special types through reflection
func (r RequestContext) Encode(v any) {
	r.Response.Header().Set("Content-Type", "application/json")
	// encode nil slices as [] and nil maps as {} (instead of null)
	if val := reflect.ValueOf(v); val.Kind() == reflect.Slice && val.Len() == 0 {
		_, _ = r.Response.Write([]byte("[]\n"))
		return
	} else if val.Kind() == reflect.Map && val.Len() == 0 {
		_, _ = r.Response.Write([]byte("{}\n"))
		return
	}
	enc := json.NewEncoder(r.Response)
	enc.SetIndent("", "\t")
	_ = enc.Encode(v)
}

// Decode reads and parses the request body as JSON into the provided value.
// Returns an error wrapped with type information if decoding fails
func (r RequestContext) Decode(v any) error {
	// Read and restore body when GetBody isn't available
	bodyContent, err := io.ReadAll(r.Request.Body)
	if err != nil {
		return fmt.Errorf("error reading request body: %w", err)
	}
	// Restore original body from the read content
	r.Request.Body = io.NopCloser(bytes.NewReader(bodyContent))
	if err := json.NewDecoder(r.Request.Body).Decode(v); err != nil {
		return fmt.Errorf("couldn't decode request type (%T): %w", v, err)
	}
	return nil
}

// Error writes an HTTP error response and returns the error. Formats:
// - ValidationErrors as 422 Unprocessable Entity with field-specific errors
// - All other errors as JSON response with "error" field and appropriate status code
// Maintains consistent JSON formatting for all error responses.
func (r RequestContext) Error(err error, status int) error {
	// Handle validation errors with structured response
	if vErr, ok := err.(*ValidationError); ok {
		r.Response.Header().Set("Content-Type", "application/json")
		r.Response.WriteHeader(status)
		json.NewEncoder(r.Response).Encode(map[string]any{
			"error":  "validation failed",
			"fields": vErr.Fields(),
		})
		return err
	}

	// Default error handling with JSON response
	r.Response.Header().Set("Content-Type", "application/json")
	r.Response.WriteHeader(status)
	json.NewEncoder(r.Response).Encode(map[string]any{
		"error": err.Error(),
	})
	return err
}

// Check simplifies error handling by:
// - Returning nil if err is nil
// - Wrapping the error with a message
// - Sending an HTTP 500 response if error exists
// Use this for checking operational errors that should result in server errors
func (r RequestContext) Check(msg string, err error) error {
	if err != nil {
		return r.Error(fmt.Errorf("%v: %w", msg, err), http.StatusInternalServerError)
	}
	return nil
}

// DecodeForm parses and converts form values to various types. Supports:
// - Standard types (string, int, bool, time.Time)
// - Slice types from comma-separated values
// - Custom types implementing UnmarshalText or LoadString interfaces
// Panics if passed an unsupported type
func (r RequestContext) DecodeForm(key string, v any) error {
	// Ensure form is parsed first
	if r.Request.PostForm == nil {
		err := r.Request.ParseForm()
		if err != nil {
			return err
		}
	}
	value := r.Request.PostFormValue(key)
	if value == "" {
		return nil
	}
	var err error
	switch v := v.(type) {
	case interface{ UnmarshalText([]byte) error }:
		err = v.UnmarshalText([]byte(value))
	case interface{ LoadString(string) error }:
		err = v.LoadString(value)
	case *string:
		*v = value
	case *[]string:
		*v = strings.Split(value, ",")
	case *int:
		*v, err = strconv.Atoi(value)
	case *int64:
		*v, err = strconv.ParseInt(value, 10, 64)
	case *uint64:
		*v, err = strconv.ParseUint(value, 10, 64)
	case *bool:
		*v, err = strconv.ParseBool(value)
	case **time.Time:
		parsedTime, err := time.Parse(time.RFC3339, value)
		if err == nil {
			*v = &parsedTime
		}
	default:
		panic(fmt.Sprintf("unsupported type %T", v))
	}
	if err != nil {
		return r.Error(fmt.Errorf("invalid form value %q: %w", key, err), http.StatusBadRequest)
	}
	return nil
}
