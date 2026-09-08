package vaultify

import "context"

// Provider is the interface implemented by all configuration sources.
// Providers are sorted by ascending priority; higher-priority providers
// override lower-priority values during merge.
type Provider interface {
	Name() string
	Load(ctx context.Context) (map[string]any, error)
	Priority() int
	Watch(ctx context.Context, events chan<- WatchEvent) error
	Stop() error
}

func sortProvidersByPriority(providers []Provider) []Provider {
	sorted := make([]Provider, len(providers))
	copy(sorted, providers)

	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j].Priority() < sorted[j-1].Priority(); j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}

	return sorted
}
