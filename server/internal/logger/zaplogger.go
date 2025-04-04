package logger

import (
	"time"

	"github.com/MrPunder/sirius-loyality-system/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type zapLogger struct {
	logZap *zap.SugaredLogger
	logger *zap.Logger // Сохраняем ссылку на оригинальный логгер для вызова Sync()
}

func NewZapLogger(conf *config.Config) (*zapLogger, error) {
	logLevel, err := zap.ParseAtomicLevel(conf.Log.Level)
	if err != nil {
		return nil, err
	}
	// Настройка энкодера
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	encoder := zapcore.NewConsoleEncoder(encoderConfig)

	// Настройка ротации логов для обычных логов
	stdLogWriter := &lumberjack.Logger{
		Filename:   conf.Log.Path,
		MaxSize:    conf.Log.MaxSize,    // Максимальный размер в МБ
		MaxBackups: conf.Log.MaxBackups, // Максимальное количество файлов бэкапа
		MaxAge:     conf.Log.MaxAge,     // Максимальный возраст в днях
		Compress:   conf.Log.Compress,   // Сжимать ротированные файлы
	}

	// Настройка ротации логов для ошибок
	errLogWriter := &lumberjack.Logger{
		Filename:   conf.Log.ErrorPath,
		MaxSize:    conf.Log.MaxSize,
		MaxBackups: conf.Log.MaxBackups,
		MaxAge:     conf.Log.MaxAge,
		Compress:   conf.Log.Compress,
	}

	// Создание ядра логгера
	stdCore := zapcore.NewCore(
		encoder,
		zapcore.AddSync(stdLogWriter),
		logLevel,
	)

	errCore := zapcore.NewCore(
		encoder,
		zapcore.AddSync(errLogWriter),
		zap.ErrorLevel,
	)

	// Объединение ядер
	core := zapcore.NewTee(stdCore, errCore)

	// Создание логгера
	logger := zap.New(core, zap.Development(), zap.AddCaller(), zap.AddCallerSkip(1))

	return &zapLogger{
		logZap: logger.Sugar(),
		logger: logger,
	}, nil
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

// Debugf logs formatted message at debug level
func (logger *zapLogger) Debugf(str string, arg ...any) {
	logger.logZap.Debugf(str, arg...)
}

// ResponseLog makes response log
func (logger *zapLogger) ResponseLog(status int, size int, duration time.Duration) {
	logger.logZap.Infow("Send response with",
		"status", status,
		"size", size,
		"time", duration.String(),
	)
}

// Close закрывает логгер, сбрасывая все буферизованные логи
func (logger *zapLogger) Close() error {
	return logger.logger.Sync()
}
