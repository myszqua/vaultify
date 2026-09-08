package models

// Dependency names a supported configurable component.
type Dependency string

const (
	// DependencyDatabase is the database dependency.
	DependencyDatabase Dependency = "database"
	// DependencyRedis is the redis dependency.
	DependencyRedis Dependency = "redis"
	// DependencyS3 is the s3 dependency.
	DependencyS3 Dependency = "s3"
	// DependencyRateLimiter is the rate limiter dependency.
	DependencyRateLimiter Dependency = "rate_limiter"
	// DependencyCircuitBreaker is the circuit breaker dependency.
	DependencyCircuitBreaker Dependency = "circuit_breaker"
	// DependencyLogger is the logger dependency.
	DependencyLogger Dependency = "logger"
	// DependencyTracer is the tracer dependency.
	DependencyTracer Dependency = "tracer"
	// DependencyServer is the server dependency.
	DependencyServer Dependency = "server"
	// DependencyVault is the vault dependency.
	DependencyVault Dependency = "vault"
	// DependencyConfig is the config dependency.
	DependencyConfig Dependency = "config"
)

// AllDependencies lists every built-in dependency name.
var AllDependencies = []Dependency{
	DependencyDatabase,
	DependencyRedis,
	DependencyS3,
	DependencyRateLimiter,
	DependencyCircuitBreaker,
	DependencyLogger,
	DependencyTracer,
	DependencyServer,
}
