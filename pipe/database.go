package pipe

import (
	"github.com/myszqua/vaultify/models"
)

// DatabaseConfig standardizes a database dependency's configuration.
type DatabaseConfig struct {
	Host              string          `yaml:"host" json:"host" config:"required"`
	Port              int             `yaml:"port" json:"port" default:"5432" min:"1" max:"65535"`
	User              string          `yaml:"user" json:"user" config:"required"`
	Password          string          `yaml:"password" json:"password" config:"required"`
	Name              string          `yaml:"name" json:"name" config:"required"`
	SSLMode           string          `yaml:"ssl_mode" json:"ssl_mode" default:"disable"`
	MaxOpenConns      int             `yaml:"max_open_conns" json:"max_open_conns" default:"25"`
	MaxIdleConns      int             `yaml:"max_idle_conns" json:"max_idle_conns" default:"10"`
	ConnMaxLifetime   models.Duration `yaml:"conn_max_lifetime" json:"conn_max_lifetime" default:"5m"`
	GormLoggerEnabled bool            `yaml:"gorm_logger_enabled" json:"gorm_logger_enabled" default:"false"`
}

// DSN builds a postgres connection string from the config.
func (d *DatabaseConfig) DSN() string {
	return "host=" + d.Host +
		" port=" + itoa(d.Port) +
		" user=" + d.User +
		" password=" + d.Password +
		" dbname=" + d.Name +
		" sslmode=" + d.SSLMode
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}

	result := ""
	for i > 0 {
		result = string(rune('0'+i%10)) + result
		i /= 10
	}

	return result
}
