package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yoshino-s/go-framework/application"
	"github.com/yoshino-s/go-framework/cmd"
	"github.com/yoshino-s/go-framework/configuration"
	"go.uber.org/zap"
)

var name = "derperer"
var app = application.NewMainApplication()

var (
	rootCmd = &cobra.Command{
		Use: name,
	}
)

func init() {
	cobra.OnInitialize(func() {
		configuration.Setup(name)

		zap.ReplaceGlobals(app.Logger)
	})

	configuration.GenerateConfiguration.Register(rootCmd.PersistentFlags())
	app.Configuration().Register(rootCmd.PersistentFlags())

	rootCmd.AddCommand(cmd.VersionCmd)
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}
