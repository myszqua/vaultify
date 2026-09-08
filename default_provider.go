package vaultify

import (
	"context"
	"fmt"
	"reflect"
)

// DefaultProvider supplies baseline configuration values from a map or
// struct-tag extraction with the lowest priority.
type DefaultProvider struct {
	defaults map[string]any
	priority int
	logger   Logger
}

// DefaultProviderOption configures a DefaultProvider.
type DefaultProviderOption func(*DefaultProvider)

// WithDefaultPriority sets the provider priority.
func WithDefaultPriority(p int) DefaultProviderOption {
	return func(dp *DefaultProvider) {
		dp.priority = p
	}
}

// WithDefaultLogger sets the logger used by the provider.
func WithDefaultLogger(l Logger) DefaultProviderOption {
	return func(dp *DefaultProvider) {
		dp.logger = l
	}
}

// NewDefaultProvider creates a provider with the given default map.
func NewDefaultProvider(defaults map[string]any, opts ...DefaultProviderOption) *DefaultProvider {
	const priority = 10

	dp := &DefaultProvider{
		defaults: defaults,
		priority: priority,
		logger:   newNopLogger(),
	}

	for _, opt := range opts {
		opt(dp)
	}

	return dp
}

// NewDefaultProviderFromStruct extracts defaults from struct `default` tags.
func NewDefaultProviderFromStruct(target any) *DefaultProvider {
	defaults := extractDefaults(target)

	return NewDefaultProvider(defaults)
}

// Name returns "default".
func (dp *DefaultProvider) Name() string {
	return "default"
}

// Priority returns the provider priority.
func (dp *DefaultProvider) Priority() int {
	return dp.priority
}

// Load returns the default values.
func (dp *DefaultProvider) Load(_ context.Context) (map[string]any, error) {
	return dp.defaults, nil
}

// Watch returns nil; the default provider does not support watching.
func (dp *DefaultProvider) Watch(_ context.Context, _ chan<- WatchEvent) error {
	dp.logger.Debug("default provider does not support watching")

	return nil
}

// Stop returns nil.
func (dp *DefaultProvider) Stop() error {
	return nil
}

func extractDefaults(target any) map[string]any {
	result := make(map[string]any)

	val := reflect.ValueOf(target)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return result
	}

	t := val.Type()

	for i := range t.NumField() {
		field := t.Field(i)

		tag := field.Tag.Get("default")
		if tag == "" {
			continue
		}

		key := field.Tag.Get("yaml")
		if key == "" {
			key = toSnakeCase(field.Name)
		}

		result[key] = parseDefaultTag(field.Type, tag)
	}

	return result
}

func parseDefaultTag(t reflect.Type, tag string) any { //nolint:ireturn
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return tag
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var v int
		_, _ = fmt.Sscanf(tag, "%d", &v)

		return v
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var v uint64
		_, _ = fmt.Sscanf(tag, "%d", &v)

		return v
	case reflect.Float32, reflect.Float64:
		var v float64
		_, _ = fmt.Sscanf(tag, "%f", &v)

		return v
	case reflect.Bool:
		return tag == "true"
	default:
		return tag
	}
}

func toSnakeCase(s string) string {
	var result []rune

	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}

		result = append(result, r)
	}

	return string(result)
}
