package vaultify_test

import (
	"context"
	"testing"

	client "github.com/myszqua/vaultify"
	"github.com/myszqua/vaultify/mocks"
	"github.com/myszqua/vaultify/models"
	"github.com/myszqua/vaultify/pipe"
	vaultmodels "github.com/myszqua/vaultify/vault/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newMockDependencyClient(t *testing.T) *mocks.MockClient {
	t.Helper()

	m := mocks.NewMockClient(t)
	m.EXPECT().AppRoleLogin(mock.Anything).Return(nil)

	return m
}

func TestSeamlessDependencyLoading(t *testing.T) {
	m := newMockDependencyClient(t)
	m.EXPECT().ListSecrets(mock.Anything).Return(&vaultmodels.Response{
		Data: map[string]any{
			"keys": []interface{}{string(models.DependencyDatabase), string(models.DependencyRedis)},
		},
	}, nil)
	m.EXPECT().
		ReadSecret(mock.Anything, string(models.DependencyDatabase)).
		Return(&vaultmodels.Response{Data: map[string]any{
			"data": map[string]any{
				"host":     "db.example.com",
				"port":     5432,
				"user":     "app",
				"password": "s3cr3t",
				"name":     "myapp",
			},
		}}, nil)
	m.EXPECT().
		ReadSecret(mock.Anything, string(models.DependencyRedis)).
		Return(&vaultmodels.Response{Data: map[string]any{
			"data": map[string]any{
				"nodes":    []any{"redis.example.com:6379"},
				"password": "redis_pass",
			},
		}}, nil)

	provider, err := client.NewVaultDependencyProvider(m, "prod",
		[]string{string(models.DependencyDatabase), string(models.DependencyRedis)})

	require.NoError(t, err)

	loader := client.New(provider)
	err = loader.Load(context.Background())
	require.NoError(t, err)

	deps := []string{
		string(models.DependencyDatabase),
		string(models.DependencyRedis),
	}

	err = loader.LoadDependencies(context.Background(), deps)
	require.NoError(t, err)

	dbConfig, err := loader.GetDependency(string(models.DependencyDatabase))
	require.NoError(t, err)
	assert.Equal(t, "db.example.com", dbConfig["host"])
	assert.Equal(t, 5432, dbConfig["port"])
	assert.Equal(t, "s3cr3t", dbConfig["password"])

	redisConfig, err := loader.GetDependency(string(models.DependencyRedis))
	require.NoError(t, err)
	assert.Equal(t, "redis_pass", redisConfig["password"])
}

func TestSeamlessDependencyUnmarshalConfigs(t *testing.T) {
	m := newMockDependencyClient(t)
	m.EXPECT().ListSecrets(mock.Anything).Return(&vaultmodels.Response{
		Data: map[string]any{
			"keys": []interface{}{string(models.DependencyDatabase)},
		},
	}, nil)
	m.EXPECT().
		ReadSecret(mock.Anything, string(models.DependencyDatabase)).
		Return(&vaultmodels.Response{Data: map[string]any{
			"data": map[string]any{
				"host":     "db.example.com",
				"port":     5432,
				"user":     "app",
				"password": "s3cr3t",
				"name":     "myapp",
				"ssl_mode": "require",
			},
		}}, nil)

	provider, err := client.NewVaultDependencyProvider(m, "prod",
		[]string{string(models.DependencyDatabase)})

	require.NoError(t, err)

	loader := client.New(provider)
	require.NoError(t, loader.Load(context.Background()))

	var db pipe.DatabaseConfig
	err = loader.UnmarshalKey(string(models.DependencyDatabase), &db)
	require.NoError(t, err)

	assert.Equal(t, "db.example.com", db.Host)
	assert.Equal(t, 5432, db.Port)
	assert.Equal(t, "s3cr3t", db.Password)
	assert.Equal(t, "require", db.SSLMode)
	assert.NotEmpty(t, db.DSN())
}

func TestGetDependencyNotFound(t *testing.T) {
	fileProvider := client.NewFileProvider([]string{"/nonexistent/client.yaml"})
	loader := client.New(fileProvider)
	require.NoError(t, loader.Load(context.Background()))

	_, err := loader.GetDependency("redis")
	assert.Error(t, err)
	assert.ErrorIs(t, err, client.ErrKeyNotFound)
}
