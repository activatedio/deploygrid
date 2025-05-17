package k8s

import (
	"context"
	"github.com/activatedio/deploygrid/pkg/k8s"
	"github.com/activatedio/deploygrid/pkg/repository"
	"github.com/go-errors/errors"
	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	"time"
)

type resourceRepository struct {
	client     dynamic.Interface
	gvr        schema.GroupVersionResource
	toResource func(obj *unstructured.Unstructured) (*repository.Resource, error)
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

	} else {
		return errors.New("type is not unstructured")
	}
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

func (c *resourceStoreAdapter) Replace(i []interface{}, s string) error {
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
				break
			case rt := <-errorChan:
				log.Error().Err(rt.Error).Msgf(rt.Message, rt.KeysAndValues)
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

		for {
			select {
			case <-ctx.Done():
				break
			default:
				err := wait.ExponentialBackoffWithContext(ctx, backoff, func(ctx context.Context) (done bool, err error) {
					ref.RunWithContext(ctx)
					return false, errors.New("watch ended")
				})
				if err != nil {
					log.Error().Err(err).Msg("wait backoff")
				}
			}
		}
	}()

}

type ToResource func(obj *unstructured.Unstructured) (*repository.Resource, error)

type ResourceRepositoryParams struct {
	Client               dynamic.Interface
	GroupVersionResource schema.GroupVersionResource
	ToResource           ToResource
}

func NewResourceRepository(params ResourceRepositoryParams) repository.ResourceRepository {
	return &resourceRepository{
		client:     params.Client,
		gvr:        params.GroupVersionResource,
		toResource: params.ToResource,
	}
}
