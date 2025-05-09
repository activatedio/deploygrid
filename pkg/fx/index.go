package fx

import (
	"fmt"
	apiinframux "github.com/activatedio/deploygrid/pkg/apiinfra/mux"
	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/controller"
	"github.com/activatedio/deploygrid/pkg/repository/stub"
	"github.com/activatedio/deploygrid/pkg/runner"
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
		RepositoryIndex(v),
		fx.Provide(
			runner.NewServer,
			apiinframux.NewOpenapi,
		),
		fx.Invoke(func(r *mux.Router, o apiinframux.Openapi, d controller.Deployments) error {
			return o.Mount(r, d.OpenapiBuilder())
		}),
	)
}

func RepositoryIndex(v *viper.Viper) fx.Option {

	rc := config.NewRepositoryCommonConfig(v)

	switch rc.Mode {
	case config.RepositoryModeStub:
		return stub.Index(v)
	case config.RepositoryModeK8s:
		panic(fmt.Errorf("not yet supported %s", rc.Mode))
	default:
		panic(fmt.Errorf("invalid repository mode %s", rc.Mode))
	}
}
