package cmd

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	rootCmd = &cobra.Command{
		Use: "derperer",
	}
)

// Execute executes the root command.
func Execute() {
	if Version == "dev" {
		logger, _ := zap.NewDevelopment()
		zap.ReplaceGlobals(logger)
		zap.L().Info("running in dev mode")
	} else {
		logger, _ := zap.NewProduction()
		zap.ReplaceGlobals(logger)
	}

	if err := rootCmd.Execute(); err != nil {
		zap.L().Fatal("failed to execute root command", zap.Error(err))
	}
}
