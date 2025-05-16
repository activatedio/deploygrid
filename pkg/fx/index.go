package fx

import (
	apiinframux "github.com/activatedio/deploygrid/pkg/apiinfra/mux"
	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/controller"
	"github.com/activatedio/deploygrid/pkg/runner"
	"github.com/activatedio/deploygrid/pkg/service"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

func Index(v *viper.Viper) fx.Option {

	config.LoadDefaults(v)

	return fx.Module("deploygrid", fx.Provide(
		func() *viper.Viper {
			return v
		},
		func() *apiinframux.OpenapiConfig {
			return &apiinframux.OpenapiConfig{
				Title:       "Deploy Grid",
				Version:     "1.0",
				Description: "Deploy Grid",
			}
		},
	),
		config.Index(),
		controller.Index(v),
		fx.Provide(
			runner.NewServer,
			apiinframux.NewOpenapi,
			service.NewGridService,
		),
		fx.Invoke(func(r *mux.Router, o apiinframux.Openapi, d controller.Deployments) error {
			return o.Mount(r, d.OpenapiBuilder())
		}),
	)
}
