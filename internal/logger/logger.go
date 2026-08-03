// Package logger provides the node's single structured (JSON) logger,
// built on uber-go/zap, as required by the "Логи и Метрики" spec.
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New builds a JSON zap.Logger. level is one of debug/info/warn/error
// (defaults to info on anything else/empty).
func New(level string) (*zap.Logger, error) {
	var zlvl zapcore.Level
	if err := zlvl.UnmarshalText([]byte(level)); err != nil {
		zlvl = zapcore.InfoLevel
	}

	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(zlvl),
		Development:      false,
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	return cfg.Build()
}
