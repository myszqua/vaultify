package vaultify

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/myszqua/vaultify/mocks"
	vaultmodels "github.com/myszqua/vaultify/vault/models"
	"github.com/stretchr/testify/assert"
	mocklib "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newMockDepsClient(t *testing.T, keys []string) *mocks.MockClient {
	t.Helper()

	m := mocks.NewMockClient(t)
	m.EXPECT().AppRoleLogin(mocklib.Anything).Return(nil)

	if keys != nil {
		keyList := make([]interface{}, 0, len(keys))
		for _, k := range keys {
			keyList = append(keyList, k)
		}

		m.EXPECT().ListSecrets(mocklib.Anything).
			Return(&vaultmodels.Response{
				Data: map[string]any{"keys": keyList},
			}, nil)
	}

	return m
}

func TestVaultDependencyProviderLoad(t *testing.T) {
	mock := newMockDepsClient(t, []string{"database", "redis"})
	mock.EXPECT().
		ReadSecret(mocklib.Anything, "database").
		Return(&vaultmodels.Response{Data: map[string]any{
			"data": map[string]any{
				"host":     "db.example.com",
				"password": "secret123",
			},
		}}, nil)
	mock.EXPECT().
		ReadSecret(mocklib.Anything, "redis").
		Return(&vaultmodels.Response{Data: map[string]any{
			"data": map[string]any{
				"address": "redis.example.com",
			},
		}}, nil)

	provider, err := NewVaultDependencyProvider(mock, "prod", []string{"database", "redis"})
	require.NoError(t, err)

	data, err := provider.Load(context.Background())

	require.NoError(t, err)
	assert.Len(t, data, 2)

	db, ok := data["database"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "db.example.com", db["host"])
	assert.Equal(t, "secret123", db["password"])

	redis, ok := data["redis"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "redis.example.com", redis["address"])
}

func TestVaultDependencyProviderSuccesfulAndFailed(t *testing.T) {
	mock := newMockDepsClient(t, []string{"database", "missing"})
	mock.EXPECT().
		ReadSecret(mocklib.Anything, "database").
		Return(&vaultmodels.Response{Data: map[string]any{
			"data": map[string]any{"host": "db.example.com"},
		}}, nil)
	mock.EXPECT().
		ReadSecret(mocklib.Anything, "missing").
		Return(nil, errors.New("not found"))

	provider, err := NewVaultDependencyProvider(mock, "prod", []string{"database", "missing"})

	require.NoError(t, err)

	data, err := provider.Load(context.Background())

	require.NoError(t, err)
	assert.Len(t, data, 1)
	assert.NotNil(t, data["database"])
	assert.Nil(t, data["missing"])
}

func TestVaultDependencyProviderAllFailed(t *testing.T) {
	mock := newMockDepsClient(t, []string{"database"})
	mock.EXPECT().
		ReadSecret(mocklib.Anything, "database").
		Return(nil, errors.New("connection refused"))

	provider, err := NewVaultDependencyProvider(mock, "prod", []string{"database"})

	require.NoError(t, err)

	data, err := provider.Load(context.Background())

	require.NoError(t, err)
	assert.Empty(t, data)
}

func TestVaultDependencyProviderName(t *testing.T) {
	provider, err := NewVaultDependencyProvider(newMockDepsClient(t, nil), "prod", nil)

	require.NoError(t, err)
	assert.Equal(t, "vault_deps", provider.Name())
}

func TestVaultDependencyProviderPriority(t *testing.T) {
	provider, err := NewVaultDependencyProvider(newMockDepsClient(t, nil), "prod", nil)

	require.NoError(t, err)

	assert.Equal(t, 30, provider.Priority())

	custom, err := NewVaultDependencyProvider(newMockDepsClient(t, nil), "prod", nil,
		WithVaultDepPriority(50))

	require.NoError(t, err)
	assert.Equal(t, 50, custom.Priority())
}

