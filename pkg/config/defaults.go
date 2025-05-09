package config

import "github.com/spf13/viper"

func LoadDefaults(v *viper.Viper) {

	v.SetDefault(PrefixLogging, &LoggingConfig{
		Level:   "info",
		DevMode: false,
	})
	v.SetDefault(PrefixRepositoryCommon, &RepositoryCommonConfig{
		Mode: RepositoryModeK8s,
	})
	v.SetDefault(PrefixServer, &ServerConfig{
		Host: "127.0.0.1",
		Port: 8080,
	})

}
