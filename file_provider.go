package vaultify

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

// FileProvider loads configuration from YAML, JSON, or TOML files.
// Multiple files layer together with later files overriding earlier ones.
type FileProvider struct {
	paths       []string
	priority    int
	logger      Logger
	watcher     *fsnotify.Watcher
	watchCtx    context.Context
	watchCancel context.CancelFunc
	mu          sync.Mutex
	stopOnce    sync.Once
}

// FileProviderOption configures a FileProvider.
type FileProviderOption func(*FileProvider)

// WithFilePriority sets the provider priority.
func WithFilePriority(p int) FileProviderOption {
	return func(fp *FileProvider) {
		fp.priority = p
	}
}

// WithFileLogger sets the logger used by the provider.
func WithFileLogger(l Logger) FileProviderOption {
	return func(fp *FileProvider) {
		fp.logger = l
	}
}

// NewFileProvider creates a provider that loads the given file paths.
func NewFileProvider(paths []string, opts ...FileProviderOption) *FileProvider {
	const priority = 20

	fp := &FileProvider{
		paths:    paths,
		priority: priority,
		logger:   newNopLogger(),
	}

	for _, opt := range opts {
		opt(fp)
	}

	return fp
}

// Name returns "file".
func (fp *FileProvider) Name() string {
	return "file"
}

// Priority returns the provider priority.
func (fp *FileProvider) Priority() int {
	return fp.priority
}

// Load reads and parses each configured file, merging them in order.
func (fp *FileProvider) Load(_ context.Context) (map[string]any, error) {
	result := make(map[string]any)

	for _, path := range fp.paths {
		data, err := os.ReadFile(path) //nolint:gosec // file paths are user-configured
		if err != nil {
			if os.IsNotExist(err) {
				fp.logger.Debug("config file not found, skipping: ", path)

				continue
			}

			return nil, fmt.Errorf("read file %s: %w", path, err)
		}

		parsed, err := fp.parseFile(path, data)
		if err != nil {
			return nil, fmt.Errorf("parse file %s: %w", path, err)
		}

		result = mergeMaps(result, parsed)
	}

	return result, nil
}

func (fp *FileProvider) parseFile(path string, data []byte) (map[string]any, error) {
	format, ok := detectFormat(path)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, filepath.Ext(path))
	}

	var result map[string]any

	switch format {
	case FormatYAML:
		if err := yaml.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("yaml parse: %w", err)
		}
	case FormatJSON:
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("json parse: %w", err)
		}
	case FormatTOML:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, filepath.Ext(path))
	}

	return result, nil
}

// Watch starts fsnotify watching on the directories containing the config
// files and emits events on file changes.
func (fp *FileProvider) Watch(ctx context.Context, events chan<- WatchEvent) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create file watcher: %w", err)
	}

	fp.watcher = watcher
	fp.watchCtx, fp.watchCancel = context.WithCancel(ctx)

	for _, path := range fp.paths {
		dir := filepath.Dir(path)

		if err := watcher.Add(dir); err != nil {
			watcher.Close()

			return fmt.Errorf("watch directory %s: %w", dir, err)
		}
	}

	go fp.watchLoop(events)

	return nil
}

func (fp *FileProvider) watchLoop(events chan<- WatchEvent) {
	for {
		select {
		case <-fp.watchCtx.Done():
			return
		case event, ok := <-fp.watcher.Events:
			if !ok {
				return
			}

			if !fp.isRelevantEvent(event) {
				continue
			}

			fp.logger.Debug("file change detected: ", event.Name)

			events <- WatchEvent{
				Source: fp.Name(),
				Type:   EventChanged,
			}
		case err, ok := <-fp.watcher.Errors:
			if !ok {
				return
			}

			events <- WatchEvent{
				Source: fp.Name(),
				Type:   EventError,
				Error:  err,
			}
		}
	}
}

func (fp *FileProvider) isRelevantEvent(event fsnotify.Event) bool {
	if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
		return false
	}

	for _, path := range fp.paths {
		if event.Name == path || event.Name == filepath.Dir(path) {
			return true
		}
	}

	return false
}

// Stop stops file watching and closes the watcher.
func (fp *FileProvider) Stop() error {
	fp.stopOnce.Do(func() {
		fp.mu.Lock()
		defer fp.mu.Unlock()

		if fp.watchCancel != nil {
			fp.watchCancel()
		}

		if fp.watcher != nil {
			fp.watcher.Close()
		}
	})

	return nil
}

func detectFormat(path string) (FileFormat, bool) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".yaml", ".yml":
		return FormatYAML, true
	case ".json":
		return FormatJSON, true
	case ".toml":
		return FormatTOML, true
	default:
		return 0, false
	}
}

func mergeMaps(dst, src map[string]any) map[string]any {
	for key, srcVal := range src {
		dstVal, exists := dst[key]
		if !exists {
			dst[key] = srcVal

			continue
		}

		srcMap, srcOk := srcVal.(map[string]any)
		dstMap, dstOk := dstVal.(map[string]any)

		if srcOk && dstOk {
			dst[key] = mergeMaps(dstMap, srcMap)
		} else {
			dst[key] = srcVal
		}
	}

	return dst
}
