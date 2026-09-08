package models

import (
	"testing"

	"github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvert(t *testing.T) {
	t.Run("Full response", func(t *testing.T) {
		data := map[string]interface{}{
			"username": "admin",
			"password": "secret",
		}

		raw := &vault.Response[map[string]interface{}]{
			RequestID:     "req-123",
			LeaseID:       "lease-456",
			LeaseDuration: 3600,
			Renewable:     true,
			Data:          data,
			Warnings:      []string{"warning 1", "warning 2"},
		}

		resp := Convert(raw)

		require.NotNil(t, resp)
		assert.Equal(t, data, resp.Data)
		require.NotNil(t, resp.Metadata)
		assert.Equal(t, []string{"warning 1", "warning 2"}, resp.Metadata.Warnings)
		assert.Equal(t, "req-123", resp.Metadata.RequestID)
		assert.Equal(t, "lease-456", resp.LeaseID)
		assert.Equal(t, 3600, resp.LeaseDuration)
		assert.True(t, resp.Renewable)
	})

	t.Run("Empty response", func(t *testing.T) {
		raw := &vault.Response[map[string]interface{}]{}

		resp := Convert(raw)

		require.NotNil(t, resp)
		assert.Empty(t, resp.Data)
		require.NotNil(t, resp.Metadata)
		assert.Nil(t, resp.Metadata.Warnings)
		assert.Empty(t, resp.Metadata.RequestID)
		assert.Empty(t, resp.LeaseID)
		assert.Zero(t, resp.LeaseDuration)
		assert.False(t, resp.Renewable)
	})

	t.Run("Nil wrapped warnings", func(t *testing.T) {
		raw := &vault.Response[map[string]interface{}]{
			Data: map[string]interface{}{"key": "value"},
		}

		resp := Convert(raw)

		require.NotNil(t, resp)
		assert.Equal(t, map[string]interface{}{"key": "value"}, resp.Data)
		require.NotNil(t, resp.Metadata)
		assert.Nil(t, resp.Metadata.Warnings)
	})
}

func TestConvertFromList(t *testing.T) {
	t.Run("Full response", func(t *testing.T) {
		raw := &vault.Response[schema.StandardListResponse]{
			RequestID:     "req-list-1",
			LeaseID:       "lease-list",
			LeaseDuration: 120,
			Renewable:     true,
			Data: schema.StandardListResponse{
				Keys: []string{"app1", "app2"},
			},
			Warnings: []string{"warn"},
		}

		resp := ConvertFromList(raw)

		require.NotNil(t, resp)
		assert.Equal(t, []string{"app1", "app2"}, resp.Data)
		require.NotNil(t, resp.Metadata)
		assert.Equal(t, []string{"warn"}, resp.Metadata.Warnings)
		assert.Equal(t, "req-list-1", resp.Metadata.RequestID)
		assert.Equal(t, "lease-list", resp.LeaseID)
		assert.Equal(t, 120, resp.LeaseDuration)
		assert.True(t, resp.Renewable)
	})

	t.Run("Empty response", func(t *testing.T) {
		raw := &vault.Response[schema.StandardListResponse]{}

		resp := ConvertFromList(raw)

		require.NotNil(t, resp)
		assert.Nil(t, resp.Data)
		require.NotNil(t, resp.Metadata)
		assert.Nil(t, resp.Metadata.Warnings)
		assert.Empty(t, resp.Metadata.RequestID)
	})
}
