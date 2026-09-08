package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/myszqua/vaultify/vault"
	"github.com/myszqua/vaultify/vault/config"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type Resp struct {
	Val map[string]any `json:"value"`
}

func main() {
	ctx := context.Background()

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	cfg, err := loadConfig("config.yaml")
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	// vault.New loads the PFX client certificate and CA from the following
	// environment variables (read via config methods):
	//   APP_ENV_VAULT_CERT_PATH     – path to the .pfx certificate
	//   APP_ENV_VAULT_CERT_PASSWORD – password for the .pfx certificate
	//   APP_ENV_VAULT_CA_PATH       – path to the CA certificate (PEM)
	client, err := vault.New(cfg, &zapLogger{logger: logger})
	if err != nil {
		logger.Fatal("failed to create vault client", zap.Error(err))
	}

	if err := run(ctx, client, logger); err != nil {
		logger.Fatal("vault operation failed", zap.Error(err))
	}
}

func run(ctx context.Context, client vault.Client, logger *zap.Logger) error {
	if err := client.HealthCheck(ctx); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	logger.Info("vault is healthy")

	// Authenticate via AppRole.
	// Requires: APP_ENV_VAULT_MOUNT_PATH, APP_ENV_ROLE_ID, APP_ENV_ROLE_SECRET_ID.
	if err := client.AppRoleLogin(ctx); err != nil {
		return fmt.Errorf("approle login failed: %w", err)
	}
	logger.Info("authenticated via approle")

	listResp, err := client.ListSecrets(ctx)
	if err != nil {
		return fmt.Errorf("list secrets failed: %w", err)
	}

	fmt.Printf("available keys: %v\n", listResp)

	if keys, ok := listResp.Data["keys"].([]interface{}); ok {
		for _, key := range keys {
			fmt.Printf("key: %v\n", key)

			readResp, err := client.ReadSecret(ctx, key.(string))
			if err != nil {
				return fmt.Errorf("read secret failed: %w", err)
			}

			fmt.Printf("secret: %v\n", readResp.Data["data"])
		}

	}

	return nil
}

func loadConfig(path string) (*config.Client, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &config.Client{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if cfg.Address == "" {
		cfg.Address = os.Getenv("VAULT_ADDR")
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.RetryBackoffMin == 0 {
		cfg.RetryBackoffMin = 100 * time.Millisecond
	}
	if cfg.RetryBackoffMax == 0 {
		cfg.RetryBackoffMax = 400 * time.Millisecond
	}

	return cfg, nil
}

type zapLogger struct {
	logger *zap.Logger
}

func (l *zapLogger) Info(args ...any)  { l.logger.Info(fmt.Sprint(args...)) }
func (l *zapLogger) Warn(args ...any)  { l.logger.Warn(fmt.Sprint(args...)) }
func (l *zapLogger) Debug(args ...any) { l.logger.Debug(fmt.Sprint(args...)) }
