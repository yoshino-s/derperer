package derperer

import (
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/yoshino-s/go-framework/configuration"
	"github.com/yoshino-s/go-framework/utils"
)

var _ configuration.Configuration = (*config)(nil)

type config struct {
	RefetchInterval time.Duration `mapstructure:"refetch_interval"`
	FetchLimit      int           `mapstructure:"fetch_limit"`

	RecheckInterval  time.Duration `mapstructure:"recheck_interval"`
	CheckDuration    time.Duration `mapstructure:"check_duration"`
	CheckConcurrency int           `mapstructure:"check_concurrency"`

	CN bool `mapstructure:"cn"`
}

func (c *config) Register(set *pflag.FlagSet) {
	set.Duration("derperer.refetch_interval", time.Minute*10, "The interval at which to fetch data")
	set.Int("derperer.fetch_limit", 100, "Limit of fofa result to fetch")
	set.Duration("derperer.recheck_interval", time.Second*10, "The interval at which to recheck abandoned nodes")
	set.Duration("derperer.check_duration", time.Second*10, "The duration for which to check nodes")
	set.Int("derperer.check_concurrency", 10, "The number of concurrent tests to run")
	set.Bool("derperer.cn", false, "Only fetch nodes in China")
	utils.MustNoError(viper.BindPFlags(set))
	configuration.Register(c)
}

func (c *config) Read() {
	utils.MustDecodeFromMapstructure(viper.AllSettings()["derperer"], c)
}
