package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	rootCmd = &cobra.Command{
		Use: "derperer",
	}
	logLevel string
)

func init() {
	cobra.OnInitialize(initLog)
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level")
}

func initConfig() {
	viper.AutomaticEnv()

	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.derperer")
	viper.SetConfigName("derperer")
	viper.SetConfigType("yaml")

	viper.ReadInConfig()
}

func initLog() {
	var config zap.Config
	if Version == "dev" {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}
	err := config.Level.UnmarshalText([]byte(logLevel))
	if err != nil {
		fmt.Printf("failed to parse log level: %s\n", logLevel)
	}
	logger, err := config.Build()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(logger)
}

// Execute executes the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		zap.L().Fatal("failed to execute root command", zap.Error(err))
	}
}
