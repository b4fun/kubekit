package logger

type LogFunc func(msg string, args ...interface{})

func (f LogFunc) Log(msg string, args ...interface{}) {
	f(msg, args...)
}
