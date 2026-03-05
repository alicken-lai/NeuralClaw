package observability

import (
	"log"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

// Tracker is the global token usage tracker, initialized after the logger.
var Tracker *TokenTracker

func InitLogger(level string) {
	config := zap.NewProductionConfig()

	// Parse log level
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		zapLevel = zap.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(zapLevel)

	logger, err := config.Build()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}

	Logger = logger
}

// InitTokenTracker sets up the global Tracker, writing to a JSONL file under dataDir.
func InitTokenTracker(dataDir string) {
	fp := filepath.Join(dataDir, "token_usage.jsonl")
	t, err := NewTokenTracker(fp)
	if err != nil {
		Logger.Warn("Failed to initialize TokenTracker, token tracking disabled", zap.Error(err))
		return
	}
	Tracker = t
	Logger.Info("TokenTracker initialized", zap.String("path", fp))
}

func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
	if Tracker != nil {
		_ = Tracker.Close()
	}
}
