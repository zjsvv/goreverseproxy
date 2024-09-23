package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

var (
	revproxConfigPath = "conf/config.yaml"
	revProxyConfig    = &RevProxyConfig{}
)

type RevProxyConfig struct {
	TargetUrl             string              `yaml:"targetUrl"`
	TargetPort            string              `yaml:"targetPort"`
	BlockedHeaders        []string            `yaml:"blockedHeaders"`
	BlockedHeadersMap     map[string]struct{} `yaml:"-"`
	BlockedQueryParams    []string            `yaml:"blockedQueryParams"`
	BlockedQueryParamsMap map[string]struct{} `yaml:"-"`
	MaskedNeededKeys      []string            `yaml:"maskedNeededKeys"`
	MaskedNeededKeysMap   map[string]struct{} `yaml:"-"`
}

func (r *RevProxyConfig) loadConfig() {
	file, err := os.ReadFile(revproxConfigPath)
	if err != nil {
		panic(fmt.Sprintf("os.ReadFile failed. err: %+v", err))
	}
	err = yaml.Unmarshal(file, r)
	if err != nil {
		panic(fmt.Sprintf("yaml.Unmarshal failed. err: %+v", err))
	}

	// update blockedHeaders mapping
	r.BlockedHeadersMap = make(map[string]struct{})
	for _, header := range r.BlockedHeaders {
		r.BlockedHeadersMap[header] = struct{}{}
	}

	// update blockedQueryParams mapping
	r.BlockedQueryParamsMap = make(map[string]struct{})
	for _, param := range r.BlockedQueryParams {
		r.BlockedQueryParamsMap[param] = struct{}{}
	}

	// update maskedNeededKeys mapping
	r.MaskedNeededKeysMap = make(map[string]struct{})
	for _, key := range r.MaskedNeededKeys {
		r.MaskedNeededKeysMap[key] = struct{}{}
	}
}

func (r *RevProxyConfig) IsHeaderBlocked(header string) bool {
	_, exist := r.BlockedHeadersMap[header]
	return exist
}

func (r *RevProxyConfig) IsQueryParamBlocked(param string) bool {
	_, exist := r.BlockedQueryParamsMap[param]
	return exist
}

func GetConfig() *RevProxyConfig {
	return revProxyConfig
}

func InitConfig() {
	revProxyConfig.loadConfig()
}
