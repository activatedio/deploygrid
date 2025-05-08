package viper

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
	"strings"
)

const (
	KeyConfigPath = "CONFIG_PATH"
)

type options struct {
	configPath string
}

type Option func(*options)

func WithConfigPath(path string) Option {
	return func(o *options) {
		o.configPath = path
	}
}

func NewViper(opts ...Option) *viper.Viper {

	o := &options{}

	for _, _o := range opts {
		_o(o)
	}

	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(`.`, `_`))

	if o.configPath == "" {
		o.configPath = os.Getenv(KeyConfigPath)
	}

	if o.configPath != "" {
		v.SetConfigFile(o.configPath)
		if err := v.ReadInConfig(); err != nil {
			panic(fmt.Errorf("Fatal error config file: %s \n", err))
		}
	}

	return v
}
