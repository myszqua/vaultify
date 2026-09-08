package pipe

// S3Config standardizes an s3 dependency's configuration.
type S3Config struct {
	Endpoint        string `yaml:"endpoint" json:"endpoint" config:"required"`
	Region          string `yaml:"region" json:"region" config:"required"`
	AccessKeyID     string `yaml:"access_key_id" json:"access_key_id" config:"required"`
	SecretAccessKey string `yaml:"secret_access_key" json:"secret_access_key" config:"required"`
	Bucket          string `yaml:"bucket" json:"bucket" config:"required"`
	UseSSL          bool   `yaml:"use_ssl" json:"use_ssl" default:"true"`
	ForcePathStyle  bool   `yaml:"force_path_style" json:"force_path_style" default:"false"`
}
