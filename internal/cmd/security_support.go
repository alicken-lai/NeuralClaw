package cmd

import (
	"go.uber.org/zap"

	"neuralclaw/internal/config"
	"neuralclaw/internal/observability"
	"neuralclaw/internal/security"
)

func newSecurityGuard() *security.Guard {
	guard, err := security.NewGuard(config.GlobalConfig.Security)
	if err != nil {
		observability.Logger.Fatal("Failed to initialize security guard", zap.Error(err))
	}
	return guard
}
