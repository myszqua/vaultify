package pipe

import "time"

// CircuitBreakerConfig standardizes a circuit breaker dependency's configuration.
type CircuitBreakerConfig struct {
	Timeout          time.Duration `yaml:"timeout" json:"timeout" default:"60s"`
	FailureThreshold int           `yaml:"failure_threshold" json:"failure_threshold" default:"5"`
	SuccessThreshold int           `yaml:"success_threshold" json:"success_threshold" default:"3"`
	MaxRequests      int           `yaml:"max_requests" json:"max_requests" default:"10"`
}
