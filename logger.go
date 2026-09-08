package vaultify

// Logger is the minimal logging interface used across vaultify.
type Logger interface {
	Info(args ...any)
	Warn(args ...any)
	Debug(args ...any)
}

type nopLogger struct{}

func (nopLogger) Info(...any)  {}
func (nopLogger) Warn(...any)  {}
func (nopLogger) Debug(...any) {}

func newNopLogger() Logger { //nolint:ireturn
	return nopLogger{}
}
