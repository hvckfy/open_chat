package logger

import (
	"fmt"
	"openchat/services/config"
	"os"

	"github.com/gin-gonic/gin"
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

// HTTP-aware logging functions

// LogError logs errors with HTTP context
func LogError(c *gin.Context, msg string, err error, fields ...zap.Field) {
	if Log == nil {
		return
	}

	baseFields := []zap.Field{
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("client_ip", c.ClientIP()),
		zap.Error(err),
	}

	Log.Error(msg, append(baseFields, fields...)...)
}

// LogWarn logs warnings with HTTP context
func LogWarn(c *gin.Context, msg string, fields ...zap.Field) {
	if Log == nil {
		return
	}

	baseFields := []zap.Field{
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("client_ip", c.ClientIP()),
	}

	Log.Warn(msg, append(baseFields, fields...)...)
}

// LogAuthSuccess logs successful authentication
func LogAuthSuccess(c *gin.Context, username, action string) {
	if Log == nil {
		return
	}

	Log.Info("Authentication successful",
		zap.String("action", action),
		zap.String("username", username),
		zap.String("client_ip", c.ClientIP()))
}

// Simple logging functions (without HTTP context)

// Error logs an error message
func Error(msg string, fields ...zap.Field) {
	if Log != nil {
		Log.Error(msg, fields...)
	}
}

// Warn logs a warning message
func Warn(msg string, fields ...zap.Field) {
	if Log != nil {
		Log.Warn(msg, fields...)
	}
}

// Info logs an info message
func Info(msg string, fields ...zap.Field) {
	if Log != nil {
		Log.Info(msg, fields...)
	}
}

// Debug logs a debug message
func Debug(msg string, fields ...zap.Field) {
	if Log != nil {
		Log.Debug(msg, fields...)
	}
}
