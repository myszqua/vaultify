package pipe

// ServerConfig standardizes a server dependency's configuration.
type ServerConfig struct {
	GrpcAddress      string `yaml:"grpc_address" json:"grpc_address"`
	Port             int    `yaml:"port" json:"port" default:"8080" min:"1" max:"65535"`
	BaseURL          string `yaml:"base_url" json:"base_url" default:"/"`
	Mode             string `yaml:"mode" json:"mode" default:"debug"`
	Timeout          string `yaml:"timeout" json:"timeout" default:"30s"`
	RequestSizeLimit string `yaml:"request_size_limit" json:"request_size_limit" default:"10mb"`
	ServiceName      string `yaml:"service_name" json:"service_name" config:"required"`
	RouterVersion    string `yaml:"router_version" json:"router_version" default:"v1"`
	VersionEndpoints bool   `yaml:"version_endpoints" json:"version_endpoints" default:"true"`
}
