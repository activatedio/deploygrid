package config

type LoggingConfig struct {
	Level   string `mapstructure:"level"`
	DevMode bool   `mapstructure:"dev_mode"`
}
