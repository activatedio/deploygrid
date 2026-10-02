package fx

import (
	"github.com/gorilla/mux"
	"go.uber.org/fx"

	apiinframux "github.com/activatedio/deploygrid/pkg/apiinfra/mux"
	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/controller"
	"github.com/activatedio/deploygrid/pkg/repository/k8s"
	"github.com/activatedio/deploygrid/pkg/runner"
	"github.com/activatedio/deploygrid/pkg/service"
)

// Index assembles the application from an already-loaded configuration.
func Index(m *config.Main) fx.Option {

	// The declared half of the model comes from custom resources when a
	// control cluster is configured; otherwise a single "default" System is
	// synthesised from the v1 cluster configuration.
	catalog := fx.Provide(service.NewConfigCatalog)
	if m.Control.Enabled {
		catalog = fx.Options(
			fx.Provide(k8s.NewControllers),
			fx.Provide(service.NewControlCatalog),
			fx.Provide(service.NewStatusWriter),
			fx.Invoke(func(*service.StatusWriter) {}),
		)
	}

	return fx.Module("deploygrid",
		fx.Provide(
			func() *apiinframux.OpenapiConfig {
				return &apiinframux.OpenapiConfig{
					Title:       "Deploy Grid",
					Version:     "2.0",
					Description: "Deploy Grid",
				}
			},
		),
		config.Index(m),
		controller.Index(),
		k8s.Index(),
		catalog,
		fx.Provide(
			runner.NewServer,
			apiinframux.NewOpenapi,
			service.NewSystemService,
			service.NewGridService,
		),
		fx.Invoke(func(service service.GridService) {
			service.Init()
		}),
		fx.Invoke(func(r *mux.Router, o apiinframux.Openapi, g controller.Grid, s controller.Systems) error {
			return o.Mount(r, g.OpenapiBuilder(), s.OpenapiBuilder())
		}),
	)
}
