package vaultify

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultProviderLoad(t *testing.T) {
	defaults := map[string]any{
		"server": map[string]any{
			"host": "localhost",
			"port": 8080,
		},
		"debug": false,
	}

	provider := NewDefaultProvider(defaults)
	data, err := provider.Load(context.Background())

	require.NoError(t, err)
	assert.Equal(t, defaults, data)
}

func TestDefaultProviderPriority(t *testing.T) {
	provider := NewDefaultProvider(map[string]any{})
	assert.Equal(t, 10, provider.Priority())

	custom := NewDefaultProvider(map[string]any{}, WithDefaultPriority(5))
	assert.Equal(t, 5, custom.Priority())
}

func TestDefaultProviderName(t *testing.T) {
	provider := NewDefaultProvider(map[string]any{})
	assert.Equal(t, "default", provider.Name())
}

func TestDefaultProviderStop(t *testing.T) {
	provider := NewDefaultProvider(map[string]any{})
	err := provider.Stop()
	assert.NoError(t, err)
}

type testConfig struct {
	Port    int    `yaml:"port" default:"8080"`
	Host    string `yaml:"host" default:"localhost"`
	Debug   bool   `yaml:"debug" default:"false"`
	Timeout string `yaml:"timeout" default:"30s"`
}

func TestNewDefaultProviderFromStruct(t *testing.T) {
	cfg := testConfig{}
	provider := NewDefaultProviderFromStruct(cfg)
	data, err := provider.Load(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 8080, data["port"])
	assert.Equal(t, "localhost", data["host"])
	assert.Equal(t, false, data["debug"])
	assert.Equal(t, "30s", data["timeout"])
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Port", "Port"},
		{"ServerPort", "Server_Port"},
		{"HTTPServer", "H_T_T_P_Server"},
		{"host", "host"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, toSnakeCase(tt.input))
		})
	}
}

func TestExtractDefaultsEmpty(t *testing.T) {
	result := extractDefaults("not a struct")
	assert.Empty(t, result)
}
