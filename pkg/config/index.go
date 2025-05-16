package config

import "go.uber.org/fx"

func Index() fx.Option {
	return fx.Module("deploygrid.config", fx.Provide(
		NewClustersConfig,
		NewSwaggerConfig,
		NewServerConfig,
	))
}
