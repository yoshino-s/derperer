package cmd

import (
	"reflect"
	"time"

	"git.yoshino-s.xyz/yoshino-s/derperer/derperer"
	"git.yoshino-s.xyz/yoshino-s/derperer/fofa"
	"github.com/mitchellh/mapstructure"
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
	cobra.OnInitialize(initDerpererConfig)

	rootCmd.AddCommand(serverCmd)

	serverCmd.PersistentFlags().String("config.DatabaseUri", "", "database url")
	serverCmd.PersistentFlags().String("config.FofaClient.Email", "", "fofa email")
	serverCmd.PersistentFlags().String("config.FofaClient.Key", "", "fofa key")
	serverCmd.PersistentFlags().Int("config.FetchBatch", 100, "batch")
	serverCmd.PersistentFlags().String("config.Address", ":8080", "address")
	serverCmd.PersistentFlags().Duration("config.UpdateInterval", 10*time.Minute, "update interval")
	serverCmd.PersistentFlags().Duration("config.FetchInterval", 4*time.Hour, "fetch interval")
	serverCmd.PersistentFlags().Duration("config.LatencyLimit", time.Second, "latency limit")
	serverCmd.PersistentFlags().Duration("config.ProbeTimeout", 5*time.Second, "probe timeout")
	serverCmd.PersistentFlags().Int("config.TestBatch", 5, "test batch")

	viper.BindPFlags(serverCmd.PersistentFlags())
}

func toTimeHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if t == reflect.TypeOf(time.Time{}) {
			switch f.Kind() {
			case reflect.String:
				return time.Parse(time.RFC3339, data.(string))
			case reflect.Float64:
				return time.Unix(0, int64(data.(float64))*int64(time.Millisecond)), nil
			case reflect.Int64:
				return time.Unix(0, data.(int64)*int64(time.Millisecond)), nil
			default:
				return data, nil
			}
		} else if t == reflect.TypeOf(time.Duration(0)) {
			switch f.Kind() {
			case reflect.String:
				return time.ParseDuration(data.(string))
			case reflect.Float64:
				return time.Duration(data.(float64)), nil
			case reflect.Int64:
				return time.Duration(data.(int64)), nil
			default:
				return data, nil
			}
		}

		return data, nil
	}
}

func decode(input interface{}, result interface{}) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: nil,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			toTimeHookFunc()),
		Result: result,
	})
	if err != nil {
		return err
	}

	if err := decoder.Decode(input); err != nil {
		return err
	}
	return err
}

func initDerpererConfig() {
	err := decode(viper.AllSettings()["config"], &derpererConfig)
	if err != nil {
		zap.L().Fatal("failed to decode config", zap.Error(err))
	}
}
