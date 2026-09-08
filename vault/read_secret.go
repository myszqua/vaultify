package vault

import (
	"context"
	"fmt"

	"github.com/myszqua/vaultify/vault/models"
)

// ReadSecret reads a secret under the configured data mount path.
func (s *Sender) ReadSecret(ctx context.Context, path string) (*models.Response, error) {
	resp, err := s.client.Read(ctx, fmt.Sprintf("%s/%s", s.dataMountPath, path))
	if err != nil {
		return nil, err
	}

	return models.Convert(resp), nil
}
