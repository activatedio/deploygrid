package config

type LoggingConfig struct {
	Level   string `mapstructure:"level"`
	DevMode bool   `mapstructure:"dev_mode"`
}

type ClusterConfig struct {
	Name    string `mapstructure:"name"`
	Address string `mapstructure:"address"`
}

type ClustersConfig struct {
	Clusters []ClusterConfig `mapstructure:"clusters"`
}

type SwaggerConfig struct {
	SwaggerUIURL string `mapstructure:"swagger_ui_url"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}
