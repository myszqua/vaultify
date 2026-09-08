package config

import (
	"fmt"
	"os"
	"time"

	"github.com/myszqua/vaultify/vault/models"
)

// Client configures the connection and authentication to a Vault server.
type Client struct {
	Address            string        `yaml:"address" validate:"required,url"`
	Timeout            time.Duration `yaml:"timeout" default:"10s" validate:"min=1s"`
	RetryBackoffMin    time.Duration `yaml:"retry_backoff_min" default:"100ms"`
	RetryBackoffMax    time.Duration `yaml:"retry_backoff_max" default:"400ms"`
	InsecureSkipVerify bool          `yaml:"insecure_skip_verify" default:"false"`
	DebugMode          bool          `yaml:"debug_mode" default:"false"`
}

// GetCertificatePath returns the mTLS PFX certificate path from the environment.
func (c *Client) GetCertificatePath() (string, error) {
	if p := os.Getenv(models.ENVCertPath); p != "" {
		return p, nil
	}

	return "", fmt.Errorf("not found certificate: %w", models.ErrEmptyENVField)
}

// GetCertificatePassword returns the mTLS PFX certificate password from the environment.
func (c *Client) GetCertificatePassword() (string, error) {
	if p := os.Getenv(models.ENVCertPassword); p != "" {
		return p, nil
	}

	return "", fmt.Errorf("not found certificate password: %w", models.ErrEmptyENVField)
}
