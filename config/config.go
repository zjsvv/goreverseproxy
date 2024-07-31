package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var (
	revproxConfigAbsPath = "../config.yaml"
	revProxyConfig       = &RevProxyConfig{}
)

func init() {
	revProxyConfig = LoadConfig()
}

func LoadConfig() *RevProxyConfig {
	configPath, err := filepath.Abs(revproxConfigAbsPath)
	if err != nil {
		panic(fmt.Sprintf("filepath.Abs failed. err: %+v", err))
	}

	conf := &RevProxyConfig{}

	file, err := os.ReadFile(configPath)
	if err != nil {
		panic(fmt.Sprintf("os.ReadFile failed. err: %+v", err))
	}
	err = yaml.Unmarshal(file, conf)
	if err != nil {
		panic(fmt.Sprintf("yaml.Unmarshal failed. err: %+v", err))
	}

	// update blockedHeaders mapping
	conf.BlockedHeadersMap = make(map[string]struct{})
	for _, header := range conf.BlockedHeaders {
		conf.BlockedHeadersMap[header] = struct{}{}
	}

	// update blockedQueryParams mapping
	conf.BlockedQueryParamsMap = make(map[string]struct{})
	for _, param := range conf.BlockedQueryParams {
		conf.BlockedQueryParamsMap[param] = struct{}{}
	}

	return conf
}

type RevProxyConfig struct {
	TargetUrl             string              `yaml:"targetUrl"`
	TargetPort            string              `yaml:"targetPort"`
	BlockedHeaders        []string            `yaml:"blockedHeaders"`
	BlockedHeadersMap     map[string]struct{} `yaml:"-"`
	BlockedQueryParams    []string            `yaml:"blockedQueryParams"`
	BlockedQueryParamsMap map[string]struct{} `yaml:"-"`
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
