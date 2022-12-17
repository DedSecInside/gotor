package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var zapper *zap.Logger

func init() {
	cfg := zap.NewProductionConfig()
	// TODO - Setup flag to alter log level
	cfg.Level.SetLevel(zap.InfoLevel)
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zapper = zap.Must(cfg.Build())
}

func Debug(msg string, keysAndValues ...interface{}) {
	zapper.Sugar().Debugw(msg, keysAndValues)
}

func Info(msg string, keysAndValues ...interface{}) {
	zapper.Sugar().Infow(msg, keysAndValues...)
}

func Warn(msg string, keysAndValues ...interface{}) {
	zapper.Sugar().Warnw(msg, keysAndValues...)
}

func Error(msg string, keysAndValues ...interface{}) {
	zapper.Sugar().Errorw(msg, keysAndValues...)
}

func Fatal(msg string, keysAndValues ...interface{}) {
	zapper.Sugar().Fatalw(msg)
}
