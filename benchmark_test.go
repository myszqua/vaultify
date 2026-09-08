package vaultify

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkDeepMerge(b *testing.B) {
	base := map[string]any{
		"server": map[string]any{
			"host": "localhost",
			"port": 8080,
			"nested": map[string]any{
				"a": "value",
				"b": 42,
			},
		},
		"database": map[string]any{
			"host": "db.example.com",
			"ttl":  "30s",
		},
		"items": []any{"a", "b", "c"},
	}
	override := map[string]any{
		"server": map[string]any{
			"port": 9090,
			"nested": map[string]any{
				"c": "new",
			},
		},
		"database": map[string]any{
			"host": "override.db.example.com",
		},
		"items": []any{"d", "e"},
	}
	opts := MergeOptions{SliceStrategy: MergeStrategyReplace}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		deepMerge(base, override, opts, newNopLogger())
	}
}

func BenchmarkEnvProviderLoad(b *testing.B) {
	t := os.Getenv("BENCH")
	_ = t

	provider := NewEnvProvider("APP_")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = provider.Load(context.Background())
	}
}

func BenchmarkFileProviderLoad(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := `
server:
  host: localhost
  port: 8080
  nested:
    a: value
    b: 42
database:
  host: db.example.com
  ttl: 30s
logger:
  level: debug
  format: json
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		b.Fatal(err)
	}

	provider := NewFileProvider([]string{path})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = provider.Load(context.Background())
	}
}

func BenchmarkValidate(b *testing.B) {
	cfg := testValidConfig{
		Name: "test",
		Port: 8080,
		Host: "localhost",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Validate(cfg)
	}
}

func BenchmarkMaskSensitive(b *testing.B) {
	data := map[string]any{
		"host":     "localhost",
		"password": "super_secret_password_123",
		"token":    "abc123xyz789",
		"db": map[string]any{
			"credential": "my_credential",
			"name":       "mydb",
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		MaskSensitive(data)
	}
}
