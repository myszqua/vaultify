package vault

import (
	"context"
	"os"

	"github.com/hashicorp/vault-client-go/schema"
	"github.com/myszqua/vaultify/vault/models"
)

// AppRoleLogin authenticates to Vault using AppRole role/secret ids from the
// environment and stores the resulting client token.
func (s *Sender) AppRoleLogin(ctx context.Context) error {
	resp, err := s.client.Auth.AppRoleLogin(
		ctx,
		schema.AppRoleLoginRequest{
			RoleId:   os.Getenv(models.ENVAppLoginRoleID),
			SecretId: os.Getenv(models.ENVAppLoginRoleSecretID),
		},
	)
	if err != nil {
		return err
	}

	if err := s.client.SetToken(resp.Auth.ClientToken); err != nil {
		return err
	}

	return nil
}

// K8SLogin authenticates to Vault using a Kubernetes JWT and role name from
// the environment and stores the resulting client token.
func (s *Sender) K8SLogin(ctx context.Context) error {
	resp, err := s.client.Auth.KubernetesLogin(
		ctx,
		schema.KubernetesLoginRequest{
			Jwt:  os.Getenv(models.ENVK8sJWT),
			Role: os.Getenv(models.ENVK8sRoleName),
		},
	)
	if err != nil {
		return err
	}

	if err := s.client.SetToken(resp.Auth.ClientToken); err != nil {
		return err
	}

	return nil
}
