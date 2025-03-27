package httputil

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"

	z "github.com/Oudwins/zog"
	"go.lumeweb.com/httputil/internal/mocks"
)

func TestValidate_ValidRequest(t *testing.T) {
	// Create mock request with valid JSON body
	body := []byte(`{"field": "valid_value"}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r := Context(req, w)

	validator := mocks.NewMockDTOValidator(t)
	schema := z.Struct(z.Schema{
		"field": z.String().Min(3),
	})

	validator.EXPECT().Schema().Return(schema)

	err := r.Validate(validator)
	if err != nil {
		t.Errorf("Validate returned error for valid request: %v", err)
	}
}

func TestValidate_InvalidRequest(t *testing.T) {
	// Create mock request with invalid JSON body
	body := []byte(`{"field": "iv"}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r := Context(req, w)

	validator := mocks.NewMockDTOValidator(t)
	schema := z.Struct(z.Schema{
		"field": z.String().Min(3),
	})

	validator.EXPECT().Schema().Return(schema)

	err := r.Validate(validator)
	if err == nil {
		t.Fatal("Validate should have returned error for invalid request")
	}

	expected := "field must be at least 3 characters"
	if strings.Contains(err.Error(), expected) {
		t.Errorf("Expected error %q, got %q", expected, err.Error())
	}
}

func TestValidate_NilSchema(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r := Context(req, w)

	validator := mocks.NewMockDTOValidator(t)
	validator.EXPECT().Schema().Return(nil)

	err := r.Validate(validator)
	if err == nil {
		t.Fatal("Validate should have returned error for nil schema")
	}

	expected := "schema is nil: invalid schema"
	if err.Error() != expected {
		t.Errorf("Expected error %q, got %q", expected, err.Error())
	}
}

func TestValidateRequest_Valid(t *testing.T) {
	body := []byte(`{"field": "valid_value"}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r := Context(req, w)

	validator := mocks.NewMockDTOValidator(t)
	schema := z.Struct(z.Schema{
		"field": z.String().Min(3),
	})

	validator.EXPECT().Schema().Return(schema)

	validationErrors, err := r.ValidateRequest(validator)
	if err != nil {
		t.Fatalf("ValidateRequest returned unexpected error: %v", err)
	}
	if validationErrors != nil {
		t.Fatalf("Expected no validation errors, got %v", validationErrors)
	}
}

func TestValidateRequest_Invalid(t *testing.T) {
	body := []byte(`{"field": "iv"}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r := Context(req, w)

	validator := mocks.NewMockDTOValidator(t)
	schema := z.Struct(z.Schema{
		"field": z.String().Min(3),
	})

	validator.EXPECT().Schema().Return(schema)

	validationErrors, err := r.ValidateRequest(validator)
	if err != nil {
		t.Fatalf("ValidateRequest returned unexpected error: %v", err)
	}
	if validationErrors == nil {
		t.Fatal("Expected validation errors, got nil")
	}

	expected := "field must be at least 3 characters"
	if strings.Contains(validationErrors["field"], expected) {
		t.Errorf("Expected error %q, got %q", expected, validationErrors["field"])
	}
}

func TestToJSON(t *testing.T) {
	// Create a mock data
	data := map[string]string{"field": "value"}

	// Call the ToJSON function
	jsonBytes, err := ToJSON(data)
	if err != nil {
		t.Errorf("ToJSON returned an error: %v", err)
	}

	// Verify the result
	expectedJSON := `{"field":"value"}`
	if string(jsonBytes) != expectedJSON {
		t.Errorf("ToJSON returned incorrect JSON: got %v, want %v", string(jsonBytes), expectedJSON)
	}

	// Test with a struct
	type TestStruct struct {
		Field string `json:"field"`
	}
	dataStruct := TestStruct{Field: "value"}
	jsonBytes, err = ToJSON(dataStruct)
	if err != nil {
		t.Errorf("ToJSON returned an error: %v", err)
	}
	expectedJSON = `{"field":"value"}`
	if string(jsonBytes) != expectedJSON {
		t.Errorf("ToJSON returned incorrect JSON: got %v, want %v", string(jsonBytes), expectedJSON)
	}
}

func TestValidateRequest_NilSchema(t *testing.T) {
	// Create a mock request with a JSON body
	body := []byte(`{"field": "abc"}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Create a RequestContext
	r := Context(req, w)

	// Create an instance of the DTOValidator with a nil schema
	validator := mocks.NewMockDTOValidator(t)

	// Set up expectations
	validator.EXPECT().Schema().Return(nil)

	// Call the ValidateRequest function
	validationErrors, err := r.ValidateRequest(validator)

	// Verify that the function returns an error
	if err == nil {
		t.Errorf("ValidateRequest should have returned an error for nil schema")
	}

	// Verify that the error message is correct
	expectedErrorMessage := "schema is nil: invalid schema"
	if err != nil && err.Error() != expectedErrorMessage {
		t.Errorf("Error message is incorrect: got %v, want %v", err.Error(), expectedErrorMessage)
	}

	// Verify that the validationErrors is nil
	if validationErrors != nil {
		t.Errorf("ValidateRequest should not have returned validation errors for nil schema")
	}
}

func TestValidate_EmptyField(t *testing.T) {
	// Create a mock request and response writer
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Create a RequestContext
	r := Context(req, w)

	// Create an instance of the DTOValidator
	validator := mocks.NewMockDTOValidator(t)

	schema := z.Struct(z.Schema{
		"field": z.String().Min(3), // Match JSON field name
	})
	// Set up expectations
	validator.EXPECT().Schema().Return(schema)

	// Call the Validate function
	err := r.Validate(validator)

	if err == nil {
		t.Errorf("Validate should have returned an error for invalid data")
	}

	expectedErrorMessage := "field must be at least 3 characters"
	if err != nil && strings.Contains(err.Error(), expectedErrorMessage) {
		t.Errorf("Error message is incorrect: got %v, want %v", err.Error(), expectedErrorMessage)
	}
}
