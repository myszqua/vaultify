package vaultify

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileProviderLoadYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	err := os.WriteFile(path, []byte(`
server:
  host: localhost
  port: 8080
database:
  host: db.example.com
  port: 5432
`), 0o600)
	require.NoError(t, err)

	provider := NewFileProvider([]string{path})
	data, err := provider.Load(context.Background())

	require.NoError(t, err)

	server, ok := data["server"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "localhost", server["host"])
	assert.Equal(t, 8080, server["port"])

	database, ok := data["database"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "db.example.com", database["host"])
}

func TestFileProviderLoadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	err := os.WriteFile(path, []byte(`{
		"app": {
			"name": "test",
			"version": 1
		}
	}`), 0o600)
	require.NoError(t, err)

	provider := NewFileProvider([]string{path})
	data, err := provider.Load(context.Background())

	require.NoError(t, err)

	app, ok := data["app"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "test", app["name"])
	assert.Equal(t, float64(1), app["version"])
}

func TestFileProviderMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.yaml")
	override := filepath.Join(dir, "override.yaml")

	err := os.WriteFile(base, []byte(`
server:
  host: localhost
  port: 8080
debug: false
`), 0o600)
	require.NoError(t, err)

	err = os.WriteFile(override, []byte(`
server:
  port: 9090
`), 0o600)
	require.NoError(t, err)

	provider := NewFileProvider([]string{base, override})
	data, err := provider.Load(context.Background())

	require.NoError(t, err)

	server, ok := data["server"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "localhost", server["host"])
	assert.Equal(t, 9090, server["port"])
	assert.Equal(t, false, data["debug"])
}

func TestFileProviderMissingFile(t *testing.T) {
	provider := NewFileProvider([]string{"/nonexistent/config.yaml"})
	data, err := provider.Load(context.Background())

	require.NoError(t, err)
	assert.Empty(t, data)
}

func TestFileProviderUnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.xyz")

	err := os.WriteFile(path, []byte(`key = value`), 0o600)
	require.NoError(t, err)

	provider := NewFileProvider([]string{path})
	_, err = provider.Load(context.Background())

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedFormat)
}

func TestFileProviderPriority(t *testing.T) {
	provider := NewFileProvider([]string{})
	assert.Equal(t, 20, provider.Priority())

	custom := NewFileProvider([]string{}, WithFilePriority(50))
	assert.Equal(t, 50, custom.Priority())
}

func TestFileProviderName(t *testing.T) {
	provider := NewFileProvider([]string{})
	assert.Equal(t, "file", provider.Name())
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		path   string
		format FileFormat
		ok     bool
	}{
		{"config.yaml", FormatYAML, true},
		{"config.yml", FormatYAML, true},
		{"config.json", FormatJSON, true},
		{"config.toml", FormatTOML, true},
		{"config.xyz", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			format, ok := detectFormat(tt.path)
			assert.Equal(t, tt.format, format)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

func TestFileProviderStop(t *testing.T) {
	provider := NewFileProvider([]string{})
	err := provider.Stop()
	assert.NoError(t, err)

	err = provider.Stop()
	assert.NoError(t, err)
}
