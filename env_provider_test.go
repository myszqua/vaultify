package vaultify

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvProviderLoad(t *testing.T) {
	t.Setenv("APP_DATABASE_HOST", "db.example.com")
	t.Setenv("APP_DATABASE_PORT", "5432")
	t.Setenv("APP_DEBUG", "true")

	provider := NewEnvProvider("APP_")
	data, err := provider.Load(context.Background())

	require.NoError(t, err)

	db, ok := data["database"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "db.example.com", db["host"])
	assert.Equal(t, 5432, db["port"])
	assert.Equal(t, true, data["debug"])
}

func TestEnvProviderPrefixFiltering(t *testing.T) {
	t.Setenv("MYAPP_HOST", "localhost")
	t.Setenv("OTHER_HOST", "other.com")

	provider := NewEnvProvider("MYAPP_")
	data, err := provider.Load(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "localhost", data["host"])
	assert.Nil(t, data["other"])
}

func TestEnvProviderTypecasting(t *testing.T) {
	t.Setenv("TEST_INT", "42")
	t.Setenv("TEST_FLOAT", "3.14")
	t.Setenv("TEST_BOOLT", "true")
	t.Setenv("TEST_BOOLF", "false")
	t.Setenv("TEST_STRING", "hello")

	provider := NewEnvProvider("TEST_")
	data, err := provider.Load(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 42, data["int"])
	assert.Equal(t, true, data["boolt"])
	assert.Equal(t, false, data["boolf"])
	assert.Equal(t, "hello", data["string"])
}

func TestEnvProviderNestedStructures(t *testing.T) {
	t.Setenv("APP_SERVER_DATABASE_HOST", "localhost")
	t.Setenv("APP_SERVER_DATABASE_PORT", "5432")

	provider := NewEnvProvider("APP_")
	data, err := provider.Load(context.Background())

	require.NoError(t, err)

	server, ok := data["server"].(map[string]any)
	require.True(t, ok)

	db, ok := server["database"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "localhost", db["host"])
	assert.Equal(t, 5432, db["port"])
}

func TestEnvProviderPriority(t *testing.T) {
	provider := NewEnvProvider("APP_")
	assert.Equal(t, 40, provider.Priority())

	custom := NewEnvProvider("APP_", WithEnvPriority(60))
	assert.Equal(t, 60, custom.Priority())
}

func TestEnvProviderName(t *testing.T) {
	provider := NewEnvProvider("APP_")
	assert.Equal(t, "env", provider.Name())
}

func TestEnvProviderGetenv(t *testing.T) {
	t.Setenv("APP_TEST_KEY", "test_value")

	provider := NewEnvProvider("APP_")
	_, _ = provider.Load(context.Background())

	val, ok := provider.Getenv("APP_TEST_KEY")
	assert.True(t, ok)
	assert.Equal(t, "test_value", val)
}

func TestEnvProviderStop(t *testing.T) {
	provider := NewEnvProvider("APP_")
	err := provider.Stop()
	assert.NoError(t, err)
}

func TestCastEnvValue(t *testing.T) {
	tests := []struct {
		input    string
		expected any
	}{
		{"", ""},
		{"true", true},
		{"false", false},
		{"42", 42},
		{"3.14", 3.14},
		{"hello", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := castEnvValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnvToMap(t *testing.T) {
	result := envToMap("server_host", "localhost", '_')
	server, ok := result["server"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "localhost", server["host"])
}

func TestEnvProviderEmptyPrefix(t *testing.T) {
	os.Clearenv()
	t.Setenv("SOME_VAR", "value")

	provider := NewEnvProvider("")
	data, err := provider.Load(context.Background())

	require.NoError(t, err)
	assert.NotNil(t, data)
}
