package config

type LoggingConfig struct {
	Level   string `mapstructure:"level"`
	DevMode bool   `mapstructure:"dev_mode"`
}

const (
	RepositoryModeK8s  = "k8s"
	RepositoryModeStub = "stub"
)

type RepositoryCommonConfig struct {
	Mode string `mapstructure:"mode"`
}

type RepositoryStubConfig struct {
	StaticDataBytes []byte
	StaticDataPath  string `mapstructure:"static_data_path"`
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
