package pipe

// LoggerConfig standardizes a logger dependency's configuration.
type LoggerConfig struct {
	Level             string            `yaml:"level" json:"level" default:"info"`
	Encoding          string            `yaml:"encoding" json:"encoding" default:"json"`
	DisableStacktrace bool              `yaml:"disable_stacktrace" json:"disable_stacktrace" default:"false"`
	OutputPaths       []string          `yaml:"output_paths" json:"output_paths" default:"[stdout]"`
	ErrorOutputPaths  []string          `yaml:"error_output_paths" json:"error_output_paths" default:"[stderr]"`
	InitialFields     map[string]string `yaml:"initial_fields" json:"initial_fields"`
}
