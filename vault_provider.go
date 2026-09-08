package vaultify

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	vaultpkg "github.com/myszqua/vaultify/vault"
)

// VaultProvider loads a single Vault secret at secret/{env}/{service} with
// cache fallback and poll-based change detection.
type VaultProvider struct {
	client       vaultpkg.Client
	env          string
	service      string
	priority     int
	logger       Logger
	pollInterval time.Duration
	cachedData   map[string]any
	mu           sync.RWMutex
	stopOnce     sync.Once
	stopCh       chan struct{}
}

// VaultProviderOption configures a VaultProvider.
type VaultProviderOption func(*VaultProvider)

// WithVaultPriority sets the provider priority.
func WithVaultPriority(p int) VaultProviderOption {
	return func(vp *VaultProvider) {
		vp.priority = p
	}
}

// WithVaultPollInterval sets how often Vault is polled for changes.
func WithVaultPollInterval(d time.Duration) VaultProviderOption {
	return func(vp *VaultProvider) {
		vp.pollInterval = d
	}
}

// WithVaultLogger sets the logger used by the provider.
func WithVaultLogger(l Logger) VaultProviderOption {
	return func(vp *VaultProvider) {
		vp.logger = l
	}
}

// NewVaultProvider creates a provider reading secret/{env}/{service} from Vault.
func NewVaultProvider(client vaultpkg.Client, env, service string, opts ...VaultProviderOption) *VaultProvider {
	const priority = 30

	vp := &VaultProvider{
		client:       client,
		env:          env,
		service:      service,
		priority:     priority,
		logger:       newNopLogger(),
		pollInterval: 30 * time.Second, //nolint:mnd
		stopCh:       make(chan struct{}),
	}

	for _, opt := range opts {
		opt(vp)
	}

	return vp
}

// Name returns "vault".
func (vp *VaultProvider) Name() string {
	return "vault"
}

// Priority returns the provider priority.
func (vp *VaultProvider) Priority() int {
	return vp.priority
}

// Load reads the Vault secret, falling back to cached data on failure.
func (vp *VaultProvider) Load(ctx context.Context) (map[string]any, error) {
	path := fmt.Sprintf("secret/%s/%s", vp.env, vp.service)

	resp, err := vp.client.ReadSecret(ctx, path)
	if err != nil {
		vp.logger.Warn("failed to read vault secret at ", path, ": ", err)

		vp.mu.RLock()
		cached := vp.cachedData
		vp.mu.RUnlock()

		if cached != nil {
			vp.logger.Warn("using cached vault data due to connection error")

			return cached, nil
		}

		return nil, fmt.Errorf("%w: %s: %w", ErrVaultUnavailable, path, err)
	}

	vp.mu.Lock()
	vp.cachedData = resp.Data
	vp.mu.Unlock()

	return resp.Data, nil
}

// Watch starts the Vault polling loop.
func (vp *VaultProvider) Watch(ctx context.Context, events chan<- WatchEvent) error {
	go vp.pollLoop(ctx, events)

	return nil
}

func (vp *VaultProvider) pollLoop(ctx context.Context, events chan<- WatchEvent) {
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

func (vp *VaultProvider) poll(ctx context.Context, events chan<- WatchEvent) {
	path := fmt.Sprintf("secret/%s/%s", vp.env, vp.service)

	resp, err := vp.client.ReadSecret(ctx, path)
	if err != nil {
		vp.logger.Warn("vault poll failed for ", path, ": ", err)

		events <- WatchEvent{
			Source: vp.Name(),
			Type:   EventError,
			Error:  fmt.Errorf("%w: %w", ErrVaultUnavailable, err),
		}

		return
	}

	vp.mu.RLock()
	oldData := vp.cachedData
	vp.mu.RUnlock()

	if mapsEqual(oldData, resp.Data) {
		return
	}

	vp.mu.Lock()
	vp.cachedData = resp.Data
	vp.mu.Unlock()

	events <- WatchEvent{
		Source: vp.Name(),
		Type:   EventChanged,
	}
}

// Stop stops the polling loop.
func (vp *VaultProvider) Stop() error {
	vp.stopOnce.Do(func() {
		close(vp.stopCh)
	})

	return nil
}

// GetCached returns the last successfully cached secret data.
func (vp *VaultProvider) GetCached() map[string]any {
	vp.mu.RLock()
	defer vp.mu.RUnlock()

	return vp.cachedData
}

func mapsEqual(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}

	for key, valA := range a {
		valB, ok := b[key]
		if !ok {
			return false
		}

		if fmt.Sprintf("%v", valA) != fmt.Sprintf("%v", valB) {
			return false
		}
	}

	return true
}

func splitVaultPath(path string) (env, service, key string) {
	parts := strings.SplitN(path, "/", 3) //nolint:mnd
	if len(parts) < 2 {                   //nolint:mnd
		return "", "", path
	}

	if len(parts) < 3 { //nolint:mnd
		return parts[0], parts[1], ""
	}

	return parts[0], parts[1], parts[2]
}