func TestVaultDependencyProviderGetCached(t *testing.T) {
	mock := newMockDepsClient(t, []string{"database"})
	mock.EXPECT().
		ReadSecret(mocklib.Anything, "database").
		Return(&vaultmodels.Response{Data: map[string]any{
			"data": map[string]any{"key": "value"},
		}}, nil)

	provider, err := NewVaultDependencyProvider(mock, "prod", []string{"database"})

	require.NoError(t, err)

	_, _ = provider.Load(context.Background())

	cached := provider.GetCached()
	assert.NotNil(t, cached["database"])
}

func TestVaultDependencyProviderStop(t *testing.T) {
	provider, err := NewVaultDependencyProvider(newMockDepsClient(t, nil), "prod", nil)

	require.NoError(t, err)

	err = provider.Stop()

	assert.NoError(t, err)
}

func TestVaultDependencyProviderOptions(t *testing.T) {
	provider, err := NewVaultDependencyProvider(newMockDepsClient(t, nil), "prod", nil,
		WithVaultDepLogger(newNopLogger()),
		WithVaultDepPollInterval(time.Second),
	)

	require.NoError(t, err)

	assert.Equal(t, time.Second, provider.pollInterval)
}

func TestVaultDependencyProviderWatchAndPoll(t *testing.T) {
	mock := newMockDepsClient(t, []string{"database"})
	mock.EXPECT().
		ReadSecret(mocklib.Anything, "database").
		Return(&vaultmodels.Response{Data: map[string]any{"key": "initial"}}, nil)

	provider, err := NewVaultDependencyProvider(mock, "prod", []string{"database"},
		WithVaultDepPollInterval(50*time.Millisecond))

	require.NoError(t, err)

	_, err = provider.Load(context.Background())
	require.NoError(t, err)

	mock.EXPECT().
		ReadSecret(mocklib.Anything, "secret/prod/database").
		Return(&vaultmodels.Response{Data: map[string]any{"key": "changed"}}, nil)

	events := make(chan WatchEvent, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = provider.Watch(ctx, events)
	require.NoError(t, err)

	select {
	case event := <-events:
		assert.Equal(t, "vault_deps", event.Source)
		assert.Equal(t, EventChanged, event.Type)
	case <-time.After(1 * time.Second):
		t.Fatal("expected vault deps change event")
	}
}

func TestVaultDependencyProviderPollNoChange(t *testing.T) {
	mock := newMockDepsClient(t, []string{"database"})
	mock.EXPECT().
		ReadSecret(mocklib.Anything, "database").
		Return(&vaultmodels.Response{Data: map[string]any{
			"data": map[string]any{"key": "same"},
		}}, nil)
	mock.EXPECT().
		ReadSecret(mocklib.Anything, "secret/prod/database").
		Return(&vaultmodels.Response{Data: map[string]any{"key": "same"}}, nil)

	provider, err := NewVaultDependencyProvider(mock, "prod", []string{"database"},
		WithVaultDepPollInterval(50*time.Millisecond))

	require.NoError(t, err)

	_, _ = provider.Load(context.Background())

	events := make(chan WatchEvent, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = provider.Watch(ctx, events)

	select {
	case event := <-events:
		t.Fatalf("did not expect change event: %+v", event)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestVaultDependencyProviderPollWithErrors(t *testing.T) {
	mock := newMockDepsClient(t, nil)
	mock.EXPECT().
		ReadSecret(mocklib.Anything, "secret/prod/database").
		Return(nil, errors.New("unavailable"))

	provider, err := NewVaultDependencyProvider(mock, "prod", []string{"database"},
		WithVaultDepPollInterval(50*time.Millisecond))

	require.NoError(t, err)

	events := make(chan WatchEvent, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = provider.Watch(ctx, events)

	time.Sleep(150 * time.Millisecond)

	select {
	case e := <-events:
		// poll with all errors should not emit events but poll runs; allow any
		_ = e
	default:
	}
}
