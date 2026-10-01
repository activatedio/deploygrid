package config

import "go.uber.org/fx"

// Index provides the already-loaded Main configuration and its sections.
func Index(m *Main) fx.Option {
	return fx.Module("deploygrid.config",
		fx.Supply(m),
		fx.Provide(NewSubConfigs),
	)
}
