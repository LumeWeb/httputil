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

// Validate applies schema validation to the request body using zog schemas.
// Returns:
// - nil on successful validation
// - ValidationError with field-specific errors on failure
// - Error wrapping original parse failure if schema validation cannot be performed
func (r RequestContext) Validate(validator DTOValidator) error {
	schema := validator.Schema()

	if schema == nil {
		return &ValidationError{
			FieldErrors: map[string]string{
				"": "schema is nil: invalid schema",
			},
			joinedError: errors.New("schema is nil: invalid schema"),
		}
	}

	// Get a fresh copy of the request body
	var body bytes.Buffer
	if r.Request.GetBody == nil {
		// Read and restore body when GetBody isn't available
		bodyContent, err := io.ReadAll(r.Request.Body)
		if err != nil {
			return fmt.Errorf("error reading request body: %w", err)
		}
		// Restore original body from the read content
		r.Request.Body = io.NopCloser(bytes.NewReader(bodyContent))
		// Copy into validation buffer
		body.Write(bodyContent)
	} else {
		// Use GetBody to get a fresh copy
		bodyCopy, err := r.Request.GetBody()
		if err != nil {
			return fmt.Errorf("error getting request body: %w", err)
		}
		_, err = body.ReadFrom(bodyCopy)
		if err != nil {
			return fmt.Errorf("error reading request body: %w", err)
		}
	}

	// Parse returns []ZogIssue and handles validation
	issues := schema.Parse(zjson.Decode(&body), validator)
	if len(issues) > 0 {
		// Use Zog's built-in sanitizer to convert issues to a simple map
		sanitized := z.Issues.SanitizeMap(issues)

		fieldErrors := make(map[string]string, len(sanitized))
		var errs []error

		for path, messages := range sanitized {
			if len(messages) > 0 {
				msg := fmt.Sprintf("%s: %s", path, messages[0])
				fieldErrors[path] = msg
				errs = append(errs, errors.New(msg))
			}
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
	// Parse the request body using zog
	errs := schema.Parse(zjson.Decode(r.Request.Body), entity)
	if errs != nil {
		// Use Zog's built-in sanitizer to convert errors to a simple map
		// This avoids any type assertions and uses the official API
		sanitized := z.Issues.SanitizeMap(errs)

		// Convert from map[string][]string to map[string]string by taking the first error for each field
		validationErrors := make(map[string]string)
		for path, messages := range sanitized {
			if len(messages) > 0 {
				validationErrors[path] = messages[0]
			}
		}

		return validationErrors, nil
	}
	return nil, nil
}

// ToJSON is a generic helper function that marshals data to JSON.
// Provides consistent JSON encoding across the package.
func ToJSON[T any](data T) ([]byte, error) {
	return json.Marshal(data)
}
