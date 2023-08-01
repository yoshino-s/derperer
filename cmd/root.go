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
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	if err := rootCmd.Execute(); err != nil {
		zap.L().Fatal("failed to execute root command", zap.Error(err))
	}
}
