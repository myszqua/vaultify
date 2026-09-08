package vaultify

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/myszqua/vaultify/mocks"
	vaultmodels "github.com/myszqua/vaultify/vault/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestVaultProviderLoad(t *testing.T) {
	m := mocks.NewMockClient(t)
	m.EXPECT().
		ReadSecret(mock.Anything, "secret/prod/myservice").
		Return(&vaultmodels.Response{
			Data: map[string]any{
				"password": "secret123",
				"host":     "vault.example.com",
			},
		}, nil)

	provider := NewVaultProvider(m, "prod", "myservice")
	data, err := provider.Load(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "secret123", data["password"])
	assert.Equal(t, "vault.example.com", data["host"])
}

func TestVaultProviderLoadError(t *testing.T) {
	m := mocks.NewMockClient(t)
	m.EXPECT().
		ReadSecret(mock.Anything, "secret/prod/myservice").
		Return(nil, errors.New("connection refused"))

	provider := NewVaultProvider(m, "prod", "myservice")
	_, err := provider.Load(context.Background())

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrVaultUnavailable)
}

func TestVaultProviderLoadFallbackToCache(t *testing.T) {
	m := mocks.NewMockClient(t)
	m.EXPECT().
		ReadSecret(mock.Anything, "secret/prod/svc").
		Return(&vaultmodels.Response{
			Data: map[string]any{"key": "value"},
		}, nil).
		Once()
	m.EXPECT().
		ReadSecret(mock.Anything, "secret/prod/svc").
		Return(nil, errors.New("connection refused"))

	provider := NewVaultProvider(m, "prod", "svc")

	_, err := provider.Load(context.Background())
	require.NoError(t, err)

	data, err := provider.Load(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "value", data["key"])
}

func TestVaultProviderPriority(t *testing.T) {
	provider := NewVaultProvider(mocks.NewMockClient(t), "prod", "svc")
	assert.Equal(t, 30, provider.Priority())

	custom := NewVaultProvider(mocks.NewMockClient(t), "prod", "svc", WithVaultPriority(50))
	assert.Equal(t, 50, custom.Priority())
}

func TestVaultProviderName(t *testing.T) {
	provider := NewVaultProvider(mocks.NewMockClient(t), "prod", "svc")
	assert.Equal(t, "vault", provider.Name())
}

func TestVaultProviderGetCached(t *testing.T) {
	m := mocks.NewMockClient(t)
	m.EXPECT().
		ReadSecret(mock.Anything, "secret/prod/svc").
		Return(&vaultmodels.Response{
			Data: map[string]any{"secret": "data"},
		}, nil)

	provider := NewVaultProvider(m, "prod", "svc")
	_, _ = provider.Load(context.Background())

	cached := provider.GetCached()
	assert.Equal(t, "data", cached["secret"])
}

func TestVaultProviderStop(t *testing.T) {
	provider := NewVaultProvider(mocks.NewMockClient(t), "prod", "svc",
		WithVaultPollInterval(100*time.Millisecond))

	err := provider.Stop()
	assert.NoError(t, err)
}

func TestMapsEqual(t *testing.T) {
	a := map[string]any{"key": "value", "num": 42}
	b := map[string]any{"key": "value", "num": 42}
	c := map[string]any{"key": "value", "num": 99}

	assert.True(t, mapsEqual(a, b))
	assert.False(t, mapsEqual(a, c))
	assert.False(t, mapsEqual(a, map[string]any{"key": "value", "num": 42, "extra": true}))
	assert.True(t, mapsEqual(nil, nil))
}

func TestSplitVaultPath(t *testing.T) {
	env, svc, key := splitVaultPath("prod/myservice/db_password")
	assert.Equal(t, "prod", env)
	assert.Equal(t, "myservice", svc)
	assert.Equal(t, "db_password", key)

	env, svc, key = splitVaultPath("prod/myservice")
	assert.Equal(t, "prod", env)
	assert.Equal(t, "myservice", svc)
	assert.Empty(t, key)
}
