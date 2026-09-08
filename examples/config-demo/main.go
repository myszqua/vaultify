package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	pipeconfig "github.com/myszqua/vaultify"
	"github.com/myszqua/vaultify/models"
	"github.com/myszqua/vaultify/pipe"
	"github.com/myszqua/vaultify/vault"
	vaultconfig "github.com/myszqua/vaultify/vault/config"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	// Vault client configuration (address, mTLS cert, approle...).
	vaultCfg, err := loadVaultConfig("vault.yaml")
	if err != nil {
		logger.Fatal("failed to load vault config", zap.Error(err))
	}

	vaultClient, err := vault.New(vaultCfg, logger.Sugar())
	if err != nil {
		logger.Fatal("failed to create vault client", zap.Error(err))
	}

	if err := run(ctx, vaultClient, logger); err != nil {
		logger.Fatal("config load failed", zap.Error(err))
	}
}

func run(ctx context.Context, vaultClient vault.Client, logger *zap.Logger) error {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}

	depLogger := logger

	// Seamless dependency loading: declare the components you need once,
	// and the loader fetches each one from Vault at secret/{env}/{dep}.
	deps := []string{
		string(models.DependencyDatabase),
		string(models.DependencyRedis),
		string(models.DependencyS3),
		string(models.DependencyServer),
	}

	// 1. Local non-secret config from config.yaml.
	fileProvider := pipeconfig.NewFileProvider([]string{"config.yaml"})

	// 2. Dependency secrets from Vault.
	vaultDeps, err := pipeconfig.NewVaultDependencyProvider(
		vaultClient,
		env,
		deps,
		pipeconfig.WithVaultDepPriority(30),
		pipeconfig.WithVaultDepPollInterval(30*time.Second), //nolint:mnd
		pipeconfig.WithVaultDepLogger(depLogger.Sugar()),
	)
	if err != nil {
		return err
	}

	loader := pipeconfig.New(
		fileProvider, vaultDeps,
	).WithLogger(depLogger.Sugar())

	if err := loader.Load(ctx); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Validate that all declared dependencies actually landed in the merged
	// config, then present the raw values for each one.
	if err := loader.ValidateDependencies(ctx, deps); err != nil {
		return fmt.Errorf("load dependencies: %w", err)
	}

	// 3. Unmarshal each dependency into a standardized, typed config struct.
	var db pipe.DatabaseConfig
	if err := loader.UnmarshalKey(string(models.DependencyDatabase), &db); err != nil {
		return fmt.Errorf("unmarshal database: %w", err)
	}

	var redis pipe.RedisConfig
	if err := loader.UnmarshalKey(string(models.DependencyRedis), &redis); err != nil {
		return fmt.Errorf("unmarshal redis: %w", err)
	}

	var s3 pipe.S3Config
	if err := loader.UnmarshalKey(string(models.DependencyS3), &s3); err != nil {
		return fmt.Errorf("unmarshal s3: %w", err)
	}

	var server pipe.ServerConfig
	if err := loader.UnmarshalKey(string(models.DependencyServer), &server); err != nil {
		return fmt.Errorf("unmarshal server: %w", err)
	}

	var testStruct struct {
		Name1 string `yaml:"name1"`
		Name2 string `yaml:"name2"`
		Name3 string `yaml:"name3"`
		Name4 string `yaml:"name4"`
	}

	if err := loader.UnmarshalKey("test_struct", &testStruct); err != nil {
		return fmt.Errorf("unmarshal server: %w", err)
	}

	logger.Info("database",
		zap.String("dsn", db.DSN()),
		zap.Int("max_open_conns", db.MaxOpenConns),
	)
	logger.Info("redis",
		zap.Strings("nodes", redis.Nodes),
		zap.Int("db", redis.DB),
		zap.Int("pool_size", redis.PoolSize),
	)
	logger.Info("s3",
		zap.String("endpoint", s3.Endpoint),
		zap.String("bucket", s3.Bucket),
	)
	logger.Info("server",
		zap.String("service_name", server.ServiceName),
		zap.Int("port", server.Port),
		zap.String("mode", server.Mode),
	)

	// 4. Print the full merged config, with secrets masked.
	all := pipeconfig.MaskSensitive(loader.AllSettings())

	for k, v := range all {
		logger.Info("config", zap.String("key", k), zap.Any("value", v))
	}

	// 5. Watch for changes: FileProvider fsnotify + Vault poll, with reload
	//    and notify callbacks.
	watchMgr := pipeconfig.NewWatchManager(loader, depLogger.Sugar())
	watchMgr.OnChange(func(event pipeconfig.WatchEvent) {
		logger.Info("config changed",
			zap.String("source", event.Source),
			zap.String("type", event.Type.String()),
		)
	})

	if err := watchMgr.Start(ctx); err != nil {
		return fmt.Errorf("start watch: %w", err)
	}
	defer watchMgr.Stop()

	// Block until interrupted (Ctrl+C / SIGTERM).
	<-ctx.Done()

	return nil
}

func loadVaultConfig(path string) (*vaultconfig.Client, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &vaultconfig.Client{}
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
