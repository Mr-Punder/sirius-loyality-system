package logger

import (
	"time"

	"github.com/MrPunder/sirius-loyality-system/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type zapLogger struct {
	logZap *zap.SugaredLogger
}

func NewZapLogger(conf *config.Config) (*zapLogger, error) {
	logLevel, err := zap.ParseAtomicLevel(conf.Log.Level)
	if err != nil {
		return nil, err
	}
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	logConf := zap.Config{
		Level:             logLevel,
		Development:       true,
		Encoding:          "console",
		EncoderConfig:     encoderConfig,
		OutputPaths:       []string{conf.Log.Path},
		ErrorOutputPaths:  []string{conf.Log.ErrorPath},
		DisableStacktrace: true,
	}

	logger, err := logConf.Build()
	if err != nil {
		return nil, err
	}
	defer logger.Sync()

	return &zapLogger{logger.Sugar()}, nil
}

// RequestLog makes request log
func (logger *zapLogger) RequestLog(method string, path string) {
	logger.logZap.Infow("incoming request",
		"method", method,
		"path", path,
	)
}

// Info logs message at info level
func (logger *zapLogger) Info(mes string) {
	logger.logZap.Info(mes)
}

func (logger *zapLogger) Infof(str string, arg ...any) {
	logger.logZap.Infof(str, arg...)
}
func (logger *zapLogger) Errorf(str string, arg ...any) {
	logger.logZap.Errorf(str, arg...)

}

// Error logs message at error level
func (logger *zapLogger) Error(mes string) {
	logger.logZap.Error(mes)
}

// Debug logs message at debug level
func (logger *zapLogger) Debug(mes string) {
	logger.logZap.Debug(mes)
}

// ResponseLog makes response log
func (logger *zapLogger) ResponseLog(status int, size int, duration time.Duration) {
	logger.logZap.Infow("Send response with",
		"status", status,
		"size", size,
		"time", duration.String(),
	)
}
