package httputil

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	z "github.com/Oudwins/zog"
	"go.lumeweb.com/httputil/internal/mocks"
)

func TestDecodeAndValidateRequest(t *testing.T) {
	// Create a mock request with a JSON body
	body := []byte(`{"field": "valid_value"}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Create a RequestContext
	r := Context(req, w)

	// Create instances of the DTO and model
	dto := mocks.NewMockDTORequest(t)
	model := &struct{ Field string }{Field: "initial_value"} // Set initial value

	// Mock the JSON decoding
	err := json.Unmarshal(body, model)
	if err != nil {
		t.Fatal(err)
	}

	// Set up expectations
	dto.EXPECT().ToModel(model).Run(func(m any) {
		// Simulate the mapping behavior
		*m.(*struct{ Field string }) = *model
	}).Return(nil)

	// Call the function with the new validator that has the schema setup
	err = r.DecodeAndValidateRequest(dto, model)
	if err != nil {
		t.Fatalf("DecodeAndValidateRequest failed: %v", err)
	}

	// Verify that the model was updated correctly
	if model.Field != "valid_value" {
		t.Errorf("Model field was not updated correctly: got %v, want %v", model.Field, "valid_value")
	}
}

func TestEncodeResponse(t *testing.T) {
	// Create a mock request and response writer
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Create a RequestContext
	r := Context(req, w)

	// Create instances of the model and mock DTO
	model := &struct{ Field string }{Field: "valid_value"}
	mockDTO := mocks.NewMockDTOResponse(t)
	dto := &struct{ Field string }{}

	// Set up expectations for FromModel
	mockDTO.EXPECT().FromModel(model).Return(nil).Run(func(m any) {
		*dto = *m.(*struct{ Field string })
	})

	// Call the function with the mock DTO
	r.EncodeResponse(model, mockDTO)

	// Verify that the response body is correct
	expected := "{\n\t\"Field\": \"valid_value\"\n}\n"
	if w.Body.String() != expected {
		t.Errorf("Response body is incorrect: got %v, want %v", w.Body.String(), expected)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type header is incorrect: got %v, want %v", w.Header().Get("Content-Type"), "application/json")
	}
}

func TestDecodeAndValidateRequest_DecodeError(t *testing.T) {
	// Test invalid JSON format
	body := []byte(`{"field": "123"}`) // Invalid value for string field
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r := Context(req, w)

	model := &struct{ Field string }{Field: "valid_value"}
	dto := mocks.NewMockDTOValidator(t)

	// Set up schema expectation to prevent mock panic
	dto.EXPECT().Schema().Return(nil)

	// Call the function
	err := r.DecodeAndValidateRequest(dto, model)

	if err == nil {
		t.Fatalf("DecodeAndValidateRequest should have returned a decoding error")
	}

	// We should get a JSON decoding error
	expectedErrorMessage := "schema is nil: invalid schema"
	if !strings.Contains(err.Error(), expectedErrorMessage) {
		t.Errorf("Error message is incorrect: got %v, want %v", err.Error(), expectedErrorMessage)
	}
}

func TestDecodeAndValidateRequest_ValidationError(t *testing.T) {
	// Create mock request with valid JSON but invalid data
	body := []byte(`{"field": "iv"}`) // Invalid value (too short)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}

	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r := Context(req, w)

	// Create mock dependencies
	dto := mocks.NewMockDTOValidator(t)

	// Set up expectations with proper schema using struct field name
	dto.EXPECT().Schema().Return(z.Struct(z.Schema{
		"field": z.String().Min(3), // Match JSON field name from request body
	}))

	// Call the function
	err := r.DecodeAndValidateRequest(dto, dto)

	if err == nil {
		t.Fatalf("DecodeAndValidateRequest should have returned a validation error")
	}

	// Check if the error message contains the expected validation error
	expectedErrorMessage := "string must contain at least 3 character(s)" 
	if !strings.Contains(err.Error(), expectedErrorMessage) {
		t.Errorf("Error message is incorrect: got %v, want %v", err.Error(), expectedErrorMessage)
	}
}

func TestDecodeAndValidateRequest_DTOToModelError(t *testing.T) {
	// Create a mock request with a JSON body
	body := []byte(`{"field": "valid_value"}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Create a RequestContext
	r := Context(req, w)

	// Create instances of the DTO and model
	dto := mocks.NewMockDTORequest(t)
	model := "invalid model" // Invalid model type

	// Set up expectations
	dto.EXPECT().ToModel(&model).Return(errors.New("invalid model type"))

	// Call the function
	err := r.DecodeAndValidateRequest(dto, &model)

	if err == nil {
		t.Fatalf("DecodeAndValidateRequest should have returned an error")
	}

	// Check if the error message contains the expected error
	expectedErrorMessage := "failed to map DTO to model: invalid model type"
	if err.Error() != expectedErrorMessage {
		t.Errorf("Error message is incorrect: got %v, want %v", err.Error(), expectedErrorMessage)
	}
}

func TestEncodeResponse_FromModelError(t *testing.T) {
	// Create a mock request and response writer
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Create a RequestContext
	r := Context(req, w)

	// Create instances of the model and DTO
	model := &struct{ Field string }{Field: "test"}
	dto := mocks.NewMockDTOResponse(t) // Invalid DTO type
	dto.EXPECT().FromModel(model).Return(errors.New("invalid destination type"))

	// Call the function
	r.EncodeResponse(model, dto)

	// Verify that the response status code is 500
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Response status code is incorrect: got %v, want %v", w.Code, http.StatusInternalServerError)
	}

	// Verify that the response body contains the expected error message
	expectedErrorMessage := "failed to map model to DTO: invalid destination type"
	if w.Body.String() != expectedErrorMessage+"\n" {
		t.Errorf("Response body is incorrect: got %v, want %v", w.Body.String(), expectedErrorMessage)
	}
}
