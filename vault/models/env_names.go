package models

const (
	// ENVApp is the app environment variable name.
	ENVApp = "APP_ENV"
	// ENVServiceACLName is the service ACL name variable name.
	ENVServiceACLName = "APP_ENV_SERVICE_ACL_NAME"
	// ENVCertPath is the mTLS certificate path variable name.
	ENVCertPath = "APP_ENV_VAULT_CERT_PATH"
	// ENVCertPassword is the mTLS certificate password variable name.
	ENVCertPassword = "APP_ENV_VAULT_CERT_PASSWORD" //nolint:gosec
	// ENVAppLoginRoleID is the AppRole role id variable name.
	ENVAppLoginRoleID = "APP_ENV_ROLE_ID"
	// ENVAppLoginRoleSecretID is the AppRole secret id variable name.
	ENVAppLoginRoleSecretID = "APP_ENV_ROLE_SECRET_ID" //nolint:gosec
	// ENVK8sJWT is the Kubernetes JWT variable name.
	ENVK8sJWT = "APP_ENV_K8S_JWT_TOKEN"
	// ENVK8sRoleName is the Kubernetes role name variable name.
	ENVK8sRoleName = "APP_ENV_K8S_ROLE_NAME"
	// ENVMountPath is the Vault mount path variable name.
	ENVMountPath = "APP_ENV_VAULT_MOUNT_PATH"
)
