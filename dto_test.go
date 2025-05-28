package httputil

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/labstack/echo/v4"
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

	e := echo.New()
	// Create a RequestContext from an Echo context
	r := Context(e.NewContext(req, w))

	// Create instances of the DTO and model
	dto := mocks.NewMockDTORequest[*struct{ Field string }](t)
	model := &struct{ Field string }{Field: "valid_value"}

	// Set up expectations
	dto.EXPECT().ToModel().Return(model, nil)

	// Call the function with the new validator that has the schema setup
	modelResult, ok := DecodeAndValidateRequest(r, dto)
	if !ok {
		t.Fatal("DecodeAndValidateRequest failed")
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

	e := echo.New()
	// Create a RequestContext from an Echo context
	r := Context(e.NewContext(req, w))

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
	e := echo.New()
	r := Context(e.NewContext(req, w))

	// Create mock validator and DTO that implements both interfaces
	type testDTO struct {
		*mocks.MockDTORequest[*struct{ Field string }]
		*mocks.MockDTOValidator
		Field string `json:"field"`
	}

	dto := &testDTO{
		MockDTORequest:   mocks.NewMockDTORequest[*struct{ Field string }](t),
		MockDTOValidator: mocks.NewMockDTOValidator(t),
	}

	// Call the function - should fail at decode step before Schema() is called
	_, ok := DecodeAndValidateRequest[*struct{ Field string }](r, dto)

	// Verify Schema() was never called since decode failed first
	dto.MockDTOValidator.AssertNotCalled(t, "Schema")
	if ok {
		t.Fatal("DecodeAndValidateRequest should have failed")
	}

	// Verify response status and error format
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !strings.Contains(resp["error"].(string), "cannot unmarshal number into Go struct field") {
		t.Errorf("Error message mismatch:\ngot: %v\nwant: %v", resp["error"], "cannot unmarshal number into Go struct field")
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
	e := echo.New()
	r := Context(e.NewContext(req, w))

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
	_, ok := DecodeAndValidateRequest(r, dto)
	if ok {
		t.Fatal("DecodeAndValidateRequest should have failed")
	}

	// Verify response status and error format
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status code %d, got %d", http.StatusUnprocessableEntity, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp["error"] != "validation failed" {
		t.Errorf("Expected error 'validation failed', got %v", resp["error"])
	}

	fields := resp["fields"].(map[string]interface{})
	if len(fields) == 0 {
		t.Error("Expected field errors in response")
	}
}

