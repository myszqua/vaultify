package vault

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/myszqua/vaultify/vault/config"
	"github.com/myszqua/vaultify/vault/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockLogger struct{}

func (mockLogger) Info(...any)  {}
func (mockLogger) Warn(...any)  {}
func (mockLogger) Debug(...any) {}

func testConfig(address string) *config.Client {
	return &config.Client{
		Address:         address,
		Timeout:         5 * time.Second,
		RetryBackoffMin: time.Millisecond,
		RetryBackoffMax: 10 * time.Millisecond,
	}
}

func newTestSender(t *testing.T, server *httptest.Server) *Sender {
	t.Helper()

	s, err := New(testConfig(server.URL), mockLogger{})
	require.NoError(t, err)

	return s
}

func TestNew(t *testing.T) {
	t.Run("Valid config", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		s, err := New(testConfig(server.URL), mockLogger{})
		require.NoError(t, err)
		require.NotNil(t, s)
		assert.NotNil(t, s.client)
		assert.Equal(t, server.URL, s.cfg.Address)
	})

	t.Run("Invalid address", func(t *testing.T) {
		s, err := New(testConfig("://bad-address"), mockLogger{})
		require.Error(t, err)
		assert.Nil(t, s)
	})
}

func TestSender_ReadSecret(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		t.Setenv(models.ENVMountPath, "myvault")
		t.Setenv(models.ENVApp, "app1")
		t.Setenv(models.ENVServiceACLName, "svc1")

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/v1/myvault/data/app1/svc1/secret/data/app", r.URL.Path)

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"request_id": "req-1",
				"lease_id": "lease-1",
				"lease_duration": 3600,
				"renewable": true,
				"data": {"username": "admin", "password": "s3cr3t"},
				"warnings": ["w1"]
			}`))
		}))
		defer server.Close()

		s := newTestSender(t, server)

		resp, err := s.ReadSecret(context.Background(), "secret/data/app")
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.Equal(t, map[string]interface{}{"username": "admin", "password": "s3cr3t"}, resp.Data)
		require.NotNil(t, resp.Metadata)
		assert.Equal(t, "req-1", resp.Metadata.RequestID)
		assert.Equal(t, []string{"w1"}, resp.Metadata.Warnings)
		assert.Equal(t, "lease-1", resp.LeaseID)
		assert.Equal(t, 3600, resp.LeaseDuration)
		assert.True(t, resp.Renewable)
	})

	t.Run("Client error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		s := newTestSender(t, server)

		resp, err := s.ReadSecret(context.Background(), "secret/data/app")

		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("Malformed response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`not-json`))
		}))
		defer server.Close()

		s := newTestSender(t, server)

		resp, err := s.ReadSecret(context.Background(), "secret/data/app")

		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestSender_ListSecrets(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		t.Setenv(models.ENVMountPath, "myvault")
		t.Setenv(models.ENVApp, "app1")
		t.Setenv(models.ENVServiceACLName, "svc1")

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/v1/myvault/metadata/app1/svc1", r.URL.Path)

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"request_id": "req-list",
				"data": {"keys": ["app1", "app2"]},
				"lease_duration": 0,
				"renewable": false
			}`))
		}))
		defer server.Close()

		s := newTestSender(t, server)

		resp, err := s.ListSecrets(context.Background())
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.Equal(t, map[string]interface{}{"keys": []interface{}{"app1", "app2"}}, resp.Data)
		assert.Equal(t, "req-list", resp.Metadata.RequestID)
	})

	t.Run("Client error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		}))
		defer server.Close()

		s := newTestSender(t, server)

		resp, err := s.ListSecrets(context.Background())

		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestSender_HealthCheck(t *testing.T) {
	t.Run("Healthy", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/v1/sys/health", r.URL.Path)

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"initialized": true, "sealed": false}`))
		}))
		defer server.Close()

		s := newTestSender(t, server)

		err := s.HealthCheck(context.Background())

		assert.NoError(t, err)
	})

	t.Run("Unhealthy", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`not-json`))
		}))
		defer server.Close()

		s := newTestSender(t, server)

		err := s.HealthCheck(context.Background())

		assert.Error(t, err)
	})
}

func TestSender_AppRoleLogin(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.True(t, strings.HasPrefix(r.URL.Path, "/v1/auth/approle/login"))

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"data": {},
				"auth": {"client_token": "s.test-token"}
			}`))
		}))
		defer server.Close()

		t.Setenv(models.ENVMountPath, "approle")
		t.Setenv(models.ENVAppLoginRoleID, "role-id")
		t.Setenv(models.ENVAppLoginRoleSecretID, "secret-id")

		s := newTestSender(t, server)

		err := s.AppRoleLogin(context.Background())

		assert.NoError(t, err)
	})

	t.Run("Login request error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		t.Setenv(models.ENVMountPath, "approle")
		t.Setenv(models.ENVAppLoginRoleID, "role-id")
		t.Setenv(models.ENVAppLoginRoleSecretID, "secret-id")

		s := newTestSender(t, server)

		err := s.AppRoleLogin(context.Background())

		assert.Error(t, err)
	})

	t.Run("SetToken error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"data": {},
				"auth": {"client_token": "bad\u0001token"}
			}`))
		}))
		defer server.Close()

		t.Setenv(models.ENVMountPath, "approle")
		t.Setenv(models.ENVAppLoginRoleID, "role-id")
		t.Setenv(models.ENVAppLoginRoleSecretID, "secret-id")

		s := newTestSender(t, server)

		err := s.AppRoleLogin(context.Background())

		assert.Error(t, err)
	})
}

func TestSender_K8SLogin(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.True(t, strings.HasPrefix(r.URL.Path, "/v1/auth/kubernetes/login"))

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"data": {},
				"auth": {"client_token": "s.k8s-token"}
			}`))
		}))
		defer server.Close()

		t.Setenv(models.ENVMountPath, "kubernetes")
		t.Setenv(models.ENVK8sJWT, "jwt-token")
		t.Setenv(models.ENVK8sRoleName, "role-name")

		s := newTestSender(t, server)

		err := s.K8SLogin(context.Background())

		assert.NoError(t, err)
	})

	t.Run("Login request error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer server.Close()

		t.Setenv(models.ENVMountPath, "kubernetes")
		t.Setenv(models.ENVK8sJWT, "jwt-token")
		t.Setenv(models.ENVK8sRoleName, "role-name")

		s := newTestSender(t, server)

		err := s.K8SLogin(context.Background())

		assert.Error(t, err)
	})

	t.Run("SetToken error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"data": {},
				"auth": {"client_token": "bad\u0001token"}
			}`))
		}))
		defer server.Close()

		t.Setenv(models.ENVMountPath, "kubernetes")
		t.Setenv(models.ENVK8sJWT, "jwt-token")
		t.Setenv(models.ENVK8sRoleName, "role-name")

		s := newTestSender(t, server)

		err := s.K8SLogin(context.Background())

		assert.Error(t, err)
	})
}

func TestSenderImplementsClientInterface(_ *testing.T) {
	var _ Client = (*Sender)(nil)
}
