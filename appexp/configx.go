package appexp

import (
	"gopp"
	"sync"

	"github.com/42wim/matterbridge/bridge/config"
	"github.com/spf13/viper"
)

type Config struct {
	v *viper.Viper
	sync.RWMutex

	cv *config.BridgeValues
}

func ToPubCfg(c config.Config) *Config {
	var r *Config
	gopp.OpAssign(&r, c)
	return r
}

func ViperValues(c config.Config) *viper.Viper {
	return ToPubCfg(c).v
}
