package httputil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	z "github.com/Oudwins/zog"
	"github.com/Oudwins/zog/parsers/zjson"
)

// DTOValidator interface
// DTOValidator defines the validation interface for Data Transfer Objects.
// Implementations should return a validation schema that will be applied
// to incoming requests.
type DTOValidator interface {
	Schema() *z.StructSchema
}

// Validate method
// Validate applies schema validation to the request body using the DTOValidator's schema.
// Handles request body reading and parsing, returning validation errors formatted
// as a multi-error containing all validation issues found.
func (r RequestContext) Validate(validator DTOValidator) error {
	schema := validator.Schema()

	if schema == nil {
		return fmt.Errorf("schema is nil: invalid schema")
	}

	// Get a fresh copy of the request body
	var body bytes.Buffer
	if r.Request.GetBody == nil {
		// If GetBody wasn't set, read directly from Body
		_, err := body.ReadFrom(r.Request.Body)
		if err != nil {
			return fmt.Errorf("error reading request body: %w", err)
		}
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
		
		var valErrors error
		for path, messages := range sanitized {
			if len(messages) > 0 {
				valErrors = errors.Join(errors.New(path+": "+messages[0]), valErrors)
			}
		}
		return valErrors
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
		return nil, fmt.Errorf("schema is nil: invalid schema")
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
