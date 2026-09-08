package pipe

import (
	"github.com/myszqua/vaultify/models"
)

// RateLimiterConfig standardizes a rate limiter dependency's configuration.
type RateLimiterConfig struct {
	Rules     []RateLimitRule `yaml:"rules" json:"rules"`
	Blacklist BlacklistConfig `yaml:"blacklist" json:"blacklist"`
}

// RateLimitRule defines a single rate limiting rule.
type RateLimitRule struct {
	Clients  []string        `yaml:"clients" json:"clients"`
	Handlers []string        `yaml:"handlers" json:"handlers"`
	Limit    int             `yaml:"limit" json:"limit" config:"required"`
	Burst    int             `yaml:"burst" json:"burst" config:"required"`
	Timeout  models.Duration `yaml:"timeout" json:"timeout" default:"10s"`
}

// BlacklistConfig configures the blacklist state store.
type BlacklistConfig struct {
	NumCounters int             `yaml:"num_counters" json:"num_counters" default:"10000"`
	MaxCost     int64           `yaml:"max_cost" json:"max_cost" default:"100000"`
	BufferItems int             `yaml:"buffer_items" json:"buffer_items" default:"64"`
	TTL         models.Duration `yaml:"ttl" json:"ttl" default:"10m"`
}
