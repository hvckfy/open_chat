package logger

import (
	"account-service/services/config"
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log *zap.Logger

// InitLogger initializes the global logger
func InitLogger() error {

	if config.Data.Loki.Use != true {
		fmt.Println("Not stdout a file, UseLoki == false")
		return nil
	}

	// Use user-writable directory
	logDir := "./logs" // Relative to working directory
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// Lumberjack for log rotation
	lumberjackLogger := &lumberjack.Logger{
		Filename:   logDir + "/app.log",
		MaxSize:    10, // MB
		MaxBackups: 5,
		MaxAge:     30, // days
		Compress:   true,
	}

	// Encoder config
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// JSON encoder for Loki
	jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)

	// File core
	fileCore := zapcore.NewCore(jsonEncoder, zapcore.AddSync(lumberjackLogger), zapcore.InfoLevel)

	// Console core for development
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
	consoleCore := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel)

	// Combine cores
	core := zapcore.NewTee(fileCore, consoleCore)

	// Create logger with caller info
	Log = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	return nil
}

// Sync flushes any buffered log entries
func Sync() {
	if Log != nil {
		Log.Sync()
	}
}
