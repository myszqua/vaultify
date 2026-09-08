package vaultify

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveEnvVar(t *testing.T) {
	t.Setenv("MY_VAR", "hello_world")

	resolver := NewDynamicResolver()
	result, err := resolver.Resolve("prefix_${MY_VAR}_suffix")

	require.NoError(t, err)
	assert.Equal(t, "prefix_hello_world_suffix", result)
}

func TestResolveNoPlaceholders(t *testing.T) {
	resolver := NewDynamicResolver()
	result, err := resolver.Resolve("plain string")

	require.NoError(t, err)
	assert.Equal(t, "plain string", result)
}

func TestResolveFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret.txt")

	err := os.WriteFile(path, []byte("file_content"), 0o600)
	require.NoError(t, err)

	resolver := NewDynamicResolver()
	result, err := resolver.Resolve("${FILE:" + path + "}")

	require.NoError(t, err)
	assert.Equal(t, "file_content", result)
}

func TestResolveFileNotFound(t *testing.T) {
	resolver := NewDynamicResolver()
	result, err := resolver.Resolve("${FILE:/nonexistent/file.txt}")

	require.NoError(t, err)
	assert.Equal(t, "${FILE:/nonexistent/file.txt}", result)
}

func TestResolveMap(t *testing.T) {
	t.Setenv("TEST_DB_HOST", "localhost")

	settings := map[string]any{
		"database": map[string]any{
			"host": "${TEST_DB_HOST}",
			"port": 5432,
		},
	}

	resolver := NewDynamicResolver()
	result, err := ResolveAll(settings, resolver)

	require.NoError(t, err)

	db, ok := result["database"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "localhost", db["host"])
	assert.Equal(t, 5432, db["port"])
}

func TestResolveSlice(t *testing.T) {
	t.Setenv("ITEM1", "apple")

	settings := map[string]any{
		"fruits": []any{"${ITEM1}", "banana"},
	}

	resolver := NewDynamicResolver()
	result, err := ResolveAll(settings, resolver)

	require.NoError(t, err)

	fruits, ok := result["fruits"].([]any)
	require.True(t, ok)
	assert.Equal(t, "apple", fruits[0])
	assert.Equal(t, "banana", fruits[1])
}

func TestResolveNestedMap(t *testing.T) {
	t.Setenv("LEVEL2", "deep_value")

	settings := map[string]any{
		"level1": map[string]any{
			"level2": map[string]any{
				"key": "${LEVEL2}",
			},
		},
	}

	resolver := NewDynamicResolver()
	result, err := ResolveAll(settings, resolver)

	require.NoError(t, err)

	l1, ok := result["level1"].(map[string]any)
	require.True(t, ok)

	l2, ok := l1["level2"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "deep_value", l2["key"])
}
