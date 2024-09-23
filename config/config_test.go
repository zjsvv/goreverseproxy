package config

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Create a temporary config file for testing
func createTestConfigFile(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "config*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	_, err = tmpFile.WriteString(content)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	return tmpFile.Name()
}

func TestLoadConfig(t *testing.T) {
	testConfigContent := `
targetUrl: "http://localhost"
targetPort: "9000"

blockedHeaders:
  - "X-Custom-Key"
  - "AccessToken"

blockedQueryParams:
  - "filter"
  - "offset"

maskedNeededKeys:
  - "address"
  - "creditcard"
`
	configFilePath := createTestConfigFile(t, testConfigContent)
	defer os.Remove(configFilePath)

	// set the path to the temp file
	revproxConfigPath = configFilePath

	config := &RevProxyConfig{}
	config.loadConfig()

	// check if the config is loaded correctly
	if config.TargetUrl != "http://localhost" {
		t.Errorf("Expected TargetUrl to be 'http://localhost', got %s", config.TargetUrl)
	}
	if config.TargetPort != "9000" {
		t.Errorf("Expected TargetPort to be '9000', got %s", config.TargetPort)
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

	expectedBlockedQueryParams := []string{"filter", "offset"}
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

	expectedMaskedNeededKeys := []string{"address", "creditcard"}
	if len(config.MaskedNeededKeys) != len(expectedMaskedNeededKeys) {
		t.Fatalf("Expected %d blocked query params, got %d", len(expectedMaskedNeededKeys), len(config.MaskedNeededKeys))
	}
	for i, key := range expectedMaskedNeededKeys {
		if config.MaskedNeededKeys[i] != key {
			t.Errorf("Expected MaskedNeededKeys[%d] to be %s, got %s", i, key, config.MaskedNeededKeys[i])
		}
		if _, exists := config.MaskedNeededKeysMap[key]; !exists {
			t.Errorf("Expected query param %s to be in MaskedNeededKeysMap", key)
		}
	}
}

func TestLoadConfigPanicOnFileReadError(t *testing.T) {
	// set an invalid config path to induce a file read error
	revproxConfigPath = "invalid/path/to/config.yaml"

	// recover from panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic but did not get one")
		} else if r != "os.ReadFile failed. err: open invalid/path/to/config.yaml: no such file or directory" {
			t.Errorf("Unexpected panic message: %v", r)
		}
	}()

	config := &RevProxyConfig{}
	config.loadConfig()
}

func TestLoadConfigPanicOnYamlUnmarshalError(t *testing.T) {
	testConfigContent := `invalid_yaml:   true:`
	configFilePath := createTestConfigFile(t, testConfigContent)
	defer os.Remove(configFilePath)

	// set the path to the temp file
	revproxConfigPath = configFilePath

	// recover from panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic but did not get one")
		} else if r != "yaml.Unmarshal failed. err: yaml: mapping values are not allowed in this context" {
			t.Errorf("Unexpected panic message: %v", r)
		}
	}()

	config := &RevProxyConfig{}
	config.loadConfig()
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
			"filter": {},
			"fruit":  {},
		},
	}

	// define test cases
	testCases := []struct {
		header   string
		expected bool
	}{
		{"filter", true},
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

func TestGetConfig(t *testing.T) {
	testConfigContent := `
targetUrl: "http://localhost"
targetPort: "8888"

blockedHeaders:
  - "X-Custom-Key"

blockedQueryParams:
  - "limit"
  - "offset"

maskedNeededKeys:
  - "address"
  - "creditcard"
`
	configFilePath := createTestConfigFile(t, testConfigContent)
	defer os.Remove(configFilePath)
	revproxConfigPath = configFilePath

	revProxyConfig = &RevProxyConfig{}
	revProxyConfig.loadConfig()

	want := &RevProxyConfig{
		TargetUrl:  "http://localhost",
		TargetPort: "8888",
		BlockedHeaders: []string{
			"X-Custom-Key",
		},
		BlockedHeadersMap: map[string]struct{}{
			"X-Custom-Key": {},
		},
		BlockedQueryParams: []string{
			"limit",
			"offset",
		},
		BlockedQueryParamsMap: map[string]struct{}{
			"limit":  {},
			"offset": {},
		},
		MaskedNeededKeys: []string{
			"address",
			"creditcard",
		},
		MaskedNeededKeysMap: map[string]struct{}{
			"address":  {},
			"creditcard": {},
		},
	}

	got := GetConfig()

	if !cmp.Equal(want, got) {
		t.Errorf("Config loaded incorrectly. Got %+v, expected %+v", got, want)
	}
}

func TestInitConfig(t *testing.T) {
	testConfigContent := `
targetUrl: http://localhost
targetPort: "9000"
blockedHeaders:
  - "X-Custom-Key"
  - "AccessToken"
blockedQueryParams:
  - "filter"
  - "category"
maskedNeededKeys:
  - "address"
  - "creditcard"
`
	configFilePath := createTestConfigFile(t, testConfigContent)
	defer os.Remove(configFilePath)

	// set the path to the temp file
	revproxConfigPath = configFilePath

	revProxyConfig = &RevProxyConfig{}
	InitConfig()

	want := &RevProxyConfig{
		TargetUrl:          "http://localhost",
		TargetPort:         "9000",
		BlockedHeaders:     []string{"X-Custom-Key", "AccessToken"},
		BlockedQueryParams: []string{"filter", "category"},
		MaskedNeededKeys: []string{"address", "creditcard"},
		BlockedHeadersMap: map[string]struct{}{
			"X-Custom-Key": {},
			"AccessToken":  {},
		},
		BlockedQueryParamsMap: map[string]struct{}{
			"filter":   {},
			"category": {},
		},
		MaskedNeededKeysMap: map[string]struct{}{
			"address":   {},
			"creditcard": {},
		},
	}

	if !cmp.Equal(revProxyConfig, want) {
		t.Errorf("Config loaded incorrectly. Got %+v, expected %+v", revProxyConfig, want)
	}
}
