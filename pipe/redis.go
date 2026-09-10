package pipe

import (
	"github.com/myszqua/vaultify/models"
)

// RedisConfig standardizes a redis dependency's configuration.
type RedisConfig struct {
	ClusterMode           bool            `yaml:"cluster_mode" json:"cluster_mode" default:"false"`
	Nodes                 []string        `yaml:"nodes" json:"nodes"`
	Password              string          `yaml:"password" json:"password"`
	DB                    int             `yaml:"db" json:"db" default:"0"`
	PoolSize              int             `yaml:"pool_size" json:"pool_size" default:"10"`
	MaxRetries            int             `yaml:"max_retries" json:"max_retries" default:"3"`
	PoolConnectionTimeout models.Duration `yaml:"pool_connection_timeout" json:"pool_connection_timeout" default:"5s"`
	ReadTimeout           models.Duration `yaml:"read_timeout" json:"read_timeout" default:"3s"`
	WriteTimeout          models.Duration `yaml:"write_timeout" json:"write_timeout" default:"3s"`
	ObjectTTL             models.Duration `yaml:"object_ttl" json:"object_ttl" default:"24h"`
	MinRetryBackoff       models.Duration `yaml:"min_retry_backoff" json:"min_retry_backoff" default:"1s"`
	MaxRetryBackoff       models.Duration `yaml:"max_retry_backoff" json:"max_retry_backoff" default:"1s"`
}
