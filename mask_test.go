package vaultify

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaskSensitive(t *testing.T) {
	data := map[string]any{
		"host":     "localhost",
		"password": "super_secret_123",
		"token":    "abc123xyz789",
		"port":     8080,
		"db": map[string]any{
			"credential": "my_credential",
			"name":       "mydb",
		},
	}

	masked := MaskSensitive(data)

	assert.Equal(t, "lo*****st", masked["host"])
	assert.Equal(t, 8080, masked["port"])

	db, ok := masked["db"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "mydb", db["name"])

	pass, ok := masked["password"].(string)
	require.True(t, ok)
	assert.Contains(t, pass, "***")
	assert.NotEqual(t, "super_secret_123", pass)

	tok, ok := masked["token"].(string)
	require.True(t, ok)
	assert.Contains(t, tok, "***")
	assert.NotEqual(t, "abc123xyz789", tok)

	cred, ok := db["credential"].(string)
	require.True(t, ok)
	assert.Contains(t, cred, "*")
}

func TestMaskValueShort(t *testing.T) {
	assert.Equal(t, "***", maskValue("ab"))
	assert.Equal(t, "***", maskValue("a"))
	assert.Equal(t, "***", maskValue(""))
}

func TestMaskValueLong(t *testing.T) {
	result := maskValue("password123")
	assert.Equal(t, "pa*******23", result)
}

func TestMaskValueNonString(t *testing.T) {
	assert.Equal(t, "***", maskValue(42))
}

func TestIsSensitiveKey(t *testing.T) {
	tests := []struct {
		key       string
		sensitive bool
	}{
		{"password", true},
		{"user_password", true},
		{"api_key", true},
		{"secret", true},
		{"token", true},
		{"access_key", true},
		{"private_key", true},
		{"credential", true},
		{"auth_token", true},
		{"host", true},
		{"port", false},
		{"name", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			assert.Equal(t, tt.sensitive, isSensitiveKey(tt.key))
		})
	}
}

func TestMaskSlice(t *testing.T) {
	data := map[string]any{
		"items": []any{
			map[string]any{"password": "secret"},
			"plain",
		},
	}

	masked := MaskSensitive(data)
	items, ok := masked["items"].([]any)
	require.True(t, ok)

	item, ok := items[0].(map[string]any)
	require.True(t, ok)

	pass, ok := item["password"].(string)
	require.True(t, ok)
	assert.Contains(t, pass, "*")
	assert.Equal(t, "plain", items[1])
}
