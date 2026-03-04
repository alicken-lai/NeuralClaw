package observability

import (
	"log"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func InitLogger(level string) {
	config := zap.NewProductionConfig()

	// Parse log level
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		zapLevel = zap.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(zapLevel)

	// Customize encoding slightly for better console output if needed, but production JSON is fine
	logger, err := config.Build()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}

	Logger = logger
}

func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}
