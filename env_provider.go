package vaultify

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync"
)

// EnvProvider loads configuration from environment variables with an
// optional prefix and automatic type casting.
type EnvProvider struct {
	prefix    string
	delimiter rune
	priority  int
	logger    Logger
	envCache  map[string]string
	mu        sync.RWMutex
	stopOnce  sync.Once
	stopCh    chan struct{}
}

// EnvProviderOption configures an EnvProvider.
type EnvProviderOption func(*EnvProvider)

// WithEnvPriority sets the provider priority.
func WithEnvPriority(p int) EnvProviderOption {
	return func(ep *EnvProvider) {
		ep.priority = p
	}
}

// WithEnvDelimiter sets the key delimiter used to build nested maps.
func WithEnvDelimiter(d rune) EnvProviderOption {
	return func(ep *EnvProvider) {
		ep.delimiter = d
	}
}

// WithEnvLogger sets the logger used by the provider.
func WithEnvLogger(l Logger) EnvProviderOption {
	return func(ep *EnvProvider) {
		ep.logger = l
	}
}

// NewEnvProvider creates a provider that reads env vars with the given prefix.
func NewEnvProvider(prefix string, opts ...EnvProviderOption) *EnvProvider {
	const priority = 40

	ep := &EnvProvider{
		prefix:    prefix,
		delimiter: '_',
		priority:  priority,
		logger:    newNopLogger(),
		stopCh:    make(chan struct{}),
	}

	for _, opt := range opts {
		opt(ep)
	}

	return ep
}

// Name returns "env".
func (ep *EnvProvider) Name() string {
	return "env"
}

// Priority returns the provider priority.
func (ep *EnvProvider) Priority() int {
	return ep.priority
}

// Load reads all env vars with the configured prefix and builds a nested map.
func (ep *EnvProvider) Load(_ context.Context) (map[string]any, error) {
	result := make(map[string]any)

	ep.mu.Lock()
	ep.envCache = make(map[string]string)
	ep.mu.Unlock()

	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2) //nolint:mnd
		if len(pair) != 2 {                 //nolint:mnd
			continue
		}

		key, value := pair[0], pair[1]

		if !strings.HasPrefix(key, ep.prefix) {
			continue
		}

		trimmed := strings.TrimPrefix(key, ep.prefix)
		if trimmed == "" {
			continue
		}

		ep.mu.Lock()
		ep.envCache[key] = value
		ep.mu.Unlock()

		configKey := strings.ToLower(strings.TrimPrefix(trimmed, "_"))
		nested := envToMap(configKey, value, ep.delimiter)

		result = mergeMaps(result, nested)
	}

	ep.logger.Debug("loaded ", len(ep.envCache), " environment variables with prefix ", ep.prefix)

	return result, nil
}

// Watch emits periodic change events; env vars are polled.
func (ep *EnvProvider) Watch(_ context.Context, _ chan<- WatchEvent) error {
	ep.logger.Debug("env provider uses polling for watch")

	// todo: implement after vaultify-server

	// go func() {
	//	ticker := time.NewTicker(5 * time.Second) //nolint:mnd
	//	defer ticker.Stop()
	//
	//	for {
	//		select {
	//		case <-ctx.Done():
	//			return
	//		case <-ep.stopCh:
	//			return
	//		case <-ticker.C:
	//			events <- WatchEvent{
	//				Source: ep.Name(),
	//				Type:   EventChanged,
	//			}
	//		}
	//	}
	// }()

	return nil
}

// Stop stops env polling and clears the env cache.
func (ep *EnvProvider) Stop() error {
	ep.stopOnce.Do(func() {
		close(ep.stopCh)

		ep.mu.Lock()
		defer ep.mu.Unlock()

		ep.envCache = nil
	})

	return nil
}

// Getenv returns a cached env value and whether it was cached.
func (ep *EnvProvider) Getenv(key string) (string, bool) {
	ep.mu.RLock()
	defer ep.mu.RUnlock()

	val, ok := ep.envCache[key]

	return val, ok
}

func envToMap(key, value string, _ rune) map[string]any {
	parts := strings.Split(key, "_")

	result := make(map[string]any)
	current := result

	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = castEnvValue(value)

			continue
		}

		next, ok := current[part]
		if !ok {
			nextMap := make(map[string]any)
			current[part] = nextMap
			current = nextMap

			continue
		}

		nextMap, mapOk := next.(map[string]any)
		if !mapOk {
			nextMap = make(map[string]any)
			current[part] = nextMap
		}

		current = nextMap
	}

	return result
}

func castEnvValue(value string) any { //nolint:ireturn
	if value == "" {
		return ""
	}

	if value == "true" {
		return true
	}

	if value == "false" {
		return false
	}

	if i, err := strconv.Atoi(value); err == nil {
		return i
	}

	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}

	return value
}
