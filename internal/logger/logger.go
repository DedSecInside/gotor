package logger

import (
	"go.uber.org/zap"
)

var zapper *zap.Logger

func init() {
	var err error
	zapper, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}
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
