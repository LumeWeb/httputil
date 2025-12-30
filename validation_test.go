package httputil

import (
	"bytes"
	"errors"
	"github.com/labstack/echo/v4"
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
	e := echo.New()
	r := Context(e.NewContext(req, w))

	validator := mocks.NewMockDTOValidator(t)
	validator.Field = "valid_value"
	schema := z.Struct(z.Schema{
		"Field": z.String().Min(3).Required(), // Match struct field name
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
	e := echo.New()
	r := Context(e.NewContext(req, w))

	validator := mocks.NewMockDTOValidator(t)
	validator.Field = "iv"
	schema := z.Struct(z.Schema{
		"Field": z.String().Min(3).Required(),
	})

	validator.EXPECT().Schema().Return(schema)

	err := r.Validate(validator)
	if err == nil {
		t.Fatal("Validate should have returned error for invalid request")
	}

	var vErr *ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("Expected ValidationError, got %T", err)
	}

	expected := "string must contain at least 3 character(s)"
	if !strings.Contains(vErr.Fields()["Field"], expected) {
		t.Errorf("Expected error to contain %q, got %q", expected, vErr.Fields()["Field"])
	}
}

func TestValidate_NilSchema(t *testing.T) {
	// Create a request with empty field value
	body := []byte(`{"field": ""}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e := echo.New()
	r := Context(e.NewContext(req, w))

	validator := mocks.NewMockDTOValidator(t)
	validator.EXPECT().Schema().Return(nil)

	err := r.Validate(validator)
	if err == nil {
		t.Fatal("Validate should have returned error for nil schema")
	}

	expected := "schema is nil: invalid schema"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("Expected error to contain %q, got %q", expected, err.Error())
	}
}

func TestValidateRequest_Valid(t *testing.T) {
	body := []byte(`{"field": "valid_value"}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e := echo.New()
	r := Context(e.NewContext(req, w))

	validator := mocks.NewMockDTOValidator(t)
	schema := z.Struct(z.Schema{
		"field": z.String().Min(3).Required(), // Lowercase to match JSON field name
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
	e := echo.New()
	r := Context(e.NewContext(req, w))

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

	expected := "string must contain at least 3 character(s)"
	if !strings.Contains(validationErrors["field"], expected) {
		t.Errorf("Expected error to contain %q, got %q", expected, validationErrors["field"])
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
	e := echo.New()
	r := Context(e.NewContext(req, w))

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
	var vErr *ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("Expected ValidationError, got %T", err)
	}

	expected := "schema is nil: invalid schema"
	if !strings.Contains(vErr.Fields()[""], expected) {
		t.Errorf("Error message should contain %q, got %q", expected, vErr.Fields()[""])
	}

	// Verify that the validationErrors is nil
	if validationErrors != nil {
		t.Errorf("ValidateRequest should not have returned validation errors for nil schema")
	}
}

func TestValidate_EmptyField(t *testing.T) {
	// Create a request with empty field value
	body := []byte(`{"field": "a"}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e := echo.New()
	r := Context(e.NewContext(req, w))

	// Create and setup validator mock
	validator := mocks.NewMockDTOValidator(t)
	validator.Field = "a"
	schema := z.Struct(z.Schema{
		"Field": z.String().Min(3).Required(),
	})
	validator.EXPECT().Schema().Return(schema)

	// Call the Validate function
	err := r.Validate(validator)
	if err == nil {
		t.Fatal("Expected validation error")
	}

	var vErr *ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("Expected ValidationError, got %T", err)
	}

	expected := "string must contain at least 3 character(s)"
	if !strings.Contains(vErr.Fields()["Field"], expected) {
		t.Errorf("Error message should contain %q, got %q", expected, vErr.Fields()["Field"])
	}
}

func TestValidationError_Unwrap(t *testing.T) {
	errs := []error{
		errors.New("error1"),
		errors.New("error2"),
	}
	vErr := &ValidationError{
		FieldErrors: map[string]string{
			"field1": "error1",
			"field2": "error2",
		},
		joinedError: errors.Join(errs...),
	}

	unwrapped := vErr.Unwrap()
	if len(unwrapped) != 2 {
		t.Fatalf("Expected 2 unwrapped errors, got %d", len(unwrapped))
	}
	// Check contains both errors regardless of order
	found1 := false
	found2 := false
	for _, err := range unwrapped {
		if err.Error() == "error1" {
			found1 = true
		}
		if err.Error() == "error2" {
			found2 = true
		}
	}
	if !found1 || !found2 {
		t.Errorf("Unwrapped errors mismatch: got %v", unwrapped)
	}
}

func TestValidationError_MultipleFields(t *testing.T) {
	body := []byte(`{"field1": "a", "field2": ""}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e := echo.New()
	r := Context(e.NewContext(req, w))

	validator := mocks.NewMockDTOValidator(t)
	validator.Field1 = "a"
	validator.Field2 = ""
	schema := z.Struct(z.Schema{
		"Field1": z.String().Min(3), // Match struct field name case
		"Field2": z.String().Required(),
	})
	validator.EXPECT().Schema().Return(schema)

	err := r.Validate(validator)
	if err == nil {
		t.Fatal("Expected validation error")
	}

	vErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("Expected ValidationError, got %T", err)
	}

	expectedErrors := map[string]string{
		"Field1": "string must contain at least 3 character(s)",
		"Field2": "is required",
	}

	for field, expected := range expectedErrors {
		actual, exists := vErr.Fields()[field]
		if !exists {
			t.Errorf("Missing expected error for field %q", field)
			continue
		}
		if !strings.Contains(actual, expected) {
			t.Errorf("Field %q error mismatch: got %q, want %q", field, actual, expected)
		}
	}
}
