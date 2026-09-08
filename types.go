package vaultify

const unknown = "unknown"

// EventType represents the type of a configuration watch event.
type EventType int

const (
	// EventChanged indicates that a watched configuration source changed.
	EventChanged EventType = iota
	// EventReloaded indicates that the configuration was reloaded.
	EventReloaded
	// EventError indicates a watch error.
	EventError
)

func (e EventType) String() string {
	switch e {
	case EventChanged:
		return "changed"
	case EventReloaded:
		return "reloaded"
	case EventError:
		return "error"
	default:
		return unknown
	}
}

// WatchEvent describes a configuration change event produced by a provider.
type WatchEvent struct {
	Source string
	Type   EventType
	Error  error
}

// MergeStrategy defines how slices are merged when overriding values.
type MergeStrategy int

const (
	// MergeStrategyReplace replaces the base slice with the override.
	MergeStrategyReplace MergeStrategy = iota
	// MergeStrategyMerge appends override values, deduplicating.
	MergeStrategyMerge
)

func (s MergeStrategy) String() string {
	switch s {
	case MergeStrategyReplace:
		return "replace"
	case MergeStrategyMerge:
		return "merge"
	default:
		return unknown
	}
}

// MergeOptions configures the deep merge behavior.
type MergeOptions struct {
	SliceStrategy MergeStrategy
	LogConflicts  bool
}

// FileFormat represents a supported configuration file format.
type FileFormat int

const (
	// FormatYAML is the YAML file format.
	FormatYAML FileFormat = iota
	// FormatJSON is the JSON file format.
	FormatJSON
	// FormatTOML is the TOML file format.
	FormatTOML
)

func (f FileFormat) String() string {
	switch f {
	case FormatYAML:
		return "yaml"
	case FormatJSON:
		return "json"
	case FormatTOML:
		return "toml"
	default:
		return unknown
	}
}
