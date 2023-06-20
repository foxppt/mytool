package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var SugarLogger *zap.SugaredLogger

func init() {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	config.EncoderConfig.MessageKey = "msg"
	config.EncoderConfig.TimeKey = "ts"
	config.EncoderConfig.LineEnding = zapcore.DefaultLineEnding
	config.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	config.EncoderConfig.EncodeCaller = zapcore.FullCallerEncoder
	config.EncoderConfig.ConsoleSeparator = " | "

	logger, err := config.Build()
	if err != nil {
		panic(err)
	}

	SugarLogger = logger.Sugar()
}
