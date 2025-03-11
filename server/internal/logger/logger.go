package logger

type Logger interface {
	Info(mes string)
	Errorf(str string, arg ...any)
	Error(mess string)
	Infof(str string, arg ...any)
	Debug(mess string)
}
