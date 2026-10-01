package controller

import (
	"go.uber.org/fx"
)

func Index() fx.Option {

	return fx.Module("deploygrid.controller", fx.Provide(
		NewHealth,
		NewGrid,
		NewSystems,
		NewRouter,
	))

}
