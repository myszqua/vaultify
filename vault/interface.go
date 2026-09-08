package vault

import (
	"context"

	"github.com/myszqua/vaultify/vault/models"
)

// Client is the application-facing Vault client interface used by vaultify.
type Client interface {
	ReadSecret(ctx context.Context, path string) (*models.Response, error)
	ListSecrets(ctx context.Context) (*models.Response, error)
	HealthCheck(ctx context.Context) error
	AppRoleLogin(ctx context.Context) error
	K8SLogin(ctx context.Context) error
}
