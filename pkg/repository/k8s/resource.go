package k8s

import (
	"context"
	"time"

	"github.com/go-errors/errors"
	"github.com/rs/zerolog/log"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"

	"github.com/activatedio/deploygrid/pkg/k8s"
	"github.com/activatedio/deploygrid/pkg/repository"
)

type resourceRepository struct {
	client     dynamic.Interface
	discovery  discovery.ServerResourcesInterface // nil: watch without checking
	gvr        schema.GroupVersionResource
	toResource func(obj *unstructured.Unstructured) (*repository.Resource, error)
}

// servedCheckInterval is how often an unserved kind is re-checked, so a CRD
// installed later is picked up without a restart.
const servedCheckInterval = time.Minute

// served reports whether the API server offers the repository's resource.
func (c *resourceRepository) served() (bool, error) {
	list, err := c.discovery.ServerResourcesForGroupVersion(c.gvr.GroupVersion().String())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	for _, r := range list.APIResources {
		if r.Name == c.gvr.Resource {
			return true, nil
		}
	}
	return false, nil
}

// waitUntilServed blocks until the resource is served or ctx ends. A kind
// whose CRD is absent is logged once, not reported to the store: it is a
// normal state for a cluster that simply does not run that software.
func (c *resourceRepository) waitUntilServed(ctx context.Context) bool {
	logged := false
	for {
		ok, err := c.served()
		switch {
		case err != nil:
			log.Warn().Err(err).Str("resource", c.gvr.String()).Msg("discovery failed; retrying")
		case ok:
			if logged {
				log.Info().Str("resource", c.gvr.String()).Msg("resource is now served; starting watch")
			}
			return true
		case !logged:
			log.Info().Str("resource", c.gvr.String()).Msg("resource not served by this cluster; will check again periodically")
			logged = true
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(servedCheckInterval):
		}
	}
}

type unstructuredListWatcher struct {
	context  context.Context
	resource dynamic.ResourceInterface
}

func (u *unstructuredListWatcher) List(options metav1.ListOptions) (runtime.Object, error) {
	l, err := u.resource.List(u.context, options)
	if err != nil {
		k8s.ContextChannelErrorHandler(u.context, err, "error on list")
	}
	return l, err
}

func (u *unstructuredListWatcher) Watch(options metav1.ListOptions) (watch.Interface, error) {
	w, err := u.resource.Watch(u.context, options)
	if err != nil {
		k8s.ContextChannelErrorHandler(u.context, err, "error on watch")
	}
	return w, err
}

type resourceStoreAdapter struct {
	store      repository.ResourceStore
	toResource ToResource
}

func (c *resourceStoreAdapter) handleSingle(obj any, handler func(res *repository.Resource) error) error {

	if u, ok := obj.(*unstructured.Unstructured); ok {

		res, err := c.toResource(u)

		if err != nil {
			return err
		}

		return handler(res)
	}
	return errors.New("type is not unstructured")
}

func (c *resourceStoreAdapter) Add(obj interface{}) error {
	log.Info().Interface("adding", obj).Msgf("Adding object")
	return c.handleSingle(obj, c.store.Add)

}

func (c *resourceStoreAdapter) Update(obj interface{}) error {
	log.Info().Interface("updating", obj).Msgf("Updating object")
	return c.handleSingle(obj, c.store.Modify)
}

func (c *resourceStoreAdapter) Delete(obj interface{}) error {
	log.Info().Interface("deleting", obj).Msgf("Deleting object")
	return c.handleSingle(obj, c.store.Delete)
}

func (c *resourceStoreAdapter) Replace(i []interface{}, _ string) error {
	log.Info().Interface("replace", i).Msgf("Replace")

	var res []*repository.Resource

	for _, obj := range i {
		err := c.handleSingle(obj, func(_res *repository.Resource) error {
			res = append(res, _res)
			return nil
		})

		if err != nil {
			return err
		}
	}

	return c.store.Replace(res)

}

func (c *resourceStoreAdapter) Resync() error {
	log.Info().Msgf("Resync")
	return nil
}

func (c *resourceRepository) Watch(ctx context.Context, store repository.ResourceStore) {
	if c.discovery == nil {
		c.watch(ctx, store)
		return
	}
	go func() {
		if c.waitUntilServed(ctx) {
			c.watch(ctx, store)
		}
	}()
}

func (c *resourceRepository) watch(ctx context.Context, store repository.ResourceStore) {

	errorChan := make(chan k8s.RuntimeError)

	ctx = k8s.WithErrorReporter(ctx, errorChan)

	lw := &unstructuredListWatcher{
		context:  ctx,
		resource: c.client.Resource(c.gvr),
	}
	st := &resourceStoreAdapter{
		store:      store,
		toResource: c.toResource,
	}

	ref := cache.NewReflectorWithOptions(lw, &unstructured.Unstructured{}, st, cache.ReflectorOptions{
		MinWatchTimeout: 10 * time.Second,
	})

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case rt := <-errorChan:
				log.Error().Err(rt.Error).Msgf(rt.Message, rt.KeysAndValues...)
				store.Error(rt.Error)
			}
		}
	}()

	go func() {

		backoff := wait.Backoff{
			Steps:    10,
			Duration: 5 * time.Second,
			Factor:   5.0,
		}

		// RunWithContext returns when the watch ends; retry with backoff until
		// the context is cancelled. A plain `break` inside `select` only leaves
		// the select, so the loop condition must observe the context.
		for ctx.Err() == nil {
			err := wait.ExponentialBackoffWithContext(ctx, backoff, func(ctx context.Context) (done bool, err error) {
				ref.RunWithContext(ctx)
				return false, errors.New("watch ended")
			})
			if err != nil && ctx.Err() == nil {
				log.Error().Err(err).Msg("wait backoff")
			}
		}
	}()

}

type ToResource func(obj *unstructured.Unstructured) (*repository.Resource, error)

type ResourceRepositoryParams struct {
	Client dynamic.Interface
	// Discovery, when set, gates the watch on the resource being served.
	Discovery            discovery.ServerResourcesInterface
	GroupVersionResource schema.GroupVersionResource
	ToResource           ToResource
}

func NewResourceRepository(params ResourceRepositoryParams) repository.ResourceRepository {
	return &resourceRepository{
		client:     params.Client,
		discovery:  params.Discovery,
		gvr:        params.GroupVersionResource,
		toResource: params.ToResource,
	}
}
