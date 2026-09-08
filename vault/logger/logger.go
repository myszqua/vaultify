package logger

// Writer is the minimal logging interface vaultify needs.
type Writer interface {
	Info(args ...any)
	Warn(args ...any)
	Debug(args ...any)
}
