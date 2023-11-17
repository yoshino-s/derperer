package cmd

import (
	"reflect"
	"strconv"
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

	serverCmd.Flags().String("config.FofaClient.Email", "", "fofa email")
	serverCmd.Flags().String("config.FofaClient.Key", "", "fofa key")
	serverCmd.Flags().Int("config.FetchBatch", 100, "batch")
	serverCmd.Flags().Duration("config.FetchInterval", 4*time.Hour, "fetch interval")
	serverCmd.Flags().String("config.Address", ":8080", "address")
	serverCmd.Flags().Duration("config.DERPMapPolicy.RecheckInterval", time.Hour, "update interval")
	serverCmd.Flags().Duration("config.DERPMapPolicy.CheckDuration", 5*time.Second, "check duration")
	serverCmd.Flags().Float64("config.DERPMapPolicy.BaselineBandwidth", 2, "bandwidth limit, unit: Mbps")
	serverCmd.Flags().Int("config.DERPMapPolicy.TestConcurrency", 4, "test concurrency")
	serverCmd.Flags().String("config.AdminToken", "", "admin token")

	viper.BindPFlags(serverCmd.Flags())
}

func toTimeHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if t == reflect.TypeOf(float64(0)) {
			switch f.Kind() {
			case reflect.String:
				return strconv.ParseFloat(data.(string), 64)
			case reflect.Int64:
				return float64(data.(int64)), nil
			default:
				return data, nil
			}
		} else if t == reflect.TypeOf(time.Time{}) {
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
