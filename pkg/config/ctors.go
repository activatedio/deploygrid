package config

import (
	apiinfraconfig "github.com/activatedio/deploygrid/pkg/apiinfra/config"
	"github.com/spf13/viper"
)

func NewLoggingConfig(v *viper.Viper) *LoggingConfig {
	return apiinfraconfig.MustUnmarshallAndValidate(v, PrefixLogging, &LoggingConfig{})
}

func NewRepositoryCommonConfig(v *viper.Viper) *RepositoryCommonConfig {
	return apiinfraconfig.MustUnmarshallAndValidate(v, PrefixRepositoryCommon, &RepositoryCommonConfig{})
}

func NewRepositoryStubConfig(v *viper.Viper) *RepositoryStubConfig {
	return apiinfraconfig.MustUnmarshallAndValidate(v, PrefixRepositoryStub, &RepositoryStubConfig{})
}

func NewClustersConfig(v *viper.Viper) *ClustersConfig {
	return apiinfraconfig.MustUnmarshallAndValidate(v, PrefixClusters, &ClustersConfig{})
}
