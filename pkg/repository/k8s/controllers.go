package k8s

import (
	"context"
	"time"

	"github.com/rancher/lasso/pkg/cache"
	"github.com/rancher/lasso/pkg/client"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/wrangler/v3/pkg/generic"
	"github.com/rancher/wrangler/v3/pkg/ratelimit"
	"github.com/rancher/wrangler/v3/pkg/start"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"

	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/generated/controllers/deploygrid.activated.io"
	deploygridcontroller "github.com/activatedio/deploygrid/pkg/generated/controllers/deploygrid.activated.io/v1alpha1"
)

// Controllers exposes the wrangler controllers (clients) and informer caches
// for every deploygrid custom resource in the control cluster.
//
// The caches are obtained before the factory starts: lasso creates an
// informer lazily on first use, and only informers that exist when Start runs
// are started and synced.
type Controllers struct {
	Systems                 deploygridcontroller.SystemController
	SystemsCache            deploygridcontroller.SystemCache
	Components              deploygridcontroller.ComponentController
	ComponentsCache         deploygridcontroller.ComponentCache
	Clusters                deploygridcontroller.ClusterController
	ClustersCache           deploygridcontroller.ClusterCache
	Configurations          deploygridcontroller.ConfigurationController
	ConfigurationsCache     deploygridcontroller.ConfigurationCache
	ConfigurationViews      deploygridcontroller.ConfigurationViewController
	ConfigurationViewsCache deploygridcontroller.ConfigurationViewCache
}

// NewControllers connects to the control cluster described by cfg and starts
// the shared informers with the fx lifecycle. Caches are synced before OnStart
// returns, so services may read them as soon as the application is running.
func NewControllers(cfg *config.ControlConfig, lifecycle fx.Lifecycle) (*Controllers, error) {

	restConfig, err := controlRestConfig(cfg)
	if err != nil {
		return nil, err
	}

	appCtx, err := newContext(restConfig, cfg.Namespace)
	if err != nil {
		return nil, err
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info().Str("namespace", cfg.Namespace).Msg("starting control cluster informers")
			return appCtx.start(ctx)
		},
		OnStop: func(_ context.Context) error {
			appCtx.stop()
			return nil
		},
	})

	systems := appCtx.DG.System()
	components := appCtx.DG.Component()
	clusters := appCtx.DG.Cluster()
	configurations := appCtx.DG.Configuration()
	views := appCtx.DG.ConfigurationView()

	return &Controllers{
		Systems:                 systems,
		SystemsCache:            systems.Cache(),
		Components:              components,
		ComponentsCache:         components.Cache(),
		Clusters:                clusters,
		ClustersCache:           clusters.Cache(),
		Configurations:          configurations,
		ConfigurationsCache:     configurations.Cache(),
		ConfigurationViews:      views,
		ConfigurationViewsCache: views.Cache(),
	}, nil
}

func controlRestConfig(cfg *config.ControlConfig) (*rest.Config, error) {
	if cfg.KubeConfigPath == "" {
		return rest.InClusterConfig()
	}
	rules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: cfg.KubeConfigPath}
	overrides := &clientcmd.ConfigOverrides{CurrentContext: cfg.Context}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
}

type appContext struct {
	cancel                  context.CancelFunc
	SharedControllerFactory controller.SharedControllerFactory

	DG deploygridcontroller.Interface

	starters []start.Starter
}

// start runs the informers for the life of the application. The fx start
// context is cancelled once OnStart returns, so only its values are kept.
func (a *appContext) start(parent context.Context) error {
	ctx, cancel := context.WithCancel(context.WithoutCancel(parent))
	a.cancel = cancel
	return start.All(ctx, 50, a.starters...)
}

func (a *appContext) stop() {
	if a.cancel != nil {
		a.cancel()
	}
}

func controllerFactory(rest *rest.Config) (controller.SharedControllerFactory, error) {
	rateLimit := workqueue.NewTypedItemExponentialFailureRateLimiter[any](5*time.Millisecond, 60*time.Second)
	clientFactory, err := client.NewSharedClientFactory(rest, nil)
	if err != nil {
		return nil, err
	}

	cacheFactory := cache.NewSharedCachedFactory(clientFactory, nil)
	return controller.NewSharedControllerFactory(cacheFactory, &controller.SharedControllerFactoryOptions{
		DefaultRateLimiter: rateLimit,
		DefaultWorkers:     50,
	}), nil
}

func newContext(cl *rest.Config, namespace string) (*appContext, error) {

	cl.RateLimiter = ratelimit.None

	scf, err := controllerFactory(cl)
	if err != nil {
		return nil, err
	}

	dg, err := deploygrid.NewFactoryFromConfigWithOptions(cl, &generic.FactoryOptions{
		SharedControllerFactory: scf,
		Namespace:               namespace,
	})
	if err != nil {
		return nil, err
	}

	return &appContext{
		SharedControllerFactory: scf,
		DG:                      dg.Deploygrid().V1alpha1(),
		starters:                []start.Starter{dg},
	}, nil
}
