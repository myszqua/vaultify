package vaultify_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	config2 "github.com/myszqua/vaultify"
	"github.com/myszqua/vaultify/mocks"
	vaultmodels "github.com/myszqua/vaultify/vault/models"
	"github.com/stretchr/testify/mock"
)

type exampleConfig struct {
	Server struct {
		Host string `yaml:"host" config:"required"`
		Port int    `yaml:"port" min:"1" max:"65535"`
	} `yaml:"server"`
	Database struct {
		Host     string `yaml:"host" config:"required"`
		Password string `yaml:"password" config:"required"`
	} `yaml:"database"`
}

func Example() {
	dir, _ := os.MkdirTemp("", "config-example")
	defer os.RemoveAll(dir)

	filePath := filepath.Join(dir, "config.yaml")
	_ = os.WriteFile(filePath, []byte(`
server:
  host: localhost
  port: 8080
database:
  host: db.example.com
  password: ""
`), 0o600)

	vaultMock := &mocks.MockClient{}
	vaultMock.On("ReadSecret", mock.Anything, "secret/prod/myservice").
		Return(&vaultmodels.Response{
			Data: map[string]any{
				"server": map[string]any{
					"host": "localhost",
					"port": 8080,
				},
				"database": map[string]any{
					"password": "s3cr3t",
				},
			},
		}, nil)

	defaultProvider := config2.NewDefaultProvider(map[string]any{
		"server": map[string]any{"host": "localhost", "port": 8080},
	})
	fileProvider := config2.NewFileProvider([]string{filePath})
	envProvider := config2.NewEnvProvider("APP_")
	vaultProvider := config2.NewVaultProvider(vaultMock, "prod", "myservice")

	loader := config2.New(
		defaultProvider,
		fileProvider,
		envProvider,
		vaultProvider,
	)

	if err := loader.Load(context.Background()); err != nil {
		fmt.Println("load error:", err)
	}

	var cfg exampleConfig
	if err := loader.Unmarshal(&cfg); err != nil {
		fmt.Println("unmarshal error:", err)
	}

	if err := config2.Validate(cfg); err != nil {
		fmt.Println("validation error:", err)
	}

	fmt.Printf("server port: %d\n", cfg.Server.Port)
	fmt.Printf("database host: %s\n", cfg.Database.Host)
	fmt.Printf("database password: %s\n", cfg.Database.Password)

	settings := loader.AllSettings()
	masked := config2.MaskSensitive(settings)

	if db, ok := masked["database"].(map[string]any); ok {
		fmt.Printf("masked db password: %v\n", db["password"])
	}

	fmt.Printf("timeout set: %v\n", time.Second > 0)

	// Output:
	// server port: 8080
	// database host: db.example.com
	// database password: s3cr3t
	// masked db password: s3**3t
	// timeout set: true
}
