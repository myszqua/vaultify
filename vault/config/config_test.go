package config

import (
	"errors"
	"testing"
	"time"

	"github.com/myszqua/vaultify/vault/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientConfig(t *testing.T) {
	t.Run("Configured values", func(t *testing.T) {
		cfg := Client{
			Address:         "http://localhost:8200",
			Timeout:         30 * time.Second,
			RetryBackoffMin: 200 * time.Millisecond,
			RetryBackoffMax: 2 * time.Second,
		}

		assert.Equal(t, "http://localhost:8200", cfg.Address)
		assert.Equal(t, 30*time.Second, cfg.Timeout)
		assert.Equal(t, 200*time.Millisecond, cfg.RetryBackoffMin)
		assert.Equal(t, 2*time.Second, cfg.RetryBackoffMax)
	})

	t.Run("Zero values", func(t *testing.T) {
		cfg := Client{}

		assert.Empty(t, cfg.Address)
		assert.Zero(t, cfg.Timeout)
		assert.Zero(t, cfg.RetryBackoffMin)
		assert.Zero(t, cfg.RetryBackoffMax)
	})
}

func TestClientGetCertificatePath(t *testing.T) {
	t.Run("Set env", func(t *testing.T) {
		t.Setenv(models.ENVCertPath, "/path/to/client.pfx")

		cfg := Client{}

		p, err := cfg.GetCertificatePath()
		require.NoError(t, err)
		assert.Equal(t, "/path/to/client.pfx", p)
	})

	t.Run("Empty env", func(t *testing.T) {
		t.Setenv(models.ENVCertPath, "")

		cfg := Client{}

		p, err := cfg.GetCertificatePath()
		require.Error(t, err)
		assert.True(t, errors.Is(err, models.ErrEmptyENVField))
		assert.Empty(t, p)
	})
}

func TestClientGetCertificatePassword(t *testing.T) {
	t.Run("Set env", func(t *testing.T) {
		t.Setenv(models.ENVCertPassword, "s3cr3t")

		cfg := Client{}

		p, err := cfg.GetCertificatePassword()
		require.NoError(t, err)
		assert.Equal(t, "s3cr3t", p)
	})

	t.Run("Empty env", func(t *testing.T) {
		t.Setenv(models.ENVCertPassword, "")

		cfg := Client{}

		p, err := cfg.GetCertificatePassword()
		require.Error(t, err)
		assert.True(t, errors.Is(err, models.ErrEmptyENVField))
		assert.Empty(t, p)
	})
}
