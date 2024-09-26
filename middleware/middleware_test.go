package middleware

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)


func TestLoggerMiddleware_WithMockLog(t *testing.T) {
	mockResponseBody := "this is mock response"

	// create a mock logger
	buffer := new(bytes.Buffer)
	mockLogger := slog.New(slog.NewTextHandler(buffer, nil))
	slog.SetDefault(mockLogger)

	// mock handler that returns a status 200 response
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponseBody))
	})

	// create a new logger middleware with the mockHandler
	loggerMiddleware := NewLogger(mockHandler)

	// create a new HTTP request
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("this is request body"))
	req.Header.Set("Content-Type", "application/json")

	// create a new HTTP recorder to capture the response
	recorder := httptest.NewRecorder()

	// call the ServeHTTP method on the logger middleware
	loggerMiddleware.ServeHTTP(recorder, req)

	// get the result
	result := recorder.Result()
	defer result.Body.Close()

	// read the response body
	body, _ := io.ReadAll(result.Body)

	// verify response
	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.Equal(t, mockResponseBody, string(body))

	// verify log output captured by mock logger
	logOutput := buffer.String()

	// check if expected log fields exist in the output
	assert.Contains(t, logOutput, "method=POST")
	assert.Contains(t, logOutput, "path=/test")
	assert.Contains(t, logOutput, "this is request body")
	assert.Contains(t, logOutput, "this is mock response")
	assert.Contains(t, logOutput, "status=200")
}

func TestLoggerMiddleware_WithErrorLog(t *testing.T) {
	mockResponseBody := "Something went wrong"

	// create a mock logger
	buffer := new(bytes.Buffer)
	mockLogger := slog.New(slog.NewTextHandler(buffer, nil))
	slog.SetDefault(mockLogger)

	// mock handler that returns an internal server error
	errorHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(mockResponseBody))
	})

	// create a new logger middleware with the errorHandler
	loggerMiddleware := NewLogger(errorHandler)

	// create a new HTTP request
	req := httptest.NewRequest(http.MethodGet, "/error", nil)

	// create a new HTTP recorder to capture the response
	recorder := httptest.NewRecorder()

	// call the ServeHTTP method on the logger middleware
	loggerMiddleware.ServeHTTP(recorder, req)

	// get the result
	result := recorder.Result()
	defer result.Body.Close()

	// read the response body
	body, _ := io.ReadAll(result.Body)

	// verify response
	assert.Equal(t, http.StatusInternalServerError, result.StatusCode)
	assert.Equal(t, mockResponseBody, string(body))

	// verify log output captured by mock logger
	logOutput := buffer.String()

	// check if expected log fields exist in the output
	assert.Contains(t, logOutput, "method=GET")
	assert.Contains(t, logOutput, "path=/error")
	assert.Contains(t, logOutput, "status=500")
	assert.Contains(t, logOutput, "Something went wrong")
}

// Mock reader that always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (int, error) {
	return 0, errors.New("mock read error")
}

func TestRecordRequest_ReadAllError(t *testing.T) {
	// create a mock logger
	buffer := new(bytes.Buffer)
	mockLogger := slog.New(slog.NewTextHandler(buffer, nil))
	slog.SetDefault(mockLogger)

	// Create a mock request with the errorReader as the body
	req := &http.Request{
		Method: "POST",
		URL:    nil, // you can fill in a valid URL if needed
		Body:   io.NopCloser(&errorReader{}),
	}

	recordRequest(req)

	// verify log output captured by mock logger
	logOutput := buffer.String()

	assert.Contains(t, logOutput, "Error reading from request body")
}

func TestRecordRequest_JSONMarshalError(t *testing.T) {
	// create a mock logger
	buffer := new(bytes.Buffer)
	mockLogger := slog.New(slog.NewTextHandler(buffer, nil))
	slog.SetDefault(mockLogger)

	// Create a mock request with a body
	body := io.NopCloser(bytes.NewBufferString(`{"test":"data"}`))

	req := &http.Request{
		Method: "POST",
		URL: 	nil,
		Body:   body,
		Header: make(http.Header),
	}

	jsonMarshal = func(v any) ([]byte, error) {
		return nil, errors.New("Marshalling failed")
	}

	recordRequest(req)

	// verify log output captured by mock logger
	logOutput := buffer.String()

	assert.Contains(t, logOutput, "jsonMarshal header failed")
}

func TestComposeRequestHeaders_NilRequest(t *testing.T) {
	result := composeRequestHeaders(nil)

	assert.Equal(t, 0, len(result), "Expected an empty map")
}

func TestComposeRequestHeaders_WithHeaders(t *testing.T) {
	req := &http.Request{
		Header: http.Header{
			"X-Custom-Header": []string{"CustomValue"},
		},
		ContentLength: 123,
		Host:          "example.com",
	}

	headers := composeRequestHeaders(req)

	expectedHeaders := map[string][]string{
		"X-Custom-Header": {"CustomValue"},
		"Content-Length":  {"123"},
		"Host":            {"example.com"},
	}

	assert.Equal(t, expectedHeaders, headers, "Expected headers to be correctly copied and modified")
}

func TestComposeRequestHeaders_NoHeaders(t *testing.T) {
	req := &http.Request{
		Header:        http.Header{},
		ContentLength: 0,
		Host:          "localhost",
	}

	headers := composeRequestHeaders(req)

	expectedHeaders := map[string][]string{
		"Content-Length": {"0"},
		"Host":           {"localhost"},
	}

	assert.Equal(t, expectedHeaders, headers, "Expected headers to include Content-Length and Host")
}

func TestComposeRequestHeaders_WithMultipleHeaders(t *testing.T) {
	req := &http.Request{
		Header: http.Header{
			"X-First-Header":  []string{"Value1"},
			"X-Second-Header": []string{"Value2", "Value3"},
		},
		ContentLength: 456,
		Host:          "example.com",
	}

	headers := composeRequestHeaders(req)

	expectedHeaders := map[string][]string{
		"X-First-Header":  {"Value1"},
		"X-Second-Header": {"Value2", "Value3"},
		"Content-Length":  {"456"},
		"Host":            {"example.com"},
	}

	assert.Equal(t, expectedHeaders, headers, "Expected headers to be correctly copied with all values")
}
