package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Create a temporary config file for testing
func createTestConfigFile(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "config*.yaml")
	assert.NoError(t, err, "Failed to create temp file")
	defer tmpFile.Close()

	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err, "Failed to write to temp file")

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

	// assert that the config is loaded correctly
	assert.Equal(t, "http://localhost", config.TargetUrl)
	assert.Equal(t, "9000", config.TargetPort)

	expectedBlockedHeaders := []string{"X-Custom-Key", "AccessToken"}
	assert.Len(t, config.BlockedHeaders, len(expectedBlockedHeaders))
	assert.Equal(t, expectedBlockedHeaders, config.BlockedHeaders)
	for _, header := range expectedBlockedHeaders {
		_, exists := config.BlockedHeadersMap[header]
		assert.True(t, exists, "Expected header %s to be in BlockedHeadersMap", header)
	}

	expectedBlockedQueryParams := []string{"filter", "offset"}
	assert.Len(t, config.BlockedQueryParams, len(expectedBlockedQueryParams))
	assert.Equal(t, expectedBlockedQueryParams, config.BlockedQueryParams)
	for _, param := range expectedBlockedQueryParams {
		_, exists := config.BlockedQueryParamsMap[param]
		assert.True(t, exists, "Expected query param %s to be in BlockedQueryParamsMap", param)
	}

	expectedMaskedNeededKeys := []string{"address", "creditcard"}
	assert.Len(t, config.MaskedNeededKeys, len(expectedMaskedNeededKeys))
	assert.Equal(t, expectedMaskedNeededKeys, config.MaskedNeededKeys)
	for _, key := range expectedMaskedNeededKeys {
		_, exists := config.MaskedNeededKeysMap[key]
		assert.True(t, exists, "Expected key %s to be in MaskedNeededKeysMap", key)
	}
}

func TestLoadConfig_PanicOnFileReadError(t *testing.T) {
	// set an invalid config path to induce a file read error
	revproxConfigPath = "invalid/path/to/config.yaml"

	// recover from panic
	defer func() {
		r := recover()
		assert.NotNil(t, r, "Expected panic but did not get one")
		assert.Equal(t, "os.ReadFile failed. err: open invalid/path/to/config.yaml: no such file or directory", r, "Unexpected panic message")
	}()

	config := &RevProxyConfig{}
	config.loadConfig()
}

func TestLoadConfig_PanicOnYamlUnmarshalError(t *testing.T) {
	testConfigContent := `invalid_yaml:   true:`
	configFilePath := createTestConfigFile(t, testConfigContent)
	defer os.Remove(configFilePath)

	// set the path to the temp file
	revproxConfigPath = configFilePath

	// recover from panic
	defer func() {
		r := recover()
		assert.NotNil(t, r, "Expected panic but did not get one")
		assert.Equal(t, "yaml.Unmarshal failed. err: yaml: mapping values are not allowed in this context", r, "Unexpected panic message")
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
		assert.Equal(t, tc.expected, result, "IsHeaderBlocked(%s) = %v; expected %v", tc.header, result, tc.expected)
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
		param    string
		expected bool
	}{
		{"filter", true},
		{"fruit", true},
		{"offset", false},
		{"limit", false},
	}

	// run test cases
	for _, tc := range testCases {
		result := config.IsQueryParamBlocked(tc.param)
		assert.Equal(t, tc.expected, result, "IsQueryParamsBlocked(%s) = %v; expected %v", tc.param, result, tc.expected)
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
			"address":    {},
			"creditcard": {},
		},
	}

	got := GetConfig()

	assert.Equal(t, want, got, "Config loaded incorrectly. Got %+v, expected %+v", got, want)
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
		MaskedNeededKeys:   []string{"address", "creditcard"},
		BlockedHeadersMap: map[string]struct{}{
			"X-Custom-Key": {},
			"AccessToken":  {},
		},
		BlockedQueryParamsMap: map[string]struct{}{
			"filter":   {},
			"category": {},
		},
		MaskedNeededKeysMap: map[string]struct{}{
			"address":    {},
			"creditcard": {},
		},
	}

	assert.Equal(t, revProxyConfig, want, "Config loaded incorrectly. Got %+v, expected %+v", revProxyConfig, want)
}
