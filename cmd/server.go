package cmd

import (
	"time"

	"git.yoshino-s.xyz/yoshino-s/derperer/derperer"
	"git.yoshino-s.xyz/yoshino-s/derperer/fofa"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	derpererConfig = derperer.DerpererConfig{
		FofaClient: fofa.Fofa{},
	}
)

var serverCmd = &cobra.Command{
	Use: "server",
	Run: func(cmd *cobra.Command, args []string) {
		derperer, err := derperer.NewDerperer(derpererConfig)
		if err != nil {
			zap.L().Fatal("failed to create derperer", zap.Error(err))
		}
		derperer.Start()
	},
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.AddCommand(serverCmd)

	serverCmd.PersistentFlags().StringVarP(&derpererConfig.FofaClient.Email, "fofa-email", "e", "", "fofa email")
	serverCmd.PersistentFlags().StringVarP(&derpererConfig.FofaClient.Key, "fofa-key", "k", "", "fofa key")
	serverCmd.PersistentFlags().IntVarP(&derpererConfig.FetchBatch, "batch", "b", 100, "batch")
	serverCmd.PersistentFlags().StringVarP(&derpererConfig.Address, "address", "a", ":8080", "address")
	serverCmd.PersistentFlags().DurationVarP(&derpererConfig.UpdateInterval, "update-interval", "u", 10*time.Minute, "update interval")
	serverCmd.PersistentFlags().DurationVarP(&derpererConfig.FetchInterval, "fetch-interval", "f", 4*time.Hour, "fetch interval")
	serverCmd.PersistentFlags().DurationVarP(&derpererConfig.LatencyLimit, "latency-limit", "l", time.Second, "latency limit")
	serverCmd.PersistentFlags().DurationVarP(&derpererConfig.ProbeTimeout, "probe-timeout", "t", 5*time.Second, "probe timeout")
	serverCmd.PersistentFlags().IntVarP(&derpererConfig.TestBatch, "test-batch", "T", 5, "test batch")
}

func initConfig() {
	viper.AutomaticEnv()

	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.derperer")
	viper.SetConfigName("derperer")
	viper.SetConfigType("yaml")

	viper.ReadInConfig()

	if derpererConfig.FofaClient.Email == "" {
		derpererConfig.FofaClient.Email = viper.GetString("fofa_email")
	}
	if derpererConfig.FofaClient.Key == "" {
		derpererConfig.FofaClient.Key = viper.GetString("fofa_key")
	}
}
