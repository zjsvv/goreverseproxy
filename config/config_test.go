package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Setup a temporary directory for testing
	tmpDir := t.TempDir()

	// Create a temporary config.yaml file
	configContent := `
targetUrl: "http://localhost"
targetPort: "8080"

blockedHeaders:
  - "X-Custom-Key"
  - "AccessToken"

blockedQueryParams:
  - "fliter"
  - "apple"
`
	configFilePath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configFilePath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write temp config file: %v", err)
	}

	// override the config file path
	revproxConfigAbsPath = configFilePath

	// load the config
	config := LoadConfig()

	// check if the config is loaded correctly
	if config.TargetUrl != "http://localhost" {
		t.Errorf("Expected TargetUrl to be 'http://localhost', got %s", config.TargetUrl)
	}
	if config.TargetPort != "8080" {
		t.Errorf("Expected TargetPort to be '8080', got %s", config.TargetPort)
	}

	expectedBlockedHeaders := []string{"X-Custom-Key", "AccessToken"}
	if len(config.BlockedHeaders) != len(expectedBlockedHeaders) {
		t.Fatalf("Expected %d blocked headers, got %d", len(expectedBlockedHeaders), len(config.BlockedHeaders))
	}
	for i, header := range expectedBlockedHeaders {
		if config.BlockedHeaders[i] != header {
			t.Errorf("Expected BlockedHeaders[%d] to be %s, got %s", i, header, config.BlockedHeaders[i])
		}
		if _, exists := config.BlockedHeadersMap[header]; !exists {
			t.Errorf("Expected header %s to be in BlockedHeadersMap", header)
		}
	}

	expectedBlockedQueryParams := []string{"fliter", "apple"}
	if len(config.BlockedQueryParams) != len(expectedBlockedQueryParams) {
		t.Fatalf("Expected %d blocked query params, got %d", len(expectedBlockedQueryParams), len(config.BlockedQueryParams))
	}
	for i, param := range expectedBlockedQueryParams {
		if config.BlockedQueryParams[i] != param {
			t.Errorf("Expected BlockedQueryParams[%d] to be %s, got %s", i, param, config.BlockedQueryParams[i])
		}
		if _, exists := config.BlockedQueryParamsMap[param]; !exists {
			t.Errorf("Expected query param %s to be in BlockedQueryParamsMap", param)
		}
	}
}

func TestIsHeaderBlocked(t *testing.T) {
	// create a RevProxyConfig instance with some blocked headers
	config := &RevProxyConfig{
		BlockedHeadersMap: map[string]struct{}{
			"X-Custom-Key": {},
			"AccessToken":  {},
		},
	}

	// define test cases
	testCases := []struct {
		header   string
		expected bool
	}{
		{"X-Custom-Key", true},
		{"AccessToken", true},
		{"Authorization", false},
		{"Content-Type", false},
	}

	// run test cases
	for _, tc := range testCases {
		result := config.IsHeaderBlocked(tc.header)
		if result != tc.expected {
			t.Errorf("IsHeaderBlocked(%s) = %v; expected %v", tc.header, result, tc.expected)
		}
	}
}

func TestIsQueryParamBlocked(t *testing.T) {
	// create a RevProxyConfig instance with some blocked query params
	config := &RevProxyConfig{
		BlockedQueryParamsMap: map[string]struct{}{
			"fliter": {},
			"fruit":  {},
		},
	}

	// define test cases
	testCases := []struct {
		header   string
		expected bool
	}{
		{"fliter", true},
		{"fruit", true},
		{"offset", false},
		{"limit", false},
	}

	// run test cases
	for _, tc := range testCases {
		result := config.IsQueryParamBlocked(tc.header)
		if result != tc.expected {
			t.Errorf("IsQueryParamsBlocked(%s) = %v; expected %v", tc.header, result, tc.expected)
		}
	}
}
