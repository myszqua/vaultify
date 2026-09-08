package vaultify

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	vaultpkg "github.com/myszqua/vaultify/vault"
)

var dynamicPattern = regexp.MustCompile(`\$\{([^}]+)}`)

// DynamicResolver resolves ${...} placeholders (env, vault, file) in config values.
type DynamicResolver struct {
	vaultClient vaultpkg.Client
	fileReader  func(string) ([]byte, error)
}

// DynamicResolverOption configures a DynamicResolver.
type DynamicResolverOption func(*DynamicResolver)

// WithVaultClient provides a vault client used to resolve ${VAULT:...} tokens.
func WithVaultClient(c vaultpkg.Client) DynamicResolverOption {
	return func(dr *DynamicResolver) {
		dr.vaultClient = c
	}
}

// WithFileReader overrides the default file reader used to resolve ${FILE:...} tokens.
func WithFileReader(fn func(string) ([]byte, error)) DynamicResolverOption {
	return func(dr *DynamicResolver) {
		dr.fileReader = fn
	}
}

// NewDynamicResolver creates a resolver with the given options.
func NewDynamicResolver(opts ...DynamicResolverOption) *DynamicResolver {
	dr := &DynamicResolver{
		fileReader: os.ReadFile,
	}

	for _, opt := range opts {
		opt(dr)
	}

	return dr
}

// Resolve replaces all ${...} placeholders within input.
func (dr *DynamicResolver) Resolve(input string) (string, error) {
	if !strings.Contains(input, "${") {
		return input, nil
	}

	result := dynamicPattern.ReplaceAllStringFunc(input, func(match string) string {
		content := strings.TrimPrefix(match, "${")
		content = strings.TrimSuffix(content, "}")

		resolved, err := dr.resolveToken(content)
		if err != nil {
			return match
		}

		return resolved
	})

	return result, nil
}

func (dr *DynamicResolver) resolveToken(token string) (string, error) {
	switch {
	case strings.HasPrefix(token, "VAULT:"):
		return dr.resolveVault(token)
	case strings.HasPrefix(token, "FILE:"):
		return dr.resolveFile(token)
	default:
		return dr.resolveEnv(token)
	}
}

func (dr *DynamicResolver) resolveEnv(key string) (string, error) {
	return os.Getenv(key), nil
}

func (dr *DynamicResolver) resolveVault(token string) (string, error) {
	if dr.vaultClient == nil {
		return "", fmt.Errorf("vault client not configured")
	}

	path := strings.TrimPrefix(token, "VAULT:")
	parts := strings.SplitN(path, ":", 2) //nolint:mnd

	vaultPath := parts[0]

	key := ""
	if len(parts) > 1 { //nolint:mnd
		key = parts[1]
	}

	resp, err := dr.vaultClient.ReadSecret(context.Background(), vaultPath)
	if err != nil {
		return "", fmt.Errorf("vault read %s: %w", vaultPath, err)
	}

	if key != "" {
		if val, ok := resp.Data[key]; ok {
			return fmt.Sprintf("%v", val), nil
		}

		return "", fmt.Errorf("key %q not found in vault path %s", key, vaultPath)
	}

	return fmt.Sprintf("%v", resp.Data), nil
}

func (dr *DynamicResolver) resolveFile(token string) (string, error) {
	path := strings.TrimPrefix(token, "FILE:")

	data, err := dr.fileReader(path)
	if err != nil {
		return "", fmt.Errorf("read file %s: %w", path, err)
	}

	return strings.TrimSpace(string(data)), nil
}

// ResolveAll resolves placeholders recursively across all values in settings.
func ResolveAll(settings map[string]any, resolver *DynamicResolver) (map[string]any, error) {
	return resolveMap(settings, resolver)
}

func resolveMap(m map[string]any, resolver *DynamicResolver) (map[string]any, error) {
	result := make(map[string]any, len(m))

	for key, val := range m {
		switch v := val.(type) {
		case string:
			resolved, err := resolver.Resolve(v)
			if err != nil {
				return nil, fmt.Errorf("resolve key %s: %w", key, err)
			}

			result[key] = resolved
		case map[string]any:
			resolved, err := resolveMap(v, resolver)
			if err != nil {
				return nil, err
			}

			result[key] = resolved
		case []any:
			resolved, err := resolveSlice(v, resolver)
			if err != nil {
				return nil, err
			}

			result[key] = resolved
		default:
			result[key] = val
		}
	}

	return result, nil
}

func resolveSlice(s []any, resolver *DynamicResolver) ([]any, error) {
	result := make([]any, len(s))

	for i, val := range s {
		switch v := val.(type) {
		case string:
			resolved, err := resolver.Resolve(v)
			if err != nil {
				return nil, err
			}

			result[i] = resolved
		case map[string]any:
			resolved, err := resolveMap(v, resolver)
			if err != nil {
				return nil, err
			}

			result[i] = resolved
		case []any:
			resolved, err := resolveSlice(v, resolver)
			if err != nil {
				return nil, err
			}

			result[i] = resolved
		default:
			result[i] = val
		}
	}

	return result, nil
}
