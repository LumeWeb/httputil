package httputil

import (
	"errors"
	"fmt"
	"net/http"
)

// DTORequest defines the interface for Data Transfer Objects (DTOs) that convert
// incoming requests to domain models. Implementations should handle validation
// and conversion logic.
type DTORequest[M any] interface {
	ToModel() (M, error) // Returns a new model of type M
}

// DTOResponse defines the interface for DTOs that convert domain models to
// API responses. Implementations should handle serialization and formatting.
type DTOResponse[M any] interface {
	FromModel(model M) error // Takes a model of type M
}

// DecodeAndValidateRequest handles the complete request processing pipeline.
// It uses generics to ensure type safety when decoding and validating
func DecodeAndValidateRequest[M any, D DTORequest[M]](r RequestContext, dto D) (M, error) {
	var zero M

	if err := r.Decode(dto); err != nil {
		return zero, err
	}

	// Check if the DTO implements DTOValidator and, if so, validate it.
	if validator, ok := any(dto).(DTOValidator); ok {
		if verr := r.Validate(validator); verr != nil {
			// Handle validation errors with structured response
			var vErr *ValidationError
			if errors.As(verr, &vErr) {
				return zero, r.Error(vErr, http.StatusUnprocessableEntity)
			}
			return zero, verr
		}
	}

	return dto.ToModel()
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
