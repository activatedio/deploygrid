package config

type LoggingConfig struct {
	Level   string
	DevMode bool
}

type Main struct {
	Namespace string
	Logging   *LoggingConfig
	Server    *ServerConfig
	Swagger   *SwaggerConfig
}

type SwaggerConfig struct {
	SwaggerUiUrl string
}

type ServerConfig struct {
	Host string
	Port int
}
