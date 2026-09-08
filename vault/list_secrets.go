package vault

import (
	"context"
	"fmt"

	"github.com/myszqua/vaultify/vault/models"
)

// ListSecrets lists secret keys under the configured metadata mount path.
func (s *Sender) ListSecrets(ctx context.Context) (*models.Response, error) {
	resp, err := s.client.List(ctx, s.metaMountPath)
	if err != nil {
		return nil, fmt.Errorf("list secrets failed: %w", err)
	}

	return models.Convert(resp), nil
}
