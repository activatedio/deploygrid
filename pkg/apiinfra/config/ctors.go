package config

import (
	"github.com/spf13/viper"
)

func NewLoggingConfig(v *viper.Viper) *LoggingConfig {
	return MustUnmarshallAndValidate(v, PrefixLogging, &LoggingConfig{})
}
