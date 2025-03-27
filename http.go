package httputil

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Encode writes a JSON response to the client. Handles special cases:
// - Empty slices are encoded as []
// - Empty maps are encoded as {}
// - Sets proper Content-Type header
// - Uses indented JSON for readability
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
// Returns:
// - error with HTTP status 400 if decoding fails
// - error is wrapped with type information for better error messages
func (r RequestContext) Decode(v any) error {
	if err := json.NewDecoder(r.Request.Body).Decode(v); err != nil {
		return r.Error(fmt.Errorf("couldn't decode request type (%T): %w", v, err), http.StatusBadRequest)
	}
	return nil
}

// Error writes an HTTP error response and returns the error.
// This ensures errors are properly communicated to both the client (via HTTP response)
// and the caller (via returned error).
func (r RequestContext) Error(err error, status int) error {
	http.Error(r.Response, err.Error(), status)
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
