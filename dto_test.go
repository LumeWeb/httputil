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
	dto := mocks.NewMockDTORequest[*struct{ Field string }](t)
	model := &struct{ Field string }{Field: "valid_value"}

	// Set up expectations
	dto.EXPECT().ToModel().Return(model, nil)

	// Call the function with the new validator that has the schema setup
	modelResult, err := DecodeAndValidateRequest(r, dto)
	if err != nil {
		t.Fatalf("DecodeAndValidateRequest failed: %v", err)
	}

	// Verify that the model was updated correctly
	if modelResult.Field != "valid_value" {
		t.Errorf("Model field was not updated correctly: got %v, want %v", modelResult.Field, "valid_value")
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
	mockDTO := mocks.NewMockDTOResponse[*struct{ Field string }](t)

	// Set up expectations for FromModel and encode the model
	realDTO := &struct{ Field string }{}
	mockDTO.EXPECT().FromModel(model).Return(nil).Run(func(m *struct{ Field string }) {
		*realDTO = *m
	})

	// Call the function with the mock DTO
	_ = EncodeResponse(r, model, mockDTO)

	// Encode the real DTO to get the expected JSON
	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")
	_ = enc.Encode(realDTO)

	// Call the function with the mock DTO
	_ = EncodeResponse(r, model, mockDTO)

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
	// Test invalid JSON format (number instead of string)
	body := []byte(`{"field": 123}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r := Context(req, w)

	// Use a concrete DTO type that expects a string field
	type testDTO struct {
		Field string `json:"field"`
	}
	dto := mocks.NewMockDTORequest[*testDTO](t)

	// Call the function - should fail at decode step
	_, err := DecodeAndValidateRequest(r, dto)

	if err == nil {
		t.Fatal("DecodeAndValidateRequest should have returned a decoding error")
	}

	// Verify we got the JSON unmarshal error
	expectedErrorMessage := "cannot unmarshal number into Go struct field"
	if !strings.Contains(err.Error(), expectedErrorMessage) {
		t.Errorf("Error message mismatch:\ngot: %v\nwant: %v", err.Error(), expectedErrorMessage)
	}
}

func TestDecodeAndValidateRequest_ValidationError(t *testing.T) {
	// Create mock request with valid JSON but invalid data
	body := []byte(`{"field": "iv"}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r := Context(req, w)

	// Create mock validator that implements DTOValidator
	validator := mocks.NewMockDTOValidator(t)
	schema := z.Struct(z.Schema{
		"field": z.String().Min(3).Required(), // Match nested field path
	})
	validator.EXPECT().Schema().Return(schema)

	dto := &struct {
		*mocks.MockDTORequest[*struct{ Field string }]
		*mocks.MockDTOValidator
		Field string `json:"field"`
	}{
		MockDTORequest:   mocks.NewMockDTORequest[*struct{ Field string }](t),
		MockDTOValidator: validator,
		Field:            "",
	}

	// Set up expectations
	dto.MockDTOValidator.EXPECT().Schema().Return(schema)

	// Call the function
	_, err := DecodeAndValidateRequest(r, dto)

	if err == nil {
		t.Fatalf("DecodeAndValidateRequest should have returned a validation error")
	}

	// Check structured error handling
	var vErr *ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("Expected ValidationError, got %T", err)
	}

	// Check if the error message contains the expected validation error
	expectedErrorMessage := "validation failed"
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
	dto := mocks.NewMockDTORequest[*struct{ Field string }](t)

	// Set up expectations
	dto.EXPECT().ToModel().Return(&struct{ Field string }{}, errors.New("invalid model type"))

	// Call the function
	_, err := DecodeAndValidateRequest(r, dto)

	if err == nil {
		t.Fatalf("DecodeAndValidateRequest should have returned an error")
	}

	// Check if the error message contains the expected error
	expectedErrorMessage := "invalid model type"
	if !strings.Contains(err.Error(), expectedErrorMessage) {
		t.Errorf("Error message should contain: %q, got %q", expectedErrorMessage, err.Error())
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
	dto := mocks.NewMockDTOResponse[*struct{ Field string }](t)
	dto.EXPECT().FromModel(model).Return(errors.New("invalid destination type"))

	// Call the function
	_ = EncodeResponse(r, model, dto)

	// Verify that the response status code is 500
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Response status code is incorrect: got %v, want %v", w.Code, http.StatusInternalServerError)
	}

	// Verify that the response body contains the expected error message
	expectedErrorMessage := "failed to map model to DTO: invalid destination type"
	if !strings.Contains(w.Body.String(), expectedErrorMessage) {
		t.Errorf("Response body should contain: %q, got %q", expectedErrorMessage, w.Body.String())
	}
}

type simpleModel struct {
	Name string
}

type simpleDTO struct {
	Name string `json:"name"`
}

func (d *simpleDTO) ToModel() (simpleModel, error) {
	return simpleModel{Name: d.Name}, nil
}

func TestDecodeAndValidateRequest_WithValidator(t *testing.T) {
	body := []byte(`{"field": "valid"}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	w := httptest.NewRecorder()
	r := Context(req, w)

	// Create mock validator
	validator := mocks.NewMockDTOValidator(t)
	schema := z.Struct(z.Schema{
		"field": z.String().Min(3).Required(),
	})
	validator.EXPECT().Schema().Return(schema)

	// Create mock DTO that implements both DTORequest and DTOValidator
	dto := &struct {
		*mocks.MockDTORequest[*struct{ Field string }]
		*mocks.MockDTOValidator
		Field string `json:"field"`
	}{
		MockDTORequest:   mocks.NewMockDTORequest[*struct{ Field string }](t),
		MockDTOValidator: validator,
	}

	// Set up expectations
	model := &struct{ Field string }{Field: "valid"}
	dto.MockDTORequest.EXPECT().ToModel().Return(model, nil)

	result, err := DecodeAndValidateRequest(r, dto)
	if err != nil {
		t.Fatalf("Unexpected validation error: %v", err)
	}
	if result.Field != "valid" {
		t.Errorf("Expected model field 'valid', got %q", result.Field)
	}
}

func TestDecodeAndValidateRequest_WithValidatorError(t *testing.T) {
	body := []byte(`{"field": "iv"}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r := Context(req, w)

	// Create mock validator and DTO
	dto := &mockDto{}

	_, err := DecodeAndValidateRequest(r, dto)
	if err == nil {
		t.Fatal("Expected validation error, got nil")
	}
	var vErr *ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("Expected ValidationError, got %T", err)
	}

	expectedErr := "string must contain at least 3 character(s)"
	if !strings.Contains(vErr.Fields()["field"], expectedErr) {
		t.Errorf("Expected field error to contain %q, got %q", expectedErr, vErr.Fields()["field"])
	}
}

func TestDecodeAndValidateRequest_WithoutValidator(t *testing.T) {
	body := []byte(`{"name": "test"}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r := Context(req, w)

	dto := &simpleDTO{}
	model, err := DecodeAndValidateRequest[simpleModel](r, dto)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if model.Name != "test" {
		t.Errorf("Expected model name 'test', got %q", model.Name)
	}
}

var _ DTOValidator = (*mockDto)(nil)
var _ DTORequest[*mockModel] = (*mockDto)(nil)

type mockDto struct {
	Field string `json:"field"`
}

func (m mockDto) ToModel() (*mockModel, error) {
	return &mockModel{Field: m.Field}, nil
}

type mockModel struct {
	Field string
}

func (m mockDto) Schema() *z.StructSchema {
	return z.Struct(z.Schema{
		"field": z.String().Min(3).Required(),
	})
}
