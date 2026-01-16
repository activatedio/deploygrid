package k8s

import (
	"context"
	"time"

	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/generated/controllers/deploygrid.activated.io"
	deploygridcontroller "github.com/activatedio/deploygrid/pkg/generated/controllers/deploygrid.activated.io/v1alpha1"
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
)

type ControllersResult struct {
	fx.Out
	Controllers *Controllers
}

type Controllers struct {
	DeploygridSystems            deploygridcontroller.SystemController
	DeploygridSystemsCache       deploygridcontroller.SystemCache
	DeploygridMetadatas          deploygridcontroller.MetadataController
	DeploygridMetadatasCache     deploygridcontroller.MetadataCache
	DeploygridMetadataViews      deploygridcontroller.MetadataViewController
	DeploygridMetadataViewsCache deploygridcontroller.MetadataViewCache
}

func NewControllers(cfg *config.Main, cc clientcmd.ClientConfig, rc *rest.Config, lifecycle fx.Lifecycle) ControllersResult {

	appCtx, err := newContext(cc, cfg.Namespace)

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info().Msg("starting up app context")
			// We pass our own context to ensure it is started
			err := appCtx.start()
			log.Info().Err(err).Msg("started up app context")
			return err
		},
		OnStop: func(ctx context.Context) error {
			appCtx.stop()
			return nil
		},
	})

	if err != nil {
		panic(err)
	}

	log.Info().Msg("starting to reference controllers")
	dgs := appCtx.DG.System()
	dgm := appCtx.DG.Metadata()
	dgmv := appCtx.DG.MetadataView()
	log.Info().Msg("finished to referencing controllers")

	return ControllersResult{
		Controllers: &Controllers{
			DeploygridSystems:            dgs,
			DeploygridSystemsCache:       dgs.Cache(),
			DeploygridMetadatas:          dgm,
			DeploygridMetadatasCache:     dgm.Cache(),
			DeploygridMetadataViews:      dgmv,
			DeploygridMetadataViewsCache: dgmv.Cache(),
		},
	}
}

type appContext struct {
	cancel                  context.CancelFunc
	Client                  *rest.Config
	SharedControllerFactory controller.SharedControllerFactory

	DG deploygridcontroller.Interface

	starters []start.Starter
}

func (a *appContext) start() error {
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	return start.All(ctx, 50, a.starters...)
}

func (a *appContext) stop() {
	a.cancel()
	// Give time for logging to flush
	time.Sleep(2 * time.Second)
}

func controllerFactory(rest *rest.Config) (controller.SharedControllerFactory, error) {
	rateLimit := workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 60*time.Second)
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

func newContext(cfg clientcmd.ClientConfig, namespace string) (*appContext, error) {

	log.Info().Str("namespace", namespace).Msg("creating appCtx")

	cl, err := cfg.ClientConfig()
	if err != nil {
		return nil, err
	}
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
	dgv := dg.Deploygrid().V1alpha1()

	return &appContext{
		Client:                  cl,
		SharedControllerFactory: scf,

		DG: dgv,

		starters: []start.Starter{
			dg,
		},
	}, nil
}
