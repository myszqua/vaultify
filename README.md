# vaultify

**Centralized Configuration & Dependency Manager for Go microservices.**

vaultify is a lightweight, high-performance config server that acts as a single source of truth for all your services. It standardizes how applications fetch and manage configuration, secrets, and external dependencies.

## Key Features

- **Secure Secret Storage**: Seamlessly integrates with HashiCorp Vault to store and manage sensitive credentials (database passwords, API keys, etc.).
- **Dependency Management**: Define your service's needs (e.g., `database`, `redis`, `s3`) in a simple config, and vaultify automatically injects the required connection strings and credentials.
- **Flexible Configuration**: Load settings from multiple sources (YAML files, Environment Variables) with a clear priority order.
- **Hot-Reload**: Update configuration and secrets on the fly without restarting your services — zero downtime configuration changes.
- **Unified gRPC API**: Exposes a single, standardized gRPC interface for any service to fetch its complete configuration in one request, eliminating the need for per-service integration code.
- **Clustered & Scalable**: Designed to run in a clustered mode, providing high availability and horizontal scalability for large microservice ecosystems.
