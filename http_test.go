package httputil

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"github.com/docker/go-units"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestEncode(t *testing.T) {
	testCases := []struct {
		name        string
		input       any
		expected    string
		expectError bool
	}{
		{
			name:        "Nil Slice",
			input:       []int(nil),
			expected:    "[]\n",
			expectError: false,
		},
		{
			name:        "Empty Slice",
			input:       []int{},
			expected:    "[]\n",
			expectError: false,
		},
		{
			name:        "Nil Map",
			input:       map[string]int(nil),
			expected:    "{}\n",
			expectError: false,
		},
		{
			name:        "Empty Map",
			input:       map[string]int{},
			expected:    "{}\n",
			expectError: false,
		},
		{
			name:        "Struct",
			input:       struct{ Name string }{Name: "test"},
			expected:    "{\n\t\"Name\": \"test\"\n}\n",
			expectError: false,
		},
		{
			name:        "Channel Type",
			input:       make(chan int),
			expectError: true,
		},
		{
			name:        "Struct With Unexported Field",
			input:       struct{ name string }{name: "test"}, // lowercase field
			expected:    "{}\n",
			expectError: false,
		},
		{
			name:        "Function Type",
			input:       func() {},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()
			e := echo.New()
			r := Context(e.NewContext(req, w))

			err := r.Encode(tc.input)

			if tc.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if w.Body.String() != tc.expected {
					t.Errorf("expected %q, got %q", tc.expected, w.Body.String())
				}
				if w.Header().Get("Content-Type") != "application/json" {
					t.Errorf("expected Content-Type to be application/json, got %q", w.Header().Get("Content-Type"))
				}
			}
		})
	}
}

