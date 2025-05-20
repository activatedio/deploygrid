package config

type LoggingConfig struct {
	Level   string `mapstructure:"level"`
	DevMode bool   `mapstructure:"devMode"`
}

type ClusterConfig struct {
	Name           string `mapstructure:"name"`
	Address        string `mapstructure:"address"`
	KubeConfigPath string `mapstructure:"kubeConfigPath"`
	ContextName    string `mapstructure:"contextName"`
	Local          bool   `mapstructure:"local"`
}

type ClustersConfig struct {
	Clusters     []ClusterConfig `mapstructure:"clusters"`
	Environments []string        `mapstructure:"environments"`
}

type SwaggerConfig struct {
	SwaggerUiUrl string `mapstructure:"swaggerUiUrl"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}
