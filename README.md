# Vaultify

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue)](#)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Tests](https://img.shields.io/badge/tests-passing-brightgreen)](#)

Embeddable configuration client for Go services. Loads, standardizes, and parses configs from multiple providers (files, env vars, HashiCorp Vault) with priority-based merging, hot reload, and ready-made typed config structs.

## Features

- **Multi-provider merging** -- File (YAML/JSON/TOML), Environment Variables, HashiCorp Vault, Defaults -- all merged with configurable priority
- **Dependency-based Vault loading** -- declare `database`, `redis`, `s3`, ... once, and the loader fetches each from `{env}/{dep}`
- **Hot reload** -- file watching via fsnotify, Vault polling with change detection, SIGHUP signal support
- **Standardized config structs** -- ready-made `pipe.DatabaseConfig`, `pipe.RedisConfig`, `pipe.S3Config`, etc. with validation tags
- **Dynamic value resolution** -- `${ENV_VAR}`, `${VAULT:path:key}`, `${FILE:path}` placeholders
- **Validation** -- `config:"required"`, `min`, `max`, `pattern` struct tags
- **Sensitive field masking** -- passwords, tokens, secrets are auto-masked in logs
- **Graceful degradation** -- partial Vault failures skip unavailable deps; total outage falls back to cache

## Installation

```bash
go get github.com/myszqua/vaultify
```

## Quick Start

```go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"

    vaultify "github.com/myszqua/vaultify"
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

    logger, _ := zap.NewProduction()
    defer logger.Sync()

    // 1. Create vault client
    vaultCfg := &vaultconfig.Client{Address: os.Getenv("VAULT_ADDR")} // or straight form yaml
    vaultClient, _ := vault.New(vaultCfg, logger.Sugar())
    vaultClient.AppRoleLogin(ctx)

    // 2. Declare dependencies
    deps := []string{
        string(models.DependencyDatabase),
        string(models.DependencyRedis),
        string(models.DependencyS3),
        string(models.DependencyServer),
    }

    // 3. Build providers
    fileProvider := vaultify.NewFileProvider([]string{"config.yaml"})
    vaultDeps, _ := vaultify.NewVaultDependencyProvider(vaultClient, "prod", deps)

    // 4. Load and merge
    loader := vaultify.New(fileProvider, vaultDeps).WithLogger(logger.Sugar())
    loader.Load(ctx)
    loader.LoadDependencies(ctx, deps)

    // 5. Unmarshal into typed structs
    var db pipe.DatabaseConfig
    loader.UnmarshalKey(string(models.DependencyDatabase), &db)
    db.DSN() // host=... port=... user=... password=... dbname=... sslmode=...

    var redis pipe.RedisConfig
    loader.UnmarshalKey(string(models.DependencyRedis), &redis)

    // 6. Hot reload
    watchMgr := vaultify.NewWatchManager(loader, logger.Sugar())
    watchMgr.OnChange(func(e vaultify.WatchEvent) {
        logger.Info("config changed", zap.String("source", e.Source))
    })
    watchMgr.Start(ctx)
    defer watchMgr.Stop()

    <-ctx.Done()
}
```

## Providers

Every config source implements the `Provider` interface:

```go
type Provider interface {
    Name() string
    Load(ctx context.Context) (map[string]any, error)
    Priority() int
    Watch(ctx context.Context, events chan<- WatchEvent) error
    Stop() error
}
```

### Priority order

```
Default (10) < File (20) < Vault (30) < Env (40)
```

Higher priority overrides lower during merge.

### FileProvider

Loads YAML/JSON/TOML files. Multiple files layer on top of each other.

```go
provider := vaultify.NewFileProvider(
    []string{"config.base.yaml", "config.local.yaml"},
    vaultify.WithFilePriority(20),
)
```

### EnvProvider

Loads env vars with automatic prefix filtering and type casting.

```go
// APP_SERVER_HOST=localhost → server.host = "localhost"
provider := vaultify.NewEnvProvider("APP_", vaultify.WithEnvPriority(40))
```

### VaultProvider

Loads a single Vault secret at `secret/{env}/{service}` with poll-based change detection and cache fallback.

```go
provider := vaultify.NewVaultProvider(
    vaultClient,
    "prod",
    "myservice",
    vaultify.WithVaultPriority(30),
    vaultify.WithVaultPollInterval(30*time.Second),
)
```

### VaultDependencyProvider

Loads multiple dependency secrets from Vault at `secret/{env}/{dependency}`. Authenticates on construction, lists available secrets, reads each declared dependency, and polls for changes. Partial failures are skipped gracefully.

```go
provider, _ := vaultify.NewVaultDependencyProvider(
    vaultClient,
    "prod",
    []string{"database", "redis", "s3", "server"},
    vaultify.WithVaultDepPriority(30),
    vaultify.WithVaultDepPollInterval(30*time.Second),
    vaultify.WithVaultDepLogger(logger.Sugar()),
)
```

### DefaultProvider

Provides fallback values from a map or struct tag extraction.

```go
provider := vaultify.NewDefaultProvider(map[string]any{
    "server": map[string]any{"host": "localhost", "port": 8080},
})
```

## Standardized Config Structs (`pipe`)

Ready-made structs with `yaml`/`json` tags and validation. Unmarshal directly from Vault secrets or config files.

| Struct | Dependency | Key Fields |
|--------|------------|------------|
| `pipe.DatabaseConfig` | `database` | host, port, user, password, name, ssl_mode, max_open_conns, `DSN()` |
| `pipe.RedisConfig` | `redis` | cluster_mode, nodes, password, db, pool_size, timeouts |
| `pipe.S3Config` | `s3` | endpoint, region, access_key_id, secret_access_key, bucket, permissions |
| `pipe.ServerConfig` | `server` | port, base_url, mode, timeout, service_name, router_version |
| `pipe.LoggerConfig` | `logger` | level, encoding, output_paths, initial_fields |
| `pipe.TracerConfig` | `tracer` | service_name, sampler_type, reporter_host/port |
| `pipe.RateLimiterConfig` | `rate_limiter` | rules, blacklist |
| `pipe.CircuitBreakerConfig` | `circuit_breaker` | timeout, failure_threshold, success_threshold |

### Usage

```go
var db pipe.DatabaseConfig
if err := loader.UnmarshalKey("database", &db); err != nil {
    log.Fatal(err)
}

dsn := db.DSN()
// host=db.example.com port=5432 user=admin password=... dbname=mydb sslmode=disable

maxConns := db.MaxOpenConns // 25 (default)
```

## Custom Structs & Dependencies

You are not limited to the built-in `pipe` structs. Any struct with `yaml`/`json` tags works with `UnmarshalKey` -- define your own config shapes and unmarshal arbitrary Vault secrets or config subtrees into them.

```go
type MyServiceConfig struct {
    FeatureFlags map[string]bool `yaml:"feature_flags"`
    RateLimit    int             `yaml:"rate_limit" min:"1"`
    CustomField  string          `yaml:"custom_field" config:"required"`
}

var cfg MyServiceConfig
if err := loader.UnmarshalKey("my_custom_key", &cfg); err != nil {
    log.Fatal(err)
}
```

### Custom dependencies

Any key present in the merged config can be treated as a dependency -- it does not have to be one of the predefined `models.Dependency*` constants. Declare your own dependency names and unmarshal them:

```go
deps := []string{
    "database",        // built-in
    "my_custom_deps",  // your own Vault secret at secret/{env}/my_custom_deps
    "feature_toggles", // another custom dependency
}

loader.LoadDependencies(ctx, deps)

var custom MyServiceConfig
loader.UnmarshalKey("my_custom_deps", &custom)

var toggles map[string]bool
loader.UnmarshalKey("feature_toggles", &toggles)
```

The same works for inline structs -- any nested map in the config can be unmarshaled directly:

```go
var testStruct struct {
    Name1 string `yaml:"name1"`
    Name2 string `yaml:"name2"`
    Name3 string `yaml:"name3"`
    Name4 string `yaml:"name4"`
}

if err := loader.UnmarshalKey("test_struct", &testStruct); err != nil {
    log.Fatal(err)
}
```

## Hot Reload

Three mechanisms, coordinated by `WatchManager`:

### File watching (fsnotify)

```go
watchMgr := vaultify.NewWatchManager(loader, logger)
watchMgr.OnChange(func(event vaultify.WatchEvent) {
    log.Printf("config changed from %s (type: %s)", event.Source, event.Type)
})
watchMgr.Start(ctx)
```

### Vault polling

`VaultProvider` and `VaultDependencyProvider` poll Vault at a configurable interval and emit events only when data actually changes.

### SIGHUP signal

```go
vaultify.ListenForReload(ctx, loader)
```

### Reload callbacks

```go
loader.OnReload(func() {
    log.Println("config was reloaded, re-initializing services...")
})
```

## Dynamic Values

Resolve `${...}` placeholders in config values:

| Pattern | Source |
|---------|--------|
| `${ENV_VAR}` | `os.Getenv("ENV_VAR")` |
| `${VAULT:path:key}` | Vault secret |
| `${FILE:path}` | File content |

## Validation

Reflection-based struct tag validation:

```go
type Config struct {
    Name string `config:"required" pattern:"^[a-z]+$"`
    Port int    `min:"1" max:"65535"`
}

err := vaultify.Validate(cfg)
```

| Tag | Types | Behavior |
|-----|-------|----------|
| `config:"required"` | all | Zero value check |
| `min:"N"` | int, float64, string | Minimum value or length |
| `max:"N"` | int, float64, string | Maximum value or length |
| `pattern:"regex"` | string | Regex match |

## Sensitive Field Masking

Fields with keys containing `password`, `token`, `secret`, `api_key`, `access_key`, `private_key`, `credential`, `auth` are masked in logs:

```go
masked := vaultify.MaskSensitive(loader.AllSettings())
// "super_secret_123" → "su**********23"
```

## Vault Client

The `vault` package provides a standalone Vault client with mTLS (PFX certificates), AppRole and Kubernetes auth, secret reading, listing, and health checks.

### Authentication

| Method | Env Vars |
|--------|----------|
| **AppRole** | `APP_ENV_ROLE_ID`, `APP_ENV_ROLE_SECRET_ID` |
| **Kubernetes** | `APP_ENV_K8S_JWT_TOKEN`, `APP_ENV_K8S_ROLE_NAME` |

### Connection env vars

| Env Var | Purpose |
|---------|---------|
| `APP_ENV` | Environment (local/dev/stg/prod) |
| `APP_ENV_SERVICE_ACL_NAME` | Service ACL policy name |
| `APP_ENV_VAULT_CERT_PATH` | PFX certificate path |
| `APP_ENV_VAULT_CERT_PASSWORD` | PFX certificate password |
| `APP_ENV_VAULT_MOUNT_PATH` | Vault KV mount path |

### Config YAML

```yaml
address: "https://vault.example.com:8200"
timeout: "10s"
retry_backoff_min: "100ms"
retry_backoff_max: "400ms"
debug_mode: false
```

### Standalone usage

```go
client, _ := vault.New(&vaultconfig.Client{
    Address: os.Getenv("VAULT_ADDR"),
}, logger)

client.AppRoleLogin(ctx)
client.HealthCheck(ctx)

secret, _ := client.ReadSecret(ctx, "secret/data/my-app")
```

See [`vault/README.md`](vault/README.md) for the full Vault client documentation.

## ConfigLoader API

```go
loader := vaultify.New(provider1, provider2, provider3).WithLogger(logger)

loader.Load(ctx)                          // load from all providers
loader.Reload(ctx)                        // reload + trigger callbacks

loader.Get("server.host")                 // any
loader.GetString("server.host")           // string
loader.GetInt("server.port")              // int
loader.GetBool("server.debug")            // bool

loader.Unmarshal(&cfg)                    // full config → struct
loader.UnmarshalKey("database", &db)      // subtree → struct

loader.AllSettings()                      // map[string]any
loader.GetDependency("database")          // map[string]any, error
loader.LoadDependencies(ctx, deps)        // validate deps exist

loader.OnReload(func() { ... })           // register reload callback
```

## Merge Engine

Deep merges nested maps recursively. Slices support two strategies:

```go
loader := vaultify.New(providers...).WithMergeOptions(vaultify.MergeOptions{
    SliceStrategy: vaultify.MergeStrategyMerge,  // or MergeStrategyReplace (default)
    LogConflicts:  true,
})
```

- **MergeStrategyReplace** -- higher-priority slices replace lower-priority entirely
- **MergeStrategyMerge** -- slices are concatenated with deduplication

## Testing

```bash
make test        # run all tests
make lint        # run golangci-lint
make benchmark   # run benchmarks
```

## License

MIT -- see [LICENSE](LICENSE)
