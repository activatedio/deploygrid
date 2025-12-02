package k8s

import "go.uber.org/fx"

func Index() fx.Option {
	return fx.Module("deploygrid.repository.k8s",
		fx.Provide(NewResourceRepositoryClusterAwareAccessor),
	)
}
