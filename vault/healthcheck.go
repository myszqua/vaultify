package vault

import (
	"context"
)

// HealthCheck verifies that Vault is reachable and reports its health status.
func (s *Sender) HealthCheck(ctx context.Context) error {
	_, err := s.client.System.ReadHealthStatus(ctx)
	if err != nil {
		return err
	}

	return nil
}
