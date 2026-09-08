# Vault Demo

Example integration project demonstrating how to use the `vault` client library.

## Prerequisites

- Go 1.25+
- A running HashiCorp Vault instance with TLS
- AppRole auth enabled on Vault
- A PFX client certificate for mTLS

## Environment Variables

```bash
export APP_ENV_VAULT_CERT_PATH="/path/to/client.pfx"
export APP_ENV_VAULT_CERT_PASSWORD="your-pfx-password"
export APP_ENV="<local/dev/stg/prod/etc>"
export APP_ENV_SERVICE_ACL_NAME="<policy_name>"
export APP_ENV_ROLE_ID="<role-id>"
export APP_ENV_ROLE_SECRET_ID="<role-secret-id>"
export APP_ENV_VAULT_MOUNT_PATH="<your-mount-path>"
```

## Configuration

Edit `config.yaml`:

```yaml
address: "https://vault.example.com:8200"
timeout: "10s"
retry_backoff_min: "100ms"
retry_backoff_max: "400ms"
in_secure_skip_verify: false
```

| Field | Default | Description |
| --- | --- | --- |
| `address` | — | Vault server URL. Required. |
| `timeout` | `10s` | Per-request timeout. |
| `retry_backoff_min` | `100ms` | Minimum retry wait. |
| `retry_backoff_max` | `400ms` | Maximum retry wait. |
| `in_secure_skip_verify` | `false` | Skip TLS certificate verification. |

## Running

```bash
go run main.go
```

## What It Does

1. Loads configuration from `config.yaml` (falls back to `VAULT_ADDR` env var)
2. Creates a Vault client with mTLS using PFX certificate from env vars
3. Checks Vault health via `/v1/sys/health`
4. Authenticates using AppRole credentials from environment variables
5. Reads a secret at `secret/data/my-app`
6. Lists keys under `metadata/my-app`
