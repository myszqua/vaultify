package vaultify

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testValidConfig struct {
	Name    string `config:"required" yaml:"name"`
	Port    int    `yaml:"port" min:"1" max:"65535"`
	Host    string `yaml:"host" pattern:"^[a-z]+$"`
	Timeout int    `yaml:"timeout" min:"0"`
}

func TestValidateRequired(t *testing.T) {
	cfg := testValidConfig{
		Name: "",
		Port: 8080,
	}

	err := Validate(cfg)
	require.Error(t, err)

	var validationErr *ValidationError

	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "Name", validationErr.Field)
	assert.Equal(t, "required", validationErr.Rule)
}

func TestValidateMin(t *testing.T) {
	cfg := testValidConfig{
		Name: "test",
		Port: 0,
	}

	err := Validate(cfg)
	require.Error(t, err)

	var validationErr *ValidationError

	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "Port", validationErr.Field)
	assert.Equal(t, "min", validationErr.Rule)
}

func TestValidateMax(t *testing.T) {
	cfg := testValidConfig{
		Name: "test",
		Port: 70000,
	}

	err := Validate(cfg)
	require.Error(t, err)

	var validationErr *ValidationError

	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "Port", validationErr.Field)
	assert.Equal(t, "max", validationErr.Rule)
}

func TestValidatePattern(t *testing.T) {
	cfg := testValidConfig{
		Name: "test",
		Port: 8080,
		Host: "INVALID",
	}

	err := Validate(cfg)
	require.Error(t, err)

	var validationErr *ValidationError

	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "Host", validationErr.Field)
	assert.Equal(t, "pattern", validationErr.Rule)
}

func TestValidatePass(t *testing.T) {
	cfg := testValidConfig{
		Name: "test",
		Port: 8080,
		Host: "localhost",
	}

	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidatePtr(t *testing.T) {
	cfg := &testValidConfig{
		Name: "test",
		Port: 8080,
		Host: "localhost",
	}

	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidateNonStruct(t *testing.T) {
	err := Validate("not a struct")
	assert.NoError(t, err)
}

func TestConfigLoaderValidation(t *testing.T) {
	loader := New(nil)
	loader.settings = map[string]any{
		"server": map[string]any{
			"host": "localhost",
			"port": 8080,
		},
	}

	val := loader.Get("server.host")
	assert.Equal(t, "localhost", val)
}

func TestLoaderGetMethods(t *testing.T) {
	loader := New(nil)
	loader.settings = map[string]any{
		"string_val": "hello",
		"int_val":    42,
		"bool_val":   true,
		"nested": map[string]any{
			"key": "value",
		},
	}

	assert.Equal(t, "hello", loader.GetString("string_val"))
	assert.Equal(t, 42, loader.GetInt("int_val"))
	assert.Equal(t, true, loader.GetBool("bool_val"))
	assert.Equal(t, "value", loader.GetString("nested.key"))
	assert.Equal(t, "", loader.GetString("nonexistent"))
	assert.Equal(t, 0, loader.GetInt("nonexistent"))
	assert.False(t, loader.GetBool("nonexistent"))
}

func TestLoaderAllSettings(t *testing.T) {
	loader := New(nil)
	loader.settings = map[string]any{"key": "value"}

	all := loader.AllSettings()
	assert.Equal(t, "value", all["key"])

	all["key"] = "modified"

	assert.Equal(t, "value", loader.settings["key"])
}

func TestLoaderUnmarshal(t *testing.T) {
	type cfg struct {
		Server struct {
			Host string `json:"host"`
			Port int    `json:"port"`
		} `json:"server"`
	}

	loader := New(nil)
	loader.settings = map[string]any{
		"server": map[string]any{
			"host": "localhost",
			"port": 8080,
		},
	}

	var c cfg
	err := loader.Unmarshal(&c)
	require.NoError(t, err)
	assert.Equal(t, "localhost", c.Server.Host)
	assert.Equal(t, 8080, c.Server.Port)
}

func TestLoaderUnmarshalKey(t *testing.T) {
	type serverConfig struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}

	loader := New(nil)
	loader.settings = map[string]any{
		"server": map[string]any{
			"host": "localhost",
			"port": 8080,
		},
	}

	var s serverConfig
	err := loader.UnmarshalKey("server", &s)
	require.NoError(t, err)
	assert.Equal(t, "localhost", s.Host)
	assert.Equal(t, 8080, s.Port)
}

func TestLoaderUnmarshalKeyNotFound(t *testing.T) {
	loader := New(nil)
	loader.settings = map[string]any{}

	var s any
	err := loader.UnmarshalKey("nonexistent", &s)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestLoaderReload(t *testing.T) {
	p := NewDefaultProvider(map[string]any{"key": "initial"})
	loader := New(p)

	err := loader.Load(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "initial", loader.Get("key"))

	reloaded := false

	loader.OnReload(func() {
		reloaded = true
	})

	err = loader.Reload(context.Background())
	require.NoError(t, err)
	assert.True(t, reloaded)
}

func TestSortProvidersByPriority(t *testing.T) {
	providers := []Provider{
		NewEnvProvider("APP_"),
		NewDefaultProvider(nil),
		NewFileProvider(nil),
	}

	sorted := sortProvidersByPriority(providers)

	assert.Equal(t, "default", sorted[0].Name())
	assert.Equal(t, "file", sorted[1].Name())
	assert.Equal(t, "env", sorted[2].Name())
}

func TestLoaderLoadFromProviders(t *testing.T) {
	defaults := NewDefaultProvider(map[string]any{
		"key": "default",
		"db":  "default_db",
	})
	file := NewFileProvider(nil)
	env := NewEnvProvider("NONEXISTENT_")

	loader := New(defaults, file, env)
	err := loader.Load(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "default", loader.Get("key"))
}

func TestSplitKey(t *testing.T) {
	tests := []struct {
		key      string
		expected []string
	}{
		{"server.host", []string{"server", "host"}},
		{"server.port", []string{"server", "port"}},
		{"server_port", []string{"server_port"}},
		{"a.b.c", []string{"a", "b", "c"}},
		{"simple", []string{"simple"}},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			assert.Equal(t, tt.expected, splitKey(tt.key))
		})
	}
}

func TestGetNestedValue(t *testing.T) {
	data := map[string]any{
		"server": map[string]any{
			"host": "localhost",
		},
	}

	assert.Equal(t, "localhost", getNestedValue(data, "server.host"))
	assert.Nil(t, getNestedValue(data, "server.port"))
	assert.Nil(t, getNestedValue(data, "nonexistent"))
}

func TestErrorTypes(t *testing.T) {
	err := &ConfigError{Op: "test", Err: assert.AnError}
	assert.Contains(t, err.Error(), "test")
	assert.ErrorIs(t, err, assert.AnError)

	errWithPath := &ConfigError{Op: "test", Err: assert.AnError, Key: "mykey"}
	assert.Contains(t, errWithPath.Error(), "mykey")
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{Field: "name", Rule: "required", Message: "is required"}
	assert.Contains(t, err.Error(), "name")
	assert.Contains(t, err.Error(), "required")
	assert.ErrorIs(t, err, ErrValidationFailed)
}

func TestMergeConflictError(t *testing.T) {
	err := &MergeConflict{Key: "host", Source1: "file", Source2: "env"}
	assert.Contains(t, err.Error(), "host")
	assert.Contains(t, err.Error(), "file")
	assert.Contains(t, err.Error(), "env")
}
