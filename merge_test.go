package vaultify

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeepMergeSimple(t *testing.T) {
	base := map[string]any{"a": 1, "b": 2}
	override := map[string]any{"b": 3, "c": 4}
	opts := MergeOptions{SliceStrategy: MergeStrategyReplace}

	result := deepMerge(base, override, opts, newNopLogger())

	assert.Equal(t, 1, result["a"])
	assert.Equal(t, 3, result["b"])
	assert.Equal(t, 4, result["c"])
}

func TestDeepMergeNested(t *testing.T) {
	base := map[string]any{
		"server": map[string]any{
			"host": "localhost",
			"port": 8080,
		},
	}
	override := map[string]any{
		"server": map[string]any{
			"port": 9090,
		},
	}
	opts := MergeOptions{SliceStrategy: MergeStrategyReplace}

	result := deepMerge(base, override, opts, newNopLogger())

	server, ok := result["server"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "localhost", server["host"])
	assert.Equal(t, 9090, server["port"])
}

func TestDeepMergeSliceReplace(t *testing.T) {
	base := map[string]any{
		"items": []any{"a", "b"},
	}
	override := map[string]any{
		"items": []any{"c"},
	}
	opts := MergeOptions{SliceStrategy: MergeStrategyReplace}

	result := deepMerge(base, override, opts, newNopLogger())

	items, ok := result["items"].([]any)
	require.True(t, ok)
	assert.Equal(t, []any{"c"}, items)
}

func TestDeepMergeSliceMerge(t *testing.T) {
	base := map[string]any{
		"items": []any{"a", "b"},
	}
	override := map[string]any{
		"items": []any{"b", "c"},
	}
	opts := MergeOptions{SliceStrategy: MergeStrategyMerge}

	result := deepMerge(base, override, opts, newNopLogger())

	items, ok := result["items"].([]any)
	require.True(t, ok)
	assert.Len(t, items, 3)
}

func TestDeepMergeTypeMismatch(t *testing.T) {
	base := map[string]any{
		"server": map[string]any{"host": "localhost"},
	}
	override := map[string]any{
		"server": "string_value",
	}
	opts := MergeOptions{SliceStrategy: MergeStrategyReplace}

	result := deepMerge(base, override, opts, newNopLogger())

	assert.Equal(t, "string_value", result["server"])
}

func TestApplyPriorities(t *testing.T) {
	data := []providerData{
		{name: "env", priority: 40, data: map[string]any{"key": "env_value"}},
		{name: "file", priority: 20, data: map[string]any{"key": "file_value", "file_only": true}},
		{name: "default", priority: 10, data: map[string]any{"key": "default_value", "default_only": true}},
	}
	opts := MergeOptions{SliceStrategy: MergeStrategyReplace}

	result := applyPriorities(data, opts, newNopLogger())

	assert.Equal(t, "env_value", result["key"])
	assert.Equal(t, true, result["file_only"])
	assert.Equal(t, true, result["default_only"])
}

func TestSortByPriority(t *testing.T) {
	data := []providerData{
		{name: "env", priority: 40},
		{name: "default", priority: 10},
		{name: "file", priority: 20},
	}

	sorted := sortByPriority(data)

	assert.Equal(t, 10, sorted[0].priority)
	assert.Equal(t, 20, sorted[1].priority)
	assert.Equal(t, 40, sorted[2].priority)
}

func TestTypesMatch(t *testing.T) {
	assert.True(t, typesMatch(1, 2))
	assert.True(t, typesMatch("a", "b"))
	assert.False(t, typesMatch(1, "a"))
	assert.True(t, typesMatch(nil, nil))
	assert.False(t, typesMatch(nil, 1))
}

func TestContainsAny(t *testing.T) {
	slice := []any{"a", "b", "c"}
	assert.True(t, containsAny(slice, "a"))
	assert.False(t, containsAny(slice, "d"))
}

func TestMergeSlicesMergeStrategy(t *testing.T) {
	base := []any{"a", "b"}
	override := []any{"b", "c"}

	result := mergeSlices(base, override, MergeStrategyMerge)
	items, ok := result.([]any)
	require.True(t, ok)
	assert.Len(t, items, 3)
}

func TestMergeSlicesReplaceStrategy(t *testing.T) {
	base := []any{"a", "b"}
	override := []any{"c"}

	result := mergeSlices(base, override, MergeStrategyReplace)
	items, ok := result.([]any)
	require.True(t, ok)
	assert.Equal(t, []any{"c"}, items)
}

func TestMergeStrategyString(t *testing.T) {
	assert.Equal(t, "replace", MergeStrategyReplace.String())
	assert.Equal(t, "merge", MergeStrategyMerge.String())
	assert.Equal(t, "unknown", MergeStrategy(99).String())
}

func TestEventTypeString(t *testing.T) {
	assert.Equal(t, "changed", EventChanged.String())
	assert.Equal(t, "reloaded", EventReloaded.String())
	assert.Equal(t, "error", EventError.String())
	assert.Equal(t, "unknown", EventType(99).String())
}

func TestFileFormatString(t *testing.T) {
	assert.Equal(t, "yaml", FormatYAML.String())
	assert.Equal(t, "json", FormatJSON.String())
	assert.Equal(t, "toml", FormatTOML.String())
	assert.Equal(t, "unknown", FileFormat(99).String())
}
