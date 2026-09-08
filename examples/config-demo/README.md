# Config Demo

Example integration project demonstrating **seamless dependency-based config loading**
using the `config` library together with a real HashiCorp Vault instance.

## What It Shows

1. Local non-secret config (`config.yaml`) loaded via `FileProvider`.
2. A list of declared dependencies (`database`, `redis`, `s3`, `server`) loaded
   automatically from Vault by `VaultDependencyProvider` at `secret/{env}/{dep}`.
3. `LoadDependencies(ctx, deps)` validating that every declared dependency landed
   in the merged config.
4. `UnmarshalKey(...)` mapping each dependency into a standardized, typed config
   struct from `config/configs` (e.g. `DatabaseConfig`, `RedisConfig`, `S3Config`).
5. Printing the merged config with secrets masked via `MaskSensitive`.
6. Hot reload: `WatchManager` reloads on file changes (`fsnotify`) and Vault polling,
   notifying via an `OnChange` callback.

## Prerequisites

- Go 1.25+
- A running HashiCorp Vault instance with TLS
- AppRole auth enabled on Vault
- A PFX client certificate for mTLS
- Secrets provisioned at `secret/{env}/database`, `secret/{env}/redis`,
  `secret/{env}/s3`, `secret/{env}/server`

## Environment Variables

```bash
# TLS certificates (required)
export APP_ENV_VAULT_CERT_PATH="/path/to/client.pfx"
export APP_ENV_VAULT_CERT_PASSWORD="your-pfx-password"
export APP_ENV="<local/dev/stg/prod/etc>"
export APP_ENV_SERVICE_ACL_NAME="<policy_name>"
export APP_ENV_ROLE_ID="<role-id>"
export APP_ENV_ROLE_SECRET_ID="<role-secret-id>"
export APP_ENV_VAULT_MOUNT_PATH="<your-mount-path>"
```

> `APP_ENV` controls the environment segment of the Vault path. If unset it
> defaults to `dev`.

## Configuration

Edit `vault.yaml`:

```yaml
address: "https://vault.example.com:8200"
timeout: "10s"
retry_backoff_min: "100ms"
retry_backoff_max: "400ms"
insecure_skip_verify: false
debug_mode: true
```

| Field | Default | Description |
| --- | --- | --- |
| `address` | — | Vault server URL. Required (falls back to `VAULT_ADDR`). |
| `timeout` | `10s` | Per-request timeout. |
| `retry_backoff_min` | `100ms` | Minimum retry wait. |
| `retry_backoff_max` | `400ms` | Maximum retry wait. |
| `insecure_skip_verify` | `false` | Skip TLS certificate verification. |
| `debug_mode` | `false` | Log outgoing requests to Vault. |

## Provisioned Secrets

Each dependency is read from `secret/{env}/{dep}`:

| Dependency | Example keys in the Vault secret |
| --- | --- |
| `database` | `host`, `port`, `user`, `password`, `name`, `ssl_mode` |
| `redis` | `nodes` (list), `password`, `db`, `pool_size`, `cluster_mode` |
| `s3` | `endpoint`, `region`, `access_key_id`, `secret_access_key`, `bucket` |
| `server` | `service_name`, `port`, `mode`, `base_url` |

## Running

```bash
go run main.go
```

The program loads config, prints a masked view of everything it found, and then
blocks while watching for changes. Press Ctrl+C to stop.

## Dependency Types

Declare any subset of the predefined dependencies (see `config/models`):

- `database`, `redis`, `s3`
- `rate_limiter`, `circuit_breaker`
- `logger`, `tracer`, `server`

```go
deps := []string{
    string(models.DependencyDatabase),
    string(models.DependencyRedis),
    string(models.DependencyS3),
}
```
