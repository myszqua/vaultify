package vaultify

import (
	"context"
	"fmt"
	"sync"
	"time"

	vaultpkg "github.com/myszqua/vaultify/vault"
	"go.uber.org/zap"
)

// VaultDependencyProvider loads multiple dependency secrets from Vault at
// secret/{env}/{dependency}, skipping unavailable ones and polling for changes.
type VaultDependencyProvider struct {
	client       vaultpkg.Client
	env          string
	deps         []string
	priority     int
	logger       Logger
	pollInterval time.Duration
	cachedData   map[string]any
	mu           sync.RWMutex
	stopOnce     sync.Once
	stopCh       chan struct{}
}

// VaultDependencyOption configures a VaultDependencyProvider.
type VaultDependencyOption func(*VaultDependencyProvider)

// WithVaultDepPriority sets the provider priority.
func WithVaultDepPriority(p int) VaultDependencyOption {
	return func(vp *VaultDependencyProvider) {
		vp.priority = p
	}
}

// WithVaultDepPollInterval sets how often Vault is polled for changes.
func WithVaultDepPollInterval(d time.Duration) VaultDependencyOption {
	return func(vp *VaultDependencyProvider) {
		vp.pollInterval = d
	}
}

// WithVaultDepLogger sets the logger used by the provider.
func WithVaultDepLogger(l Logger) VaultDependencyOption {
	return func(vp *VaultDependencyProvider) {
		vp.logger = l
	}
}

// NewVaultDependencyProvider authenticates via AppRole and creates a provider
// that loads the given dependencies from Vault.
func NewVaultDependencyProvider(
	client vaultpkg.Client,
	env string,
	deps []string,
	opts ...VaultDependencyOption,
) (*VaultDependencyProvider, error) {
	const (
		priority                    = 30
		loginTimeoutSeconds         = 10
	)

	ctx, cancelfunc := context.WithDeadline(
		context.Background(),
		time.Now().Add(time.Second*loginTimeoutSeconds), //nolint:mnd
	)
	defer cancelfunc()

	if err := client.AppRoleLogin(ctx); err != nil {
		return nil, err
	}

	vp := &VaultDependencyProvider{
		client:       client,
		env:          env,
		deps:         deps,
		priority:     priority,
		logger:       newNopLogger(),
		pollInterval: 30 * time.Second, //nolint:mnd
		stopCh:       make(chan struct{}),
	}

	for _, opt := range opts {
		opt(vp)
	}

	return vp, nil
}

// Name returns "vault_deps".
func (vp *VaultDependencyProvider) Name() string {
	return "vault_deps"
}

// Priority returns the provider priority.
func (vp *VaultDependencyProvider) Priority() int {
	return vp.priority
}

// Load lists available secrets and reads each declared dependency.
func (vp *VaultDependencyProvider) Load(ctx context.Context) (map[string]any, error) {
	result := make(map[string]any)

	listResp, err := vp.client.ListSecrets(ctx)
	if err != nil {
		vp.logger.Warn("failed to list secrets", "error", err)

		return result, err
	}

	vp.logger.Info("got secrets", zap.Any("secrets", listResp))

	keys, ok := listResp.Data["keys"].([]interface{})
	if !ok {
		vp.logger.Warn("unexpected list response format")

		return result, nil
	}

	vp.logger.Info("got secrets", zap.Any("keys", keys))

	keysMap := make(map[string]bool)

	for _, k := range keys {
		if keyStr, ok := k.(string); ok {
			keysMap[keyStr] = true
		}
	}

	for _, dep := range vp.deps {
		vp.loadDependency(ctx, dep, keysMap, result)
	}

	vp.mu.Lock()
	vp.cachedData = result
	vp.mu.Unlock()

	vp.logger.Debug("loaded ", len(result), " dependencies from vault for env=", vp.env)

	return result, nil
}

func (vp *VaultDependencyProvider) loadDependency(
	ctx context.Context,
	dep string,
	keysMap map[string]bool,
	result map[string]any,
) {
	if _, ok := keysMap[dep]; !ok {
		vp.logger.Warn("failed to find dependency", zap.String("dependency", dep))

		return
	}

	resp, err := vp.client.ReadSecret(ctx, dep)
	if err != nil {
		vp.logger.Warn("failed to load dependency ", dep, err)

		vp.mu.RLock()
		cached := vp.cachedData
		vp.mu.RUnlock()

		if depData, ok := cached[dep]; ok {
			result[dep] = depData

			vp.logger.Info("loaded dependency ", dep, zap.Any("data", depData))

			return
		}

		vp.logger.Debug("skipping dependency ", dep, " due to load failure")

		return
	}

	result[dep] = resp.Data["data"]

	vp.logger.Info("loaded dependency ", dep, zap.Any("data", result[dep]))
}

// Watch starts the Vault polling loop.
func (vp *VaultDependencyProvider) Watch(ctx context.Context, events chan<- WatchEvent) error {
	go vp.pollLoop(ctx, events)

	return nil
}

func (vp *VaultDependencyProvider) pollLoop(ctx context.Context, events chan<- WatchEvent) {
	ticker := time.NewTicker(vp.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-vp.stopCh:
			return
		case <-ticker.C:
			vp.poll(ctx, events)
		}
	}
}

func (vp *VaultDependencyProvider) poll(ctx context.Context, events chan<- WatchEvent) {
	newData := make(map[string]any)

	for _, dep := range vp.deps {
		path := fmt.Sprintf("secret/%s/%s", vp.env, dep)

		resp, err := vp.client.ReadSecret(ctx, path)
		if err != nil {
			vp.logger.Debug("vault poll failed for dep ", dep, ": ", err)

			continue
		}

		newData[dep] = resp.Data
	}

	vp.mu.RLock()
	oldData := vp.cachedData
	vp.mu.RUnlock()

	if mapsEqual(oldData, newData) {
		return
	}

	vp.mu.Lock()
	vp.cachedData = newData
	vp.mu.Unlock()

	events <- WatchEvent{
		Source: vp.Name(),
		Type:   EventChanged,
	}
}

// Stop stops the polling loop.
func (vp *VaultDependencyProvider) Stop() error {
	vp.stopOnce.Do(func() {
		close(vp.stopCh)
	})

	return nil
}

// GetCached returns the last successfully cached dependency data.
func (vp *VaultDependencyProvider) GetCached() map[string]any {
	vp.mu.RLock()
	defer vp.mu.RUnlock()

	return vp.cachedData
}
