package cmd

import (
	"time"

	"git.yoshino-s.xyz/yoshino-s/derperer/derperer"
	"git.yoshino-s.xyz/yoshino-s/derperer/fofa"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	fofaClient = fofa.Fofa{}
	batch      int
)

var serverCmd = &cobra.Command{
	Use: "server",
	Run: func(cmd *cobra.Command, args []string) {
		derpererConfig := derperer.DerpererConfig{
			Address:        ":8080",
			UpdateInterval: 2 * time.Minute,
			FetchInterval:  time.Hour,
			FetchBatch:     batch,
			FofaClient:     fofaClient,
			LatencyLimit:   time.Second,
		}
		derperer := derperer.NewDerperer(derpererConfig)
		derperer.Start()
	},
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.AddCommand(serverCmd)

	serverCmd.PersistentFlags().StringVarP(&fofaClient.Email, "fofa-email", "e", "", "fofa email")
	serverCmd.PersistentFlags().StringVarP(&fofaClient.Key, "fofa-key", "k", "", "fofa key")
	serverCmd.PersistentFlags().IntVarP(&batch, "batch", "b", 25, "batch")
}

func initConfig() {
	viper.AutomaticEnv()

	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.derperer")
	viper.SetConfigName("derperer")
	viper.SetConfigType("yaml")

	viper.ReadInConfig()

	if fofaClient.Email == "" {
		fofaClient.Email = viper.GetString("fofa_email")
	}
	if fofaClient.Key == "" {
		fofaClient.Key = viper.GetString("fofa_key")
	}
}
