package vaultify

import (
	"strings"
)

var sensitiveKeywords = []string{
	"password", "passwd", "secret", "token",
	"api_key", "apikey", "access_key", "private_key",
	"credential", "auth", "user", "host",
}

// MaskSensitive returns a copy of data with sensitive values masked for logging.
func MaskSensitive(data map[string]any) map[string]any {
	return maskMap(data)
}

func maskMap(m map[string]any) map[string]any {
	result := make(map[string]any, len(m))

	for key, val := range m {
		if isSensitiveKey(key) {
			result[key] = maskValue(val)
		} else {
			switch v := val.(type) {
			case map[string]any:
				result[key] = maskMap(v)
			case []any:
				result[key] = maskSlice(v)
			default:
				result[key] = val
			}
		}
	}

	return result
}

func maskSlice(s []any) []any {
	result := make([]any, len(s))

	for i, val := range s {
		switch v := val.(type) {
		case map[string]any:
			result[i] = maskMap(v)
		case []any:
			result[i] = maskSlice(v)
		default:
			result[i] = val
		}
	}

	return result
}

func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)

	for _, keyword := range sensitiveKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}

	return false
}

func maskValue(val any) string {
	s, ok := val.(string)
	if !ok {
		return "***"
	}

	if len(s) <= 4 { //nolint:mnd
		return "***"
	}

	const keepEdges = 2

	return s[:keepEdges] + strings.Repeat("*", len(s)-2*keepEdges) + s[len(s)-keepEdges:]
}
