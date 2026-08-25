package httputil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// FileUploadResult contains the uploaded file and its metadata
type FileUploadResult struct {
	File     io.ReadSeekCloser
	Size     uint64
	Filename string
}

// PrepareFileUpload handles both multipart form uploads and raw body uploads.
// It supports:
// - Multipart form file uploads (with Content-Type: multipart/form-data)
// - Raw binary uploads (with any other Content-Type)
// - Automatic handling of seekable vs non-seekable sources
// - Size validation against maxUploadSize
func (r RequestContext) PrepareFileUpload(maxUploadSize int64) (*FileUploadResult, error) {
	req := r.Request()
	contentType := req.Header.Get("Content-Type")

	// Handle multipart form data uploads
	if strings.HasPrefix(contentType, "multipart/form-data") {
		// Check Content-Length header first to prevent processing large files
		if req.ContentLength > maxUploadSize {
			return nil, fmt.Errorf("file size exceeds maximum allowed size of %d bytes", maxUploadSize)
		}

		if err := req.ParseMultipartForm(maxUploadSize); err != nil {
			return nil, fmt.Errorf("failed to parse multipart form: %w", err)
		}

		multipartFile, multipartHeader, err := req.FormFile("file")
		if err != nil {
			return nil, fmt.Errorf("failed to get file from form: %w", err)
		}

		// Check if the multipart file supports seeking
		if seeker, ok := multipartFile.(io.Seeker); ok {
			// More reliable seek test - seek to start and back to original position
			if _, err := seeker.Seek(0, io.SeekStart); err == nil {
				// Success - we can use the original file with seeking support
				defer func() {
					if err != nil {
						multipartFile.Close()
					}
				}()
				return &FileUploadResult{
					File:     multipartFile,
					Size:     uint64(multipartHeader.Size),
					Filename: multipartHeader.Filename,
				}, nil
			}
		}

		// If seeking isn't supported or failed, fallback to buffering
		defer multipartFile.Close()
		data, err := io.ReadAll(multipartFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read multipart file: %w", err)
		}

		return &FileUploadResult{
			File:     readSeekNopCloser{bytes.NewReader(data)},
			Size:     uint64(len(data)),
			Filename: multipartHeader.Filename,
		}, nil
	}

	// Handle raw body uploads with size-limited reader
	limitedReader := &io.LimitedReader{R: req.Body, N: maxUploadSize + 1}
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	if limitedReader.N <= 0 {
		return nil, fmt.Errorf("file size exceeds maximum allowed size of %d bytes", maxUploadSize)
	}

	return &FileUploadResult{
		File: readSeekNopCloser{bytes.NewReader(data)},
		Size: uint64(len(data)),
	}, nil
}

// readSeekNopCloser implements io.ReadSeekCloser for bytes.Reader
type readSeekNopCloser struct {
	*bytes.Reader
}

func (rsnc readSeekNopCloser) Close() error {
	return nil
}

// Encode writes a JSON response to the client with consistent formatting:
// - Empty slices/maps render as []/{} instead of null
// - Uses indented JSON for human readability
func (r RequestContext) Encode(v any) error {
	// encode nil slices as [] and nil maps as {} (instead of null)
	if val := reflect.ValueOf(v); val.Kind() == reflect.Slice && val.Len() == 0 {
		return r.JSON(http.StatusOK, []any{})
	} else if val.Kind() == reflect.Map && val.Len() == 0 {
		return r.JSON(http.StatusOK, map[string]any{})
	}
	return r.JSONPretty(http.StatusOK, v, "\t")
}

// Decode reads and parses the request body as JSON into the provided value.
// Returns an error wrapped with type information if decoding fails
func (r RequestContext) Decode(v any) error {
	_, err := r.readRequestBody()
	if err != nil {
		return err
	}

	if err = r.Bind(v); err != nil {
		return fmt.Errorf("couldn't decode request type (%T): %w", v, err)
	}

	return nil
}

// DecodeQuery binds query parameters to the provided value using Echo's query
// parameter binder. Unlike Decode, this does not read or inspect the request
// body, making it safe for GET/HEAD/DELETE requests where only query params
// are used (e.g. filter DTOs with `query` struct tags).
func (r RequestContext) DecodeQuery(v any) error {
	type queryBinder interface {
		BindQueryParams(c echo.Context, i interface{}) error
	}

	var binder queryBinder
	if b, ok := r.Context.Echo().Binder.(queryBinder); ok {
		binder = b
	} else {
		binder = new(echo.DefaultBinder)
	}

	if err := binder.BindQueryParams(r.Context, v); err != nil {
		return fmt.Errorf("couldn't decode query type (%T): %w", v, err)
	}
	return nil
}

// DecodePathParams binds path parameters to the provided value using Echo's
// path parameter binder. Unlike Decode, this does not read or inspect the
// request body, making it safe for GET/HEAD/DELETE requests where only path
// params are used (e.g. DTOs with `param` struct tags).
func (r RequestContext) DecodePathParams(v any) error {
	type pathBinder interface {
		BindPathParams(c echo.Context, i interface{}) error
	}

	var binder pathBinder
	if b, ok := r.Context.Echo().Binder.(pathBinder); ok {
		binder = b
	} else {
		binder = new(echo.DefaultBinder)
	}

	if err := binder.BindPathParams(r.Context, v); err != nil {
		return fmt.Errorf("couldn't decode path params type (%T): %w", v, err)
	}
	return nil
}

// Error writes an HTTP error response and returns the error. Formats:
//   - ValidationErrors as 422 Unprocessable Entity with field-specific errors
//   - Errors implementing json.Marshaler as their custom JSON representation
//   - All other errors as the canonical structured error object under the "error"
//     key: {"error": {"reason": "...", "details": "..."}}
//
// Every non-nil error emits the structured errorDetail object so generated SDK
// clients (ErrorResponse/ErrorDetail) can unmarshal the response reliably.
func (r RequestContext) Error(err error, status int) error {
	if err == nil {
		return nil
	}

	var vErr *ValidationError
	if errors.As(err, &vErr) {
		err = r.JSON(status, map[string]any{
			"error":  errorDetail{Reason: "ValidationError", Details: vErr.Error()},
			"fields": vErr.Fields(),
		})
		if err != nil {
			return err
		}
		return vErr // Return original validation error
	}

	// If the error (or any error in its chain) implements json.Marshaler, use
	// its custom JSON representation (e.g. structured error objects like
	// {"error": {"reason": "...", "details": "..."}}).
	// errors.As walks the wrap chain, so structured errors wrapped via
	// fmt.Errorf("...: %w", structuredErr) are still found.
	var marshaler json.Marshaler
	if errors.As(err, &marshaler) {
		data, marshalErr := marshaler.MarshalJSON()
		if marshalErr == nil {
			if jsonErr := r.JSONBlob(status, data); jsonErr != nil {
				return jsonErr
			}
			return err
		}
	}

	jsonErr := r.JSON(status, errorResponseBody{
		Error: errorDetail{Reason: "Error", Details: err.Error()},
	})
	if jsonErr != nil {
		return jsonErr
	}
	return err // Return original error for test assertions
}

// errorDetail is the canonical structured error detail format declared by the
// API contract's ErrorDetail schema ({"reason": ..., "details": ...}).
type errorDetail struct {
	Reason  string `json:"reason"`
	Details string `json:"details,omitempty"`
}

// errorResponseBody wraps errorDetail under the "error" key, matching the
// canonical ErrorResponse schema.
type errorResponseBody struct {
	Error errorDetail `json:"error"`
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
