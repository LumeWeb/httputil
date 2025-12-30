package httputil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	z "github.com/Oudwins/zog"
	"github.com/Oudwins/zog/parsers/zjson"
	"io"
	"strings"
)

// DTOValidator defines the validation interface for Data Transfer Objects.
// Implementations should return a zog schema that will be applied to validate
// incoming requests. The schema is used to:
// - Validate request structure
// - Provide meaningful error messages
// - Ensure data integrity before processing
type DTOValidator interface {
	Schema() *z.StructSchema
}

// ValidationError represents structured validation failures with:
// - Field-specific error messages
// - Machine-readable error codes
// - Nested error support through error wrapping
type ValidationError struct {
	FieldErrors map[string]string // Field path -> error message
	joinedError error             // Pre-joined error for logging
}

func (v *ValidationError) Error() string {
	if v.joinedError != nil {
		return v.joinedError.Error()
	}
	// Fallback to field errors if no joined error
	var msgs []string
	for _, msg := range v.FieldErrors {
		msgs = append(msgs, msg)
	}
	return strings.Join(msgs, ", ")
}

func (v *ValidationError) Unwrap() []error {
	errs := make([]error, 0, len(v.FieldErrors))
	for _, msg := range v.FieldErrors {
		errs = append(errs, errors.New(msg))
	}
	return errs
}

// Fields returns the field-specific error messages
func (v *ValidationError) Fields() map[string]string {
	return v.FieldErrors
}

// IsValidationError checks if an error is or contains a ValidationError
func IsValidationError(err error) bool {
	var verr *ValidationError
	return errors.As(err, &verr)
}

// Validate applies schema validation against a pre-populated DTOValidator.
// Returns:
// - nil on successful validation
// - ValidationError with field-specific errors on failure
// - Error wrapping original parse failure if schema validation cannot be performed
func (r RequestContext) Validate(validator DTOValidator) error {
	if validator == nil {
		return &ValidationError{
			FieldErrors: map[string]string{
				"": "validator is nil: invalid validator",
			},
			joinedError: errors.New("validator is nil: invalid validator"),
		}
	}

	schema := validator.Schema()
	if schema == nil {
		return &ValidationError{
			FieldErrors: map[string]string{
				"": "schema is nil: invalid schema",
			},
			joinedError: errors.New("schema is nil: invalid schema"),
		}
	}

	// Validate returns []ZogIssue and handles validation using the request body
	issues := schema.Validate(validator)
	if len(issues) > 0 {
		fieldErrors := make(map[string]string, len(issues))
		var errs []error

		for _, issue := range issues {
			path := issue.PathString()
			if len(issue.Message) == 0 {
				continue
			}
			var msg string
			if len(path) > 0 {
				msg = fmt.Sprintf("%s: %s", path, issue.Message)
			} else {
				msg = issue.Message
			}
			fieldErrors[path] = msg
			errs = append(errs, errors.New(msg))
		}

		return &ValidationError{
			FieldErrors: fieldErrors,
			joinedError: errors.New("validation failed"),
		}
	}
	return nil
}

// ValidateRequest performs validation and returns errors as a map of field->message.
// This is useful when you need to present validation errors in a structured format.
// Returns:
// - (nil, nil) if validation passes
// - (map, nil) with validation errors if validation fails
// - (nil, error) if there was a processing error
func (r RequestContext) ValidateRequest(entity DTOValidator) (map[string]string, error) {
	schema := entity.Schema()
	if schema == nil {
		return nil, &ValidationError{
			FieldErrors: map[string]string{
				"": "schema is nil: invalid schema",
			},
			joinedError: errors.New("schema is nil: invalid schema"),
		}
	}

	body, err := r.readRequestBody()
	if err != nil {
		return nil, fmt.Errorf("error reading request body: %w", err)
	}

	errs := schema.Parse(zjson.Decode(body), entity)
	if errs != nil {
		validationErrors := make(map[string]string)
		for _, issue := range errs {
			path := issue.PathString()
			if len(issue.Message) == 0 {
				continue
			}
			validationErrors[path] = issue.Message
		}
		return validationErrors, nil
	}
	return nil, nil
}

func (r RequestContext) readRequestBody() (*bytes.Buffer, error) {
	var body bytes.Buffer

	if r.Request().GetBody == nil {
		// Read and cache the original body content
		bodyContent, err := io.ReadAll(r.Request().Body)
		if err != nil {
			return nil, err
		}

		// Replace body with seekable version
		r.Request().Body = newSeekableBuffer(bodyContent)

		// Set GetBody factory
		r.Request().GetBody = getBodyFactory(bodyContent)

		// Store content in our buffer
		body.Write(bodyContent)
	} else {
		// Use existing GetBody functionality
		bodyCopy, err := r.Request().GetBody()
		if err != nil {
			return nil, err
		}
		defer func(bodyCopy io.ReadCloser) {
			err := bodyCopy.Close()
			if err != nil {
				r.Context.Logger().Error(err)
			}
		}(bodyCopy)

		if _, err := body.ReadFrom(bodyCopy); err != nil {
			return nil, err
		}
	}

	return &body, nil
}

// ToJSON is a generic helper function that marshals data to JSON.
// Provides consistent JSON encoding across the package.
func ToJSON[T any](data T) ([]byte, error) {
	return json.Marshal(data)
}

// newSeekableBuffer creates a new seekable ReadCloser from the given content
func newSeekableBuffer(content []byte) io.ReadCloser {
	return &seekableBuffer{
		Reader: bytes.NewReader(content),
	}
}

// seekableBuffer implements io.ReadCloser with seeking support
type seekableBuffer struct {
	*bytes.Reader
}

func (s *seekableBuffer) Close() error {
	return nil
}

// getBodyFactory creates a GetBody function for the given content
func getBodyFactory(content []byte) func() (io.ReadCloser, error) {
	return func() (io.ReadCloser, error) {
		return newSeekableBuffer(content), nil
	}
}