func TestDecode(t *testing.T) {
	testCases := []struct {
		name        string
		body        string
		input       any
		expectedErr error
	}{
		{
			name:  "Valid JSON",
			body:  `{"Name": "test"}`,
			input: &struct{ Name string }{},
		},
		{
			name:        "Invalid JSON",
			body:        `{"Name": true}`,
			input:       &struct{ Name string }{},
			expectedErr: errors.New("couldn't decode request type (*struct { Name string }): code=400, message=Unmarshal type error: expected=string, got=bool, field=Name, offset=13, internal=json: cannot unmarshal bool into Go struct field .Name of type string"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			e := echo.New()
			r := Context(e.NewContext(req, w))

			err := r.Decode(tc.input)

			if tc.expectedErr != nil {
				if err == nil || !strings.Contains(err.Error(), tc.expectedErr.Error()) {
					t.Errorf("expected error to contain %q, got %q", tc.expectedErr.Error(), err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestError(t *testing.T) {
	t.Run("standard error", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		e := echo.New()
		r := Context(e.NewContext(req, w))

		err := r.Error(errors.New("test error"), http.StatusBadRequest)

		if err == nil || err.Error() != "test error" {
			t.Errorf("expected error %q, got %q", "test error", err)
		}
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status code %d, got %d", http.StatusBadRequest, w.Code)
		}
		if !strings.Contains(w.Body.String(), "test error") {
			t.Errorf("expected body to contain %q, got %q", "test error", w.Body.String())
		}
	})

	t.Run("validation error", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		e := echo.New()
		r := Context(e.NewContext(req, w))

		verr := &ValidationError{
			FieldErrors: map[string]string{
				"field1": "error1",
				"field2": "error2",
			},
			joinedError: errors.New("validation failed"),
		}
		err := r.Error(verr, http.StatusUnprocessableEntity)

		if err == nil || err.Error() != verr.Error() {
			t.Errorf("expected error %q, got %q", verr.Error(), err)
		}
		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected status code %d, got %d", http.StatusUnprocessableEntity, w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if response["error"] != "validation failed" {
			t.Errorf("expected error key 'validation failed', got %q", response["error"])
		}

		fields := response["fields"].(map[string]interface{})
		if len(fields) != 2 || fields["field1"] != "error1" || fields["field2"] != "error2" {
			t.Errorf("unexpected fields content: %v", fields)
		}
	})
}

func TestCheck(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	e := echo.New()
	r := Context(e.NewContext(req, w))

	err := r.Check("test message", errors.New("test error"))

	if err == nil || !strings.Contains(err.Error(), "test message: test error") {
		t.Errorf("expected error %q, got %q", "test message: test error", err)
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status code %d, got %d", http.StatusInternalServerError, w.Code)
	}
	if !strings.Contains(w.Body.String(), "test message: test error") {
		t.Errorf("expected body to contain %q, got %q", "test message: test error", w.Body.String())
	}

	w = httptest.NewRecorder()
	r = Context(e.NewContext(req, w))
	err = r.Check("test message", nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDecodeForm(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		value       string
		target      any
		want        any
		wantErr     string
		shouldPanic bool
	}{
		{
			name:   "string",
			key:    "string",
			value:  "test",
			target: new(string),
			want:   "test",
		},
		{
			name:   "comma separated strings",
			key:    "strings",
			value:  "a,b,c",
			target: new([]string),
			want:   []string{"a", "b", "c"},
		},
		{
			name:   "int",
			key:    "int",
			value:  "123",
			target: new(int),
			want:   123,
		},
		{
			name:   "int64",
			key:    "int64",
			value:  "1234567890",
			target: new(int64),
			want:   int64(1234567890),
		},
		{
			name:   "uint64",
			key:    "uint64",
			value:  "9876543210",
			target: new(uint64),
			want:   uint64(9876543210),
		},
		{
			name:   "bool",
			key:    "bool",
			value:  "true",
			target: new(bool),
			want:   true,
		},
		{
			name:   "time",
			key:    "time",
			value:  "2024-01-02T15:04:05Z",
			target: new(*time.Time),
			want:   mustParseTime("2024-01-02T15:04:05Z"),
		},
		{
			name:        "unsupported type",
			key:         "unsupported",
			value:       "value",
			target:      new(chan int),
			shouldPanic: true,
		},
		{
			name:   "empty string",
			key:    "empty",
			value:  "",
			target: new(string),
			want:   "",
		},
		{
			name:    "invalid int",
			key:     "invalidInt",
			value:   "abc",
			target:  new(int),
			wantErr: "invalid form value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)
			req.PostForm = map[string][]string{tt.key: {tt.value}}
			e := echo.New()
			r := Context(e.NewContext(req, httptest.NewRecorder()))

			if tt.shouldPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("expected panic did not occur")
					}
				}()
			}

			err := r.DecodeForm(tt.key, tt.target)

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			got := dereference(tt.target)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v (%T), want %v (%T)", got, got, tt.want, tt.want)
			}
		})
	}
}

func mustParseTime(s string) *time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return &t
}

func dereference(v any) any {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		return val.Elem().Interface()
	}
	return v
}

type TextUnmarshaler string

func (t *TextUnmarshaler) UnmarshalText(text []byte) error {
	*t = TextUnmarshaler("unmarshaled: " + string(text))
	return nil
}

type StringLoader string

func (s *StringLoader) LoadString(value string) error {
	*s = StringLoader("loaded: " + value)
	return nil
}

func TestPrepareFileUpload_Multipart(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, err = part.Write([]byte("test file content"))
	if err != nil {
		t.Fatal(err)
	}
	writer.Close()

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	e := echo.New()
	r := Context(e.NewContext(req, w))

	result, err := r.PrepareFileUpload(units.MB) // 1MB limit
	if err != nil {
		t.Fatalf("PrepareFileUpload failed: %v", err)
	}
	defer result.File.Close()

	if result.Filename != "test.txt" {
		t.Errorf("Expected filename 'test.txt', got '%s'", result.Filename)
	}
	if result.Size != 17 {
		t.Errorf("Expected size 17, got %d", result.Size)
	}

	content, err := io.ReadAll(result.File)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "test file content" {
		t.Errorf("Expected content 'test file content', got '%s'", string(content))
	}
}

