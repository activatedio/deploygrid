package controller

import (
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

func Index(v *viper.Viper) fx.Option {

	return fx.Module("deploygrid.controller", fx.Provide(
		NewHealth,
		NewGrid,
		NewRouter,
	))

}
