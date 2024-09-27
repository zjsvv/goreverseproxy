package main

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/zjsvv/goreverseproxy/config"
)

func TestServeHTTP_BlockRequest(t *testing.T) {
	// setup
	targetURL := "http://example.com"
	revProxy, _ := NewRevProxy(context.Background(), targetURL)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	// mock config
	mockConfig := &config.RevProxyConfig{
		BlockedHeadersMap: map[string]struct{}{"Blocked-Header": {}},
	}
	getConfig = func() *config.RevProxyConfig {
		return mockConfig
	}

	// add blocked header to request
	req.Header.Add("Blocked-Header", "test-value")
	rr := httptest.NewRecorder()

	// act
	revProxy.ServeHTTP(rr, req)

	// assert
	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.Contains(t, rr.Body.String(), "Request blocked by proxy rules")
}

func TestServeHTTP_PassRequest(t *testing.T) {
	// setup
	targetURL := "http://example.com"
	revProxy, _ := NewRevProxy(context.Background(), targetURL)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	// mock config
	mockConfig := &config.RevProxyConfig{
		BlockedHeadersMap: map[string]struct{}{},
	}
	getConfig = func() *config.RevProxyConfig {
		return mockConfig
	}

	// act
	rr := httptest.NewRecorder()
	revProxy.ServeHTTP(rr, req)

	resp := rr.Result()
	defer resp.Body.Close()

	// assert: expect request to go through, but there is no /test in the target URL
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestShouldBlockRequest_BlockedHeader(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Add("Blocked-Header", "test-value")

	// mock config
	mockConfig := &config.RevProxyConfig{
		BlockedHeadersMap: map[string]struct{}{"Blocked-Header": {}},
	}
	getConfig = func() *config.RevProxyConfig {
		return mockConfig
	}

	// act
	blocked := shouldBlockRequest(req)

	// assert
	assert.True(t, blocked)
}

func TestShouldBlockRequest_BlockedQueryParam(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/test?blockedParam=value", nil)

	// mock config with blocked query param
	mockConfig := &config.RevProxyConfig{
		BlockedQueryParamsMap: map[string]struct{}{"blockedParam": {}},
	}
	getConfig = func() *config.RevProxyConfig {
		return mockConfig
	}

	// act
	blocked := shouldBlockRequest(req)

	// assert
	assert.True(t, blocked)
}

func TestMaskSensitiveInfo(t *testing.T) {
	// mock config
	mockConfig := &config.RevProxyConfig{
		MaskedNeededKeys: []string{"password", "creditCard"},
	}
	getConfig = func() *config.RevProxyConfig {
		return mockConfig
	}

	input := `{"password":"12345","creditCard":"1234-4567-8787"}`
	maskedData, err := maskSensitiveInfo(input)

	assert.NoError(t, err)
	assert.Contains(t, maskedData, `"password":"*****"`)
	assert.Contains(t, maskedData, `"creditCard":"**************"`)
}

func TestMaskSensitiveInfo_WithErrorMaskingJSONFiels(t *testing.T) {
	// mock config
	mockConfig := &config.RevProxyConfig{
		MaskedNeededKeys: []string{"password", "creditCard"},
	}
	getConfig = func() *config.RevProxyConfig {
		return mockConfig
	}

	input := `<html></html>`
	_, err := maskSensitiveInfo(input)
	assert.Error(t, err)
}

func TestModifyResponse(t *testing.T) {
	// mock response
	body := `{"password":"12345"}`
	resp := &http.Response{
		Body:          io.NopCloser(bytes.NewBufferString(body)),
		ContentLength: int64(len(body)),
		Header:        make(http.Header),
	}

	// mock config
	mockConfig := &config.RevProxyConfig{
		MaskedNeededKeys: []string{"password"},
	}
	getConfig = func() *config.RevProxyConfig {
		return mockConfig
	}

	// act
	err := modifyResponse(resp)

	// assert
	assert.NoError(t, err)

	// check if response body is masked
	maskedBody, _ := io.ReadAll(resp.Body)
	assert.Equal(t, string(maskedBody), `{"password":"*****"}`)

	// check if content length is updated
	assert.Equal(t, strconv.Itoa(len(maskedBody)), resp.Header.Get("Content-Length"))
}

func TestGracefulShutdown(t *testing.T) {
	// Setup the proxy and server
	mockConfig := &config.RevProxyConfig{
		TargetUrl:  "http://example.com",
		TargetPort: "8080",
	}

	getConfig = func() *config.RevProxyConfig {
		return mockConfig
	}

	revProxy, _ := NewRevProxy(context.Background(), "http://example.com:8080")
	srv := &http.Server{
		Addr:    ":8080",
		Handler: revProxy,
	}

	// run the server in a goroutine
	go srv.ListenAndServe()

	// simulate interrupt signal
	_, stop := signal.NotifyContext(context.Background(), syscall.SIGINT)
	defer stop()

	// simulate graceful shutdown
	stop()

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := srv.Shutdown(ctxShutdown)

	// assert
	assert.NoError(t, err)
}

func TestGetLogLevel(t *testing.T) {
	// define test cases
	testCases := []struct {
		input    string
		expected slog.Leveler
	}{
		{"-5", slog.LevelInfo},
		{"-4", slog.LevelDebug},
		{"0", slog.LevelInfo},
		{"4", slog.LevelWarn},
		{"8", slog.LevelError},
	}

	// run test cases
	for _, tc := range testCases {
		level, err := getLogLevel(tc.input)
		assert.NoError(t, err)
		assert.Equal(t, tc.expected, level, "getLogLevel(%s) = %v; expected %v", tc.input, level, tc.expected)
	}
}

func TestGetLogLevel_InvalidInput(t *testing.T) {
	input := "astring"
	_, err := getLogLevel(input)
	assert.Error(t, err)
}

func TestGetEnv(t *testing.T) {
	// setup env variables
	os.Setenv("LOG_LEVEL", "-4")
	os.Setenv("PORT", "8080")

	// define test cases
	testCases := []struct {
		key        string
		expected   string
		defaultVal string
	}{
		{"LOG_LEVEL", "-4", "0"},
		{"PORT", "8080", "8090"},
		{"NONEXISTED_KEY", "test", "test"},
	}

	// run test cases
	for _, tc := range testCases {
		val := getEnv(tc.key, tc.defaultVal)
		assert.Equal(t, tc.expected, val, "getEnv(%s) = %v; expected %v", tc.key, val, tc.expected)
	}
}
