package httputil

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// DTORequest defines the interface for request Data Transfer Objects that convert
// HTTP requests to domain models. Implementations should:
// - Handle validation through the DTOValidator interface (optional)
// - Convert the raw request to a domain model
// - Return clean domain objects or validation errors
type DTORequest[M any] interface {
	ToModel() (M, error) // Returns a new model of type M
}

// DTOResponse defines the interface for response Data Transfer Objects that convert
// domain models to HTTP responses. Implementations should:
// - Format domain objects for JSON serialization
// - Maintain stable API contracts
// - Handle serialization-specific logic
type DTOResponse[M any] interface {
	FromModel(model M) error // Takes a model of type M
}

// ErrorHandler defines the strategy pattern for error handling during request processing.
// Implementations can customize error logging, formatting, and status code selection.
type ErrorHandler interface {
	HandleError(ctx RequestContext, err error)
}

// DefaultErrorHandler provides standardized error handling with:
// - 400 Bad Request for JSON syntax/type errors
// - 422 Unprocessable Entity for validation errors
// - 500 Internal Server Error for all other cases
type DefaultErrorHandler struct{}

// HandleError implements ErrorHandler with status code detection
func (h *DefaultErrorHandler) HandleError(ctx RequestContext, err error) {
	var status = http.StatusInternalServerError

	var jsonTypeErr *json.UnmarshalTypeError
	var jsonSyntaxErr *json.SyntaxError

	switch {
	case IsValidationError(err):
		status = http.StatusUnprocessableEntity
	case errors.As(err, &jsonTypeErr),
		errors.As(err, &jsonSyntaxErr):
		status = http.StatusBadRequest
	}

	_ = ctx.Error(err, status)
}

// VDOption configures DecodeAndValidateRequest processing behavior using the option pattern.
// These allow customizing validation handling per API endpoint while maintaining a consistent interface.
type VDOption func(*vdConfig)

type vdConfig struct {
	errorHandler ErrorHandler
}

// WithErrorHandler specifies a custom error handler for request processing.
// This option allows overriding the default error response strategy while
// maintaining the standard validation and decoding pipeline.
func WithErrorHandler(h ErrorHandler) VDOption {
	return func(c *vdConfig) {
		c.errorHandler = h
	}
}

// DecodeAndValidateRequest handles the complete request processing pipeline:
//  1. Decode request body to DTO
//  2. Validate using DTOValidator interface (if implemented)
//  3. Convert to domain model using ToModel()
//
// Returns:
//   - The parsed domain model and true on success
//   - Zero value and false on any error, with error handling delegated to the ErrorHandler
//
// The function uses generics to ensure type safety and accepts optional VDOptions
// to customize processing behavior per invocation.
func DecodeAndValidateRequest[M any, D DTORequest[M]](
	r RequestContext,
	dto D,
	opts ...VDOption,
) (M, bool) {
	var zero M

	// Configure defaults
	cfg := &vdConfig{
		errorHandler: &DefaultErrorHandler{},
	}

	// Apply options
	for _, opt := range opts {
		opt(cfg)
	}

	// Decode phase
	if err := r.Decode(dto); err != nil {
		cfg.errorHandler.HandleError(r, err)
		return zero, false
	}

	// Validation phase
	if validator, ok := any(dto).(DTOValidator); ok {
		if verr := r.Validate(validator); verr != nil {
			cfg.errorHandler.HandleError(r, verr)
			return zero, false
		}
	}

	// Model conversion
	model, err := dto.ToModel()
	if err != nil {
		cfg.errorHandler.HandleError(r, err)
		return zero, false
	}

	return model, true
}

// EncodeResponse handles the response generation pipeline.
// It uses generics to ensure type safety when encoding the response.
func EncodeResponse[M any, D DTOResponse[M]](r RequestContext, model M, dto D) error {
	if err := dto.FromModel(model); err != nil {
		return r.Error(fmt.Errorf("failed to map model to DTO: %w", err), http.StatusInternalServerError)
	}
	r.Encode(dto)
	return nil
}
