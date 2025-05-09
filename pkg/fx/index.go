package fx

import (
	"fmt"
	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/controller"
	"github.com/activatedio/deploygrid/pkg/repository/stub"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

func Index(v *viper.Viper) fx.Option {

	config.LoadDefaults(v)

	return fx.Module("deploygrid", fx.Provide(func() *viper.Viper {
		return v
	}),
		config.Index(),
		controller.Index(v),
		RepositoryIndex(v),
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
