package pipe

import (
	"github.com/myszqua/vaultify/models"
)

// TracerConfig standardizes a tracer dependency's configuration.
type TracerConfig struct {
	ServiceName                 string          `yaml:"service_name" json:"service_name" config:"required"`
	SamplerType                 string          `yaml:"sampler_type" json:"sampler_type" default:"const"`
	SamplerParam                float64         `yaml:"sampler_param" json:"sampler_param" default:"1"`
	ReporterLogSpans            bool            `yaml:"reporter_log_spans" json:"reporter_log_spans" default:"true"`
	ReporterBufferFlushInterval models.Duration `yaml:"reporter_buffer_flush_interval" json:"reporter_buffer_flush_interval" default:"1s"` //nolint:lll
	ReporterHost                string          `yaml:"reporter_host" json:"reporter_host" default:"localhost"`
	ReporterPort                int             `yaml:"reporter_port" json:"reporter_port" default:"6831"`
	Disabled                    bool            `yaml:"disabled" json:"disabled" default:"false"`
}
