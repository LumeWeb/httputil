package httputil

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Encode writes a JSON response to the client with consistent formatting:
// - Empty slices/maps render as []/{} instead of null
// - Uses indented JSON for human readability
func (r RequestContext) Encode(v any) {
	// encode nil slices as [] and nil maps as {} (instead of null)
	if val := reflect.ValueOf(v); val.Kind() == reflect.Slice && val.Len() == 0 {
		_ = r.JSON(http.StatusOK, []any{})
		return
	} else if val.Kind() == reflect.Map && val.Len() == 0 {
		_ = r.JSON(http.StatusOK, map[string]any{})
		return
	}
	_ = r.JSONPretty(http.StatusOK, v, "\t")
}

// Decode reads and parses the request body as JSON into the provided value.
// Returns an error wrapped with type information if decoding fails
func (r RequestContext) Decode(v any) error {
	if err := r.Bind(v); err != nil {
		return fmt.Errorf("couldn't decode request type (%T): %w", v, err)
	}
	return nil
}

// Error writes an HTTP error response and returns the error. Formats:
// - ValidationErrors as 422 Unprocessable Entity with field-specific errors
// - All other errors using Echo's error handler
func (r RequestContext) Error(err error, status int) error {
	if err == nil {
		return nil
	}

	var vErr *ValidationError
	if errors.As(err, &vErr) {
		err = r.JSON(status, map[string]any{
			"error":  vErr.Error(),
			"fields": vErr.Fields(),
		})
		if err != nil {
			return err
		}
		return vErr // Return original validation error
	}
	// Return the error from the JSON call
	jsonErr := r.JSON(status, map[string]any{"error": err.Error()})
	if jsonErr != nil {
		return jsonErr
	}
	return err // Return original error for test assertions
}

// Check simplifies error handling by:
// - Returning nil if err is nil
// - Wrapping the error with a message
// - Sending an HTTP 500 response if error exists
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
	value := r.FormValue(key)
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
		err = r.Error(fmt.Errorf("invalid form value %q: %w", key, err), http.StatusBadRequest)
		if err != nil {
			return err
		}
		return fmt.Errorf("invalid form value %q: %w", key, err)
	}
	return nil
}