func TestDecodeAndValidateRequest_DTOToModelError(t *testing.T) {
	// Create a mock request with a JSON body
	body := []byte(`{"field": "valid_value"}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Create a RequestContext
	e := echo.New()
	r := Context(e.NewContext(req, w))

	// Create instances of the DTO and model
	dto := mocks.NewMockDTORequest[*struct{ Field string }](t)

	// Set up expectations
	dto.EXPECT().ToModel().Return(&struct{ Field string }{}, errors.New("invalid model type"))

	// Call the function
	_, ok := DecodeAndValidateRequest(r, dto)
	if ok {
		t.Fatal("DecodeAndValidateRequest should have failed")
	}

	// Verify response status and error format
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !strings.Contains(resp["error"].(string), "invalid model type") {
		t.Errorf("Error message should contain: %q, got %q", "invalid model type", resp["error"])
	}
}

func TestEncodeResponse_FromModelError(t *testing.T) {
	// Create a mock request and response writer
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	e := echo.New()
	// Create a RequestContext from an Echo context
	r := Context(e.NewContext(req, w))

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

func TestDecodeAndValidateRequest_WithValidatorAndGetBody(t *testing.T) {
	body := []byte(`{"field": "valid"}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	w := httptest.NewRecorder()
	e := echo.New()
	r := Context(e.NewContext(req, w))

	// Create mock validator
	validator := mocks.NewMockDTOValidator(t)
	schema := z.Struct(z.Schema{
		"Field": z.String().Min(3).Required(), // Match struct field name
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
	dto.MockDTORequest.EXPECT().ToModel().Return(model, nil).Maybe()
	dto.MockDTOValidator.EXPECT().Schema().Return(schema)

	result, ok := DecodeAndValidateRequest(r, dto)
	if !ok {
		t.Fatal("Unexpected validation failure")
	}
	if result.Field != "valid" {
		t.Errorf("Expected model field 'valid', got %q", result.Field)
	}
}

func TestDecodeAndValidateRequest_WithValidatorNoGetBody(t *testing.T) {
	body := []byte(`{"field": "valid"}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e := echo.New()
	r := Context(e.NewContext(req, w))

	// Create mock validator
	validator := mocks.NewMockDTOValidator(t)
	schema := z.Struct(z.Schema{
		"Field": z.String().Min(3).Required(), // Match struct field name
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
	dto.MockDTORequest.EXPECT().ToModel().Return(model, nil).Maybe()
	dto.MockDTOValidator.EXPECT().Schema().Return(schema)

	result, ok := DecodeAndValidateRequest(r, dto)
	if !ok {
		t.Fatal("Unexpected validation failure")
	}
	if result.Field != "valid" {
		t.Errorf("Expected model field 'valid', got %q", result.Field)
	}
}

func TestDecodeAndValidateRequest_WithValidatorErrorAndGetBody(t *testing.T) {
	body := []byte(`{"field": "iv"}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e := echo.New()
	r := Context(e.NewContext(req, w))

	validator := mocks.NewMockDTOValidator(t)
	schema := z.Struct(z.Schema{
		"field": z.String().Min(3).Required(),
	})
	validator.EXPECT().Schema().Return(schema)

	// Create mock DTO that implements both interfaces
	dto := &struct {
		*mocks.MockDTORequest[*struct{ Field string }]
		*mocks.MockDTOValidator
		Field string `json:"field"`
	}{
		MockDTORequest:   mocks.NewMockDTORequest[*struct{ Field string }](t),
		MockDTOValidator: validator,
	}

	_, ok := DecodeAndValidateRequest(r, dto)
	if ok {
		t.Fatal("Expected validation failure, got success")
	}

	// Verify response status and error format
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status code %d, got %d", http.StatusUnprocessableEntity, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp["error"] == nil {
		t.Error("Expected error message in response")
	}

	fields, ok := resp["fields"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'fields' to be a map in the response")
	}
	if len(fields) == 0 {
		t.Error("Expected field errors in response")
	}
}

func TestDecodeAndValidateRequest_WithValidatorErrorNoGetBody(t *testing.T) {
	body := []byte(`{"field": "iv"}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e := echo.New()
	r := Context(e.NewContext(req, w))

	validator := mocks.NewMockDTOValidator(t)
	schema := z.Struct(z.Schema{
		"field": z.String().Min(3).Required(),
	})
	validator.EXPECT().Schema().Return(schema)

	// Create mock DTO that implements both interfaces
	dto := &struct {
		*mocks.MockDTORequest[*struct{ Field string }]
		*mocks.MockDTOValidator
		Field string `json:"field"`
	}{
		MockDTORequest:   mocks.NewMockDTORequest[*struct{ Field string }](t),
		MockDTOValidator: validator,
	}

	_, ok := DecodeAndValidateRequest(r, dto)
	if ok {
		t.Fatal("Expected validation failure, got success")
	}

	// Verify response status and error format
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status code %d, got %d", http.StatusUnprocessableEntity, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp["error"] == nil {
		t.Error("Expected error message in response")
	}

	fields, ok := resp["fields"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'fields' to be a map in the response")
	}
	if len(fields) == 0 {
		t.Error("Expected field errors in response")
	}
}

func TestDecodeAndValidateRequest_WithoutValidator(t *testing.T) {
	body := []byte(`{"name": "test"}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e := echo.New()
	r := Context(e.NewContext(req, w))

	dto := &simpleDTO{}
	model, ok := DecodeAndValidateRequest[simpleModel](r, dto)
	if !ok {
		t.Fatal("Unexpected validation failure")
	}
	if model.Name != "test" {
		t.Errorf("Expected model name 'test', got %q", model.Name)
	}
}
