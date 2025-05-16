package k8s

import (
	"context"
	"fmt"
	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/repository"
	"github.com/go-errors/errors"
	"github.com/sony/gobreaker/v2"
	"go.uber.org/fx"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"sync"
)

type cluster struct {
	config *config.ClusterConfig
	cb     *gobreaker.CircuitBreaker[repository.ResourceRepository]
}

type resourceRepositoryClusterAwareAccessor struct {
	clusters     map[string]cluster
	repositoires map[string]repository.ResourceRepository
	lock         sync.Mutex
}

func (c *resourceRepositoryClusterAwareAccessor) ClusterNames(ctx context.Context) []string {
	return nil
}

func (c *resourceRepositoryClusterAwareAccessor) Get(ctx context.Context, clusterName string) (repository.ResourceRepository, error) {

	c.lock.Lock()
	defer c.lock.Unlock()

	if r, ok := c.repositoires[clusterName]; ok {
		return r, nil
	}

	cl, ok := c.clusters[clusterName]

	if !ok {
		return nil, errors.New(fmt.Sprintf("cluster not found: %s", clusterName))
	}

	r, err := cl.cb.Execute(func() (repository.ResourceRepository, error) {

		client, err := dynamic.NewForConfig(&rest.Config{
			Host: cl.config.Address,
		})

		if err != nil {
			return nil, err
		}

		return NewResourceRepository(ResourceRepositoryParams{
			Client:               client,
			GroupVersionResource: schema.GroupVersionResource{},
			ToResource:           nil,
		}), nil
	})

	if err != nil {
		return nil, err
	}

	c.repositoires[clusterName] = r

	return r, nil
}

type ResourceRepositoryClusterAwareAccessorParams struct {
	fx.In
	ClustersConfig *config.ClustersConfig
}

func NewResourceRepositoryClusterAwareAccessor(params ResourceRepositoryClusterAwareAccessorParams) repository.ClusterAwareAccessor[repository.ResourceRepository] {

	clusters := map[string]cluster{}

	for _, c := range params.ClustersConfig.Clusters {
		clusters[c.Name] = cluster{
			config: &c,
			cb: gobreaker.NewCircuitBreaker[repository.ResourceRepository](gobreaker.Settings{
				Name: "factory",
			}),
		}
	}

	return &resourceRepositoryClusterAwareAccessor{
		clusters: clusters,
	}
}
