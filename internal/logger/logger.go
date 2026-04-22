package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

// Init 初始化日志
func Init(mode string) error {
	var config zap.Config

	if mode == "production" {
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	var err error
	log, err = config.Build(zap.AddCallerSkip(1))
	return err
}

// Sync 刷新日志缓冲区
func Sync() {
	if log != nil {
		_ = log.Sync()
	}
}

// Info 信息日志
func Info(msg string, fields ...zap.Field) {
	if log != nil {
		log.Info(msg, fields...)
	}
}

// Error 错误日志
func Error(msg string, err error, fields ...zap.Field) {
	if log == nil {
		return
	}
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	log.Error(msg, fields...)
}

// Fatal 致命错误
func Fatal(msg string, err error, fields ...zap.Field) {
	if log == nil {
		return
	}
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	log.Fatal(msg, fields...)
}

// Debug 调试日志
func Debug(msg string, fields ...zap.Field) {
	if log != nil {
		log.Debug(msg, fields...)
	}
}

// Warn 警告日志
func Warn(msg string, fields ...zap.Field) {
	if log != nil {
		log.Warn(msg, fields...)
	}
}

// With 创建带字段的日志
func With(fields ...zap.Field) *zap.Logger {
	if log == nil {
		return zap.NewNop()
	}
	return log.With(fields...)
}
