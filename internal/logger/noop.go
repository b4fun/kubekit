package logger

type noopLogger struct{}

var _ Logger = (*noopLogger)(nil)

func (l *noopLogger) Log(msg string, args ...interface{}) {

}

// NoOp returns a logger that does nothing.
var NoOp = &noopLogger{}
