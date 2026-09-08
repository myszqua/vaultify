package pipe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDatabaseConfig(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "app",
		Password: "secret",
		Name:     "mydb",
		SSLMode:  "disable",
	}

	dsn := cfg.DSN()
	assert.Contains(t, dsn, "host=localhost")
	assert.Contains(t, dsn, "port=5432")
	assert.Contains(t, dsn, "user=app")
	assert.Contains(t, dsn, "password=secret")
	assert.Contains(t, dsn, "dbname=mydb")
	assert.Contains(t, dsn, "sslmode=disable")
}

func TestDatabaseConfigDefaultDSN(t *testing.T) {
	cfg := DatabaseConfig{}
	assert.Contains(t, cfg.DSN(), "port=0")
	assert.Contains(t, cfg.DSN(), "sslmode=")
}

func TestItoa(t *testing.T) {
	assert.Equal(t, "0", itoa(0))
	assert.Equal(t, "123", itoa(123))
	assert.Equal(t, "42", itoa(42))
}

func TestDatabaseConfigDefaults(t *testing.T) {
	cfg := DatabaseConfig{}
	assert.Equal(t, 0, cfg.Port)
	assert.Equal(t, "", cfg.SSLMode)
}

func TestRedisConfig(t *testing.T) {
	cfg := RedisConfig{
		ClusterMode: true,
		Nodes:       []string{"a:6379", "b:6379"},
		Password:    "pass",
		DB:          1,
	}

	assert.True(t, cfg.ClusterMode)
	assert.Len(t, cfg.Nodes, 2)
	assert.Equal(t, "pass", cfg.Password)
	assert.Equal(t, 1, cfg.DB)
}

func TestServerConfig(t *testing.T) {
	cfg := ServerConfig{
		Port:        8080,
		ServiceName: "myservice",
	}

	assert.Equal(t, 8080, cfg.Port)
	assert.Equal(t, "myservice", cfg.ServiceName)
	assert.Equal(t, "", cfg.Mode)
}

func TestLoggerConfig(t *testing.T) {
	cfg := LoggerConfig{
		Level:    "debug",
		Encoding: "console",
	}

	assert.Equal(t, "debug", cfg.Level)
	assert.Equal(t, "console", cfg.Encoding)
}

func TestTracerConfig(t *testing.T) {
	cfg := TracerConfig{
		ServiceName: "svc",
		SamplerType: "probabilistic",
	}

	assert.Equal(t, "svc", cfg.ServiceName)
	assert.Equal(t, "probabilistic", cfg.SamplerType)
}

func TestRateLimiterConfig(t *testing.T) {
	cfg := RateLimiterConfig{
		Rules: []RateLimitRule{
			{
				Clients:  []string{"client1"},
				Handlers: []string{"/api/v1"},
				Limit:    100,
				Burst:    50,
			},
		},
		Blacklist: BlacklistConfig{
			NumCounters: 100,
			MaxCost:     500,
			BufferItems: 10,
		},
	}

	assert.Len(t, cfg.Rules, 1)
	assert.Equal(t, 100, cfg.Rules[0].Limit)
	assert.Equal(t, 100, cfg.Blacklist.NumCounters)
	assert.Equal(t, int64(500), cfg.Blacklist.MaxCost)
}

func TestCircuitBreakerConfig(t *testing.T) {
	cfg := CircuitBreakerConfig{
		Timeout:          0,
		FailureThreshold: 0,
	}

	assert.Equal(t, 0, cfg.FailureThreshold)
	assert.Equal(t, 0, cfg.SuccessThreshold)
	assert.Equal(t, 0, cfg.MaxRequests)
}

func TestS3Config(t *testing.T) {
	cfg := S3Config{
		Endpoint:        "s3.example.com",
		Region:          "us-east-1",
		AccessKeyID:     "AKIA",
		SecretAccessKey: "secret",
		Bucket:          "mybucket",
	}

	assert.Equal(t, "s3.example.com", cfg.Endpoint)
	assert.Equal(t, "us-east-1", cfg.Region)
	assert.Equal(t, "AKIA", cfg.AccessKeyID)
	assert.Equal(t, "mybucket", cfg.Bucket)
}
