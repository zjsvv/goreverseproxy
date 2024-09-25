package middleware

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComposeRequestHeadersNilRequest(t *testing.T) {
	result := composeRequestHeaders(nil)

	assert.Equal(t, 0, len(result), "Expected an empty map")
}

func TestComposeRequestHeadersWithHeaders(t *testing.T) {
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

func TestComposeRequestHeadersNoHeaders(t *testing.T) {
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

func TestComposeRequestHeadersWithMultipleHeaders(t *testing.T) {
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
