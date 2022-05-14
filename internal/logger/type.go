package logger

// Logger logs internal state.
type Logger interface {
	Log(msg string, args ...interface{})
}
