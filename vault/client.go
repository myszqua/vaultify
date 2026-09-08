package vault

import (
	"crypto/tls"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/hashicorp/vault-client-go"
	"github.com/myszqua/vaultify/vault/config"
	logger "github.com/myszqua/vaultify/vault/logger"
	"github.com/myszqua/vaultify/vault/models"
	"github.com/myszqua/vaultify/vault/tripper"
	"golang.org/x/crypto/pkcs12"
)

// Sender is the vaultify Vault client implementation backed by vault-client-go.
type Sender struct {
	logger        logger.Writer
	cfg           *config.Client
	client        *vault.Client
	metaMountPath string
	dataMountPath string
}

// New builds a Sender from the given configuration, loading the optional
// mTLS PFX client certificate from the environment.
func New(cfg *config.Client, logger logger.Writer) (*Sender, error) {
	tlsConfig, err := loadPFXCertificate(
		cfg.GetCertificatePath,
		cfg.GetCertificatePassword,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load pfx certificate: %w", err)
	}

	httpClient := &http.Client{
		Timeout: cfg.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	if cfg.DebugMode {
		httpClient.Transport = tripper.NewLoggingRoundTripper(
			httpClient.Transport,
			true,
			logger,
		)
	}

	vc, err := vault.New(
		vault.WithAddress(cfg.Address),
		vault.WithRequestTimeout(cfg.Timeout),
		vault.WithRetryConfiguration(vault.RetryConfiguration{
			RetryWaitMin: cfg.RetryBackoffMin,
			RetryWaitMax: cfg.RetryBackoffMax,
		}),
		vault.WithHTTPClient(httpClient),
		vault.WithTLS(vault.TLSConfiguration{
			InsecureSkipVerify: cfg.InsecureSkipVerify,
		}),
	)
	if err != nil {
		return nil, err
	}

	return &Sender{
		logger: logger,
		cfg:    cfg,
		client: vc,
		metaMountPath: fmt.Sprintf(
			"%s/metadata/%s/%s",
			os.Getenv(models.ENVMountPath),
			os.Getenv(models.ENVApp),
			os.Getenv(models.ENVServiceACLName),
		),
		dataMountPath: fmt.Sprintf(
			"%s/data/%s/%s",
			os.Getenv(models.ENVMountPath),
			os.Getenv(models.ENVApp),
			os.Getenv(models.ENVServiceACLName),
		),
	}, nil
}

// loadPFXCertificate loads the mTLS client certificate from a PFX file.
func loadPFXCertificate(fnPfxPath, fnPasswd func() (string, error)) (*tls.Config, error) { //nolint:cyclop
	pfxPath, err := fnPfxPath()
	if err != nil {
		if errors.Is(err, models.ErrEmptyENVField) {
			return nil, nil
		}

		return nil, err
	}

	password, err := fnPasswd()
	if err != nil {
		if errors.Is(err, models.ErrEmptyENVField) {
			return nil, nil
		}

		return nil, err
	}

	if pfxPath == "" {
		return nil, nil
	}

	pfxData, err := os.ReadFile(pfxPath) //nolint:gosec // path comes from configured env var
	if err != nil {
		return nil, fmt.Errorf("failed to read pfx file: %w", err)
	}

	blocks, err := pkcs12.ToPEM(pfxData, password)
	if err != nil {
		return nil, fmt.Errorf("failed to convert pfx to pem: %w", err)
	}

	var certPEM, keyPEM []byte

	for _, block := range blocks {
		if block.Type == "CERTIFICATE" {
			certPEM = append(certPEM, pem.EncodeToMemory(block)...)
		} else if block.Type == "PRIVATE KEY" {
			keyPEM = append(keyPEM, pem.EncodeToMemory(block)...)
		}
	}

	if len(certPEM) == 0 || len(keyPEM) == 0 {
		return nil, fmt.Errorf("no certificate or key found in pfx")
	}

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to load key pair: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}
