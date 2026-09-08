package config

import "github.com/myszqua/vaultify/vault"

// Provider describes how vaultify connects to an external config server.
type Provider struct {
	Vault *vault.Client `yaml:"vault"`
}

// Loader configures which dependencies, environment, and service vaultify loads.
type Loader struct {
	Dependencies []string `yaml:"dependencies"`
	AppEnv       string   `yaml:"app_env" default:"local"`
	ServiceName  string   `yaml:"service_name" config:"required"`
}
