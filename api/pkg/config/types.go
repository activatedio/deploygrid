package config

type LoggingConfig struct {
	Level   string
	DevMode bool
}

type ClusterConfig struct {
	Name           string
	Address        string
	KubeConfigPath string
	ContextName    string
}

type ClustersConfig struct {
	Clusters []ClusterConfig
}

type SwaggerConfig struct {
	SwaggerUIURL string
}

type ServerConfig struct {
	Host string
	Port int
}
