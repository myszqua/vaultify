package vaultify

import (
	"errors"
	"fmt"
)

var (
	// ErrProviderNotFound is returned when a requested provider does not exist.
	ErrProviderNotFound = errors.New("provider not found")
	// ErrKeyNotFound is returned when a configuration key is missing.
	ErrKeyNotFound = errors.New("key not found")
	// ErrValidationFailed is returned when config validation fails.
	ErrValidationFailed = errors.New("validation failed")
	// ErrFileNotFound is returned when a config file does not exist.
	ErrFileNotFound = errors.New("config file not found")
	// ErrUnsupportedFormat is returned for unknown configuration formats.
	ErrUnsupportedFormat = errors.New("unsupported config format")
	// ErrVaultUnavailable is returned when Vault is unreachable.
	ErrVaultUnavailable = errors.New("vault unavailable")
	// ErrEmptyConfig is returned when the loaded configuration is empty.
	ErrEmptyConfig = errors.New("config is empty")
)

// ValidationError describes a failed struct-tag validation rule.
type ValidationError struct {
	Field   string
	Rule    string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field %q: %s (%s)", e.Field, e.Message, e.Rule)
}

func (e *ValidationError) Unwrap() error {
	return ErrValidationFailed
}

// MergeConflict describes a type conflict encountered during deep merge.
type MergeConflict struct {
	Key     string
	Source1 string
	Source2 string
	Value1  any
	Value2  any
}

func (e *MergeConflict) Error() string {
	return fmt.Sprintf("merge conflict for key %q between %s and %s", e.Key, e.Source1, e.Source2)
}

// ConfigError wraps errors that occur during configuration loading/unmarshaling.
type ConfigError struct {
	Op  string
	Err error
	Key string
}

func (e *ConfigError) Error() string {
	if e.Key != "" {
		return fmt.Sprintf("config %s for key %q: %s", e.Op, e.Key, e.Err)
	}

	return fmt.Sprintf("config %s: %s", e.Op, e.Err)
}

func (e *ConfigError) Unwrap() error {
	return e.Err
}
