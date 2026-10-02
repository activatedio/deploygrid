package config

// Defaults returns the configuration used when no source provides a value.
// Every nested struct is populated so that environment variable overrides are
// discovered even when the YAML file omits the section.
func Defaults() *Main {
	return &Main{
		Logging: LoggingConfig{
			Level:   "info",
			DevMode: false,
		},
		Server: ServerConfig{
			Host: "127.0.0.1",
			Port: 8080,
		},
		Swagger: SwaggerConfig{},
		Control: ControlConfig{
			Enabled:   false,
			Namespace: DefaultControlNamespace,
		},
		Clusters: ClustersConfig{},
		Collector: CollectorConfig{
			FlushSeconds:     2,
			HeartbeatSeconds: 30,
		},
		Sources: SourcesConfig{},
	}
}