func TestPrepareFileUpload_RawBody(t *testing.T) {
	content := []byte("raw file content")
	req := httptest.NewRequest("POST", "/upload", bytes.NewReader(content))
	w := httptest.NewRecorder()
	e := echo.New()
	r := Context(e.NewContext(req, w))

	result, err := r.PrepareFileUpload(units.MB) // 1MB limit
	if err != nil {
		t.Fatalf("PrepareFileUpload failed: %v", err)
	}
	defer result.File.Close()

	if result.Size != uint64(len(content)) {
		t.Errorf("Expected size %d, got %d", len(content), result.Size)
	}

	readContent, err := io.ReadAll(result.File)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(readContent, content) {
		t.Errorf("Expected content %q, got %q", content, readContent)
	}
}

func TestPrepareFileUpload_SizeLimit(t *testing.T) {
	tests := []struct {
		name        string
		contentSize int
		limit       int64
		expectError bool
	}{
		{
			name:        "Raw body at limit",
			contentSize: 10 * units.MB,
			limit:       10 * units.MB,
			expectError: false,
		},
		{
			name:        "Raw body over limit",
			contentSize: 11 * units.MB,
			limit:       10 * units.MB,
			expectError: true,
		},
		{
			name:        "Multipart at limit",
			contentSize: (10 * units.MB) - 500,
			limit:       10 * units.MB,
			expectError: false,
		},
		{
			name:        "Multipart over limit",
			contentSize: 11 * units.MB,
			limit:       10 * units.MB,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.name, "Multipart") {
				// Test multipart form
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				part, err := writer.CreateFormFile("file", "test.txt")
				if err != nil {
					t.Fatal(err)
				}
				// Fill with random data instead of null bytes
				data := make([]byte, tt.contentSize)
				_, err = rand.Read(data)
				if err != nil {
					t.Fatal(err)
				}
				_, err = part.Write(data)
				if err != nil {
					t.Fatal(err)
				}
				err = writer.Close()
				if err != nil {
					require.NoError(t, err)
				}

				req := httptest.NewRequest("POST", "/upload", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				w := httptest.NewRecorder()
				e := echo.New()
				r := Context(e.NewContext(req, w))

				_, err = r.PrepareFileUpload(tt.limit)
				if tt.expectError {
					if err == nil {
						t.Error("Expected error for exceeding size limit")
					}
					if !strings.Contains(err.Error(), "exceeds maximum allowed size") {
						t.Errorf("Expected size limit error, got: %v", err)
					}
				} else if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			} else {
				// Test raw body
				// Fill with random data instead of null bytes
				data := make([]byte, tt.contentSize)
				_, err := rand.Read(data)
				if err != nil {
					t.Fatal(err)
				}
				req := httptest.NewRequest("POST", "/upload", bytes.NewReader(data))
				w := httptest.NewRecorder()
				e := echo.New()
				r := Context(e.NewContext(req, w))

				_, err = r.PrepareFileUpload(tt.limit)
				if tt.expectError {
					if err == nil {
						t.Error("Expected error for exceeding size limit")
					}
					if !strings.Contains(err.Error(), "exceeds maximum allowed size") {
						t.Errorf("Expected size limit error, got: %v", err)
					}
				} else if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestDecodeForm_CustomTypes(t *testing.T) {
	req := httptest.NewRequest("POST", "/", nil)
	req.PostForm = map[string][]string{
		"text":   {"test"},
		"string": {"value"},
	}
	e := echo.New()
	r := Context(e.NewContext(req, httptest.NewRecorder()))

	var text TextUnmarshaler
	err := r.DecodeForm("text", &text)
	if err != nil || text != "unmarshaled: test" {
		t.Errorf("text unmarshaler: expected %q, got %q, err %v", "unmarshaled: test", text, err)
	}

	var str StringLoader
	err = r.DecodeForm("string", &str)
	if err != nil || str != "loaded: value" {
		t.Errorf("string loader: expected %q, got %q, err %v", "loaded: value", str, err)
	}
}
