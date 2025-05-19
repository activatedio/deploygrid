package config

import (
	apiinfraconfig "github.com/activatedio/deploygrid/pkg/apiinfra/config"
	"github.com/spf13/viper"
)

func NewLoggingConfig(v *viper.Viper) *LoggingConfig {
	return apiinfraconfig.MustUnmarshallAndValidate(v, PrefixLogging, &LoggingConfig{})
}

func NewClustersConfig(v *viper.Viper) *ClustersConfig {
	return apiinfraconfig.MustUnmarshallAndValidate(v, PrefixClusters, &ClustersConfig{})
}

func NewSwaggerConfig(v *viper.Viper) *SwaggerConfig {
	return apiinfraconfig.MustUnmarshallAndValidate(v, PrefixSwagger, &SwaggerConfig{})
}

func NewServerConfig(v *viper.Viper) *ServerConfig {
	return apiinfraconfig.MustUnmarshallAndValidate(v, PrefixServer, &ServerConfig{})
}
