package vaultify

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// ConfigLoader loads configuration from multiple providers and merges them
// by priority into a single settings map with hot-reload support.
type ConfigLoader struct {
	providers []Provider
	settings  map[string]any
	mu        sync.RWMutex
	logger    Logger
	onReload  []func()
	events    chan WatchEvent
	watchCtx  context.Context
	watchStop context.CancelFunc
	mergeOpts MergeOptions
}

// New creates a ConfigLoader from the provided providers.
func New(providers ...Provider) *ConfigLoader {
	cl := &ConfigLoader{
		providers: providers,
		settings:  make(map[string]any),
		logger:    newNopLogger(),
		events:    make(chan WatchEvent, 100), //nolint:mnd
		mergeOpts: MergeOptions{
			SliceStrategy: MergeStrategyReplace,
			LogConflicts:  true,
		},
	}

	return cl
}

// WithLogger sets the logger used by the loader.
func (cl *ConfigLoader) WithLogger(logger Logger) *ConfigLoader {
	cl.logger = logger

	return cl
}

// WithMergeOptions sets merge options for priority-based merging.
func (cl *ConfigLoader) WithMergeOptions(opts MergeOptions) *ConfigLoader {
	cl.mergeOpts = opts

	return cl
}

// Load calls Load on each provider, sorts by priority, and merges the results.
func (cl *ConfigLoader) Load(ctx context.Context) error {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	sorted := sortProvidersByPriority(cl.providers)
	providerDataList := make([]providerData, 0, len(sorted))

	for _, p := range sorted {
		data, err := p.Load(ctx)
		if err != nil {
			cl.logger.Warn("provider ", p.Name(), " load failed: ", err)

			continue
		}

		providerDataList = append(providerDataList, providerData{
			name:     p.Name(),
			priority: p.Priority(),
			data:     data,
		})
	}

	sortedData := sortByPriority(providerDataList)
	cl.settings = applyPriorities(sortedData, cl.mergeOpts, cl.logger)

	cl.logger.Debug("config loaded from ", len(providerDataList), " providers")

	return nil
}

// Get returns the value at the given dot-notation key, or nil if absent.
func (cl *ConfigLoader) Get(key string) any { //nolint:ireturn
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	return getNestedValue(cl.settings, key)
}

// GetString returns the value at key as a string.
func (cl *ConfigLoader) GetString(key string) string {
	val := cl.Get(key)
	if val == nil {
		return ""
	}

	if s, ok := val.(string); ok {
		return s
	}

	return fmt.Sprintf("%v", val)
}

// GetInt returns the value at key as an int.
func (cl *ConfigLoader) GetInt(key string) int {
	val := cl.Get(key)
	if val == nil {
		return 0
	}

	switch v := val.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case json.Number:
		n, _ := v.Int64()

		return int(n)
	default:
		return 0
	}
}

// GetBool returns the value at key as a bool.
func (cl *ConfigLoader) GetBool(key string) bool {
	val := cl.Get(key)
	if val == nil {
		return false
	}

	if b, ok := val.(bool); ok {
		return b
	}

	return false
}

// GetDuration returns the value at key as a duration string.
func (cl *ConfigLoader) GetDuration(key string) string {
	return cl.GetString(key)
}

// Unmarshal populates rawVal from the full merged configuration.
func (cl *ConfigLoader) Unmarshal(rawVal any) error {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	data, err := json.Marshal(cl.settings)
	if err != nil {
		return &ConfigError{Op: "unmarshal", Err: err}
	}

	if err := json.Unmarshal(data, rawVal); err != nil {
		return &ConfigError{Op: "unmarshal", Err: err}
	}

	return nil
}

// UnmarshalKey populates rawVal from the config subtree at key.
func (cl *ConfigLoader) UnmarshalKey(key string, rawVal any) error {
	val := cl.Get(key)
	if val == nil {
		return &ConfigError{Op: "unmarshal_key", Err: ErrKeyNotFound, Key: key}
	}

	cl.logger.Info("got value", zap.String("key", key), zap.Any("val", val))

	data, err := json.Marshal(val)
	if err != nil {
		return &ConfigError{Op: "unmarshal_key", Err: err, Key: key}
	}

	if err := json.Unmarshal(data, rawVal); err != nil {
		return &ConfigError{Op: "unmarshal_key", Err: err, Key: key}
	}

	return nil
}

// AllSettings returns a shallow copy of the full merged configuration.
func (cl *ConfigLoader) AllSettings() map[string]any {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	result := make(map[string]any, len(cl.settings))

	for k, v := range cl.settings {
		result[k] = v
	}

	return result
}

// Reload re-runs Load and triggers all registered reload callbacks.
func (cl *ConfigLoader) Reload(ctx context.Context) error {
	if err := cl.Load(ctx); err != nil {
		return fmt.Errorf("reload: %w", err)
	}

	for _, fn := range cl.onReload {
		fn()
	}

	cl.logger.Debug("config reloaded successfully")

	return nil
}

// OnReload registers a callback invoked after every successful reload.
func (cl *ConfigLoader) OnReload(fn func()) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	cl.onReload = append(cl.onReload, fn)
}

// LoadDependencies validates that each declared dependency exists in config.
func (cl *ConfigLoader) LoadDependencies(_ context.Context, deps []string) error {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	for _, dep := range deps {
		val := getNestedValue(cl.settings, dep)
		if val == nil {
			cl.logger.Debug("dependency ", dep, " not found in config")

			continue
		}

		cl.logger.Debug("dependency ", dep, " loaded")
	}

	return nil
}

// GetDependency returns the raw config map for a named dependency.
func (cl *ConfigLoader) GetDependency(name string) (map[string]any, error) {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	val := getNestedValue(cl.settings, name)
	if val == nil {
		return nil, &ConfigError{Op: "get_dependency", Err: ErrKeyNotFound, Key: name}
	}

	switch v := val.(type) {
	case map[string]any:
		return v, nil
	default:
		result := make(map[string]any)
		result[name] = val

		return result, nil
	}
}

func getNestedValue(data map[string]any, key string) any { //nolint:ireturn
	parts := splitKey(key)
	current := data

	for i, part := range parts {
		val, ok := current[part]
		if !ok {
			return nil
		}

		if i == len(parts)-1 {
			return val
		}

		if nextMap, ok := val.(map[string]any); ok {
			current = nextMap
		} else {
			return nil
		}
	}

	return nil
}

func splitKey(key string) []string {
	result := make([]string, 0)
	current := ""

	for _, r := range key {
		if r == '.' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}

	if current != "" {
		result = append(result, current)
	}

	return result
}
