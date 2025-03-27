package httputil

import (
	"fmt"
	"net/http"
)

// DTORequest defines the interface for Data Transfer Objects (DTOs) that convert
// incoming requests to domain models. Implementations should handle validation
// and conversion logic.
type DTORequest interface {
	ToModel(model any) error // Maps DTO to an existing model
}

// DTOResponse defines the interface for DTOs that convert domain models to
// API responses. Implementations should handle serialization and formatting.
type DTOResponse interface {
	FromModel(model any) error // Maps from a model to the DTO
}

// DecodeAndValidateRequest function
// DecodeAndValidateRequest handles the complete request processing pipeline:
// 1. Decodes request body into DTO
// 2. Validates DTO if it implements DTOValidator
// 3. Maps DTO to domain model if DTO implements DTORequest
// Returns error at first failure in the pipeline
func (r RequestContext) DecodeAndValidateRequest(dto any, model any) error {
	if err := r.Decode(dto); err != nil {
		return err
	}

	// Check if the DTO implements DTOValidator and, if so, validate it.
	if validator, ok := dto.(DTOValidator); ok {
		if err := r.Validate(validator); err != nil {
			return err // Validation errors handled by r.Validate
		}
	}

	// Check if the DTO implements DTORequest and, if so, convert to model.
	if request, ok := dto.(DTORequest); ok {
		if err := request.ToModel(model); err != nil {
			return r.Check("failed to map DTO to model", err)
		}
	}

	return nil
}

// EncodeResponse function
// EncodeResponse handles the response generation pipeline:
// 1. Maps domain model to DTO using DTOResponse implementation
// 2. Encodes DTO to JSON response
// Sends 500 error if mapping fails
func (r RequestContext) EncodeResponse(model any, dto DTOResponse) {
	if err := dto.FromModel(model); err != nil {
		_ = r.Error(fmt.Errorf("failed to map model to DTO: %w", err), http.StatusInternalServerError)
		return
	}
	// Encode the model directly since we already mapped it to the DTO
	r.Encode(model)
}
