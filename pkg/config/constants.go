package config

const (
	// EnvPrefix is prepended to every environment variable override, for
	// example DEPLOYGRID_SERVER_PORT.
	EnvPrefix = "DEPLOYGRID"
	// EnvConfigPath names the environment variable that locates the YAML
	// configuration file when no --config flag is given.
	EnvConfigPath = "CONFIG_PATH"

	DefaultControlNamespace = "deploygrid"
)
