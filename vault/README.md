# Vault Client

Wrapper around the HashiCorp Vault Go client ([`vault-client-go`](https://pkg.go.dev/github.com/hashicorp/vault-client-go)) that exposes a small, application-facing API for reading secrets, listing secrets, checking health and authenticating via AppRole or Kubernetes.

## Configuration

The client is configured with a `config.Client` struct:

| Field | YAML | Default | Description |
| --- | --- | --- | --- |
| `Address` | `address` | — | Vault server URL (e.g. `https://vault.example.com:8200`). Required, must be a valid URL. |
| `Timeout` | `timeout` | `10s` | Per-request timeout. Minimum `1s`. |
| `RetryBackoffMin` | `retry_backoff` | `100ms` | Minimum retry wait when Vault returns a retryable error. |
| `RetryBackoffMax` | `retry_backoff` | `400ms` | Maximum retry wait when Vault returns a retryable error. |

Example YAML:

```yaml
address: "https://vault.example.com:8200"
timeout: "10s"
retry_backoff: "100ms"
```

## Environment variables

Authentication reads credentials from environment variables:

| Constant | Env var | Used by |
| --- | --- | --- |
| `ENVAppLoginRoleID` | `APP_ENV_ROLE_ID` | `AppRoleLogin` |
| `ENVAppLoginRoleSecretID` | `APP_ENV_ROLE_SECRET_ID` | `AppRoleLogin` |
| `ENVK8sJWT` | `APP_ENV_K8S_JWT_TOKEN` | `K8SLogin` |
| `ENVK8sRoleName` | `APP_ENV_K8S_ROLE_NAME` | `K8SLogin` |
| `ENVMountPath` | `APP_ENV_VAULT_MOUNT_PATH` | `AppRoleLogin`, `K8SLogin` |

## Usage

```go
package main

import (
	"context"
	"os"

	"github.com/myszqua/vaultify/vault"
	"github.com/myszqua/vaultify/config/vault/config"
	"github.com/myszqua/vaultify/config/vault/models"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()
	logger := zap.NewNop()

	// Build the vault.Client abstraction. A Sender implements the vault.Client interface.
	client, err := vault.New(&config.Client{
		Address:         os.Getenv("VAULT_ADDR"),
		Timeout:         10 * time.Second,
		RetryBackoffMin: 100 * time.Millisecond,
		RetryBackoffMax: 400 * time.Millisecond,
	}, logger)
	if err != nil {
		panic(err)
	}

	// Authenticate using AppRole credentials.
	// mount path + role/secret IDs are read from the environment
	// (APP_ENV_VAULT_MOUNT_PATH, APP_ENV_ROLE_ID, APP_ENV_ROLE_SECRET_ID).
	if err := client.AppRoleLogin(ctx); err != nil {
		panic(err)
	}

	// Optionally verify Vault is reachable.
	if err := client.HealthCheck(ctx); err != nil {
		panic(err)
	}

	// Read a secret from the given path.
	secret, err := client.ReadSecret(ctx, "secret/data/my-app")
	if err != nil {
		panic(err)
	}

	username, _ := secret.Data["username"].(string)
	password, _ := secret.Data["password"].(string)
	println(username, password)
}
```

### Reading a secret

`ReadSecret(ctx, path)` returns a normalized `models.Response`:

```go
resp, err := client.ReadSecret(ctx, "secret/data/my-app")
if err != nil {
	// handle error
}

resp.Data          // map[string]interface{} with the secret values
resp.Metadata      // request id + warnings
resp.LeaseID       // Vault lease id, if any
resp.LeaseDuration // seconds before the lease expires
resp.Renewable     // whether the secret is renewable
```

### Listing secrets

`ListSecrets(ctx, path)` lists keys under a path. The keys are available under `resp.Data["keys"]`:

```go
resp, err := client.ListSecrets(ctx, "metadata/my-app")
if err != nil {
	// handle error
}

keys := resp.Data["keys"].([]interface{})
```

### Health check

`HealthCheck(ctx)` returns `nil` when Vault is reachable, an error otherwise:

```go
if err := client.HealthCheck(ctx); err != nil {
	// Vault is unavailable
}
```

### Kubernetes auth

For clusters using the Kubernetes auth method, swap `AppRoleLogin` for `K8SLogin`.
The JWT and role name are read from `APP_ENV_K8S_JWT_TOKEN` and `APP_ENV_K8S_ROLE_NAME`.

```go
if err := client.K8SLogin(ctx); err != nil {
	// handle error
}
```

## Interface

`vault.Sender` implements the `vault.Client` interface, so you can depend on the interface in your own code:

```go
func Setup(ctx context.Context, client vault.Client) {
	if err := client.AppRoleLogin(ctx); err != nil {
		// handle error
	}
}
```

The `vault.Logger` interface (`Info`, `Warn`, `Debug`) lets you pass your own logger (e.g. a `*zap.Logger` wrapper) into `New`.