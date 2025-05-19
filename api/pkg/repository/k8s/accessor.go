package k8s

import (
	"context"
	"fmt"
	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/repository"
	"github.com/go-errors/errors"
	"github.com/sony/gobreaker/v2"
	"go.uber.org/fx"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sync"
)

type cluster struct {
	config *config.ClusterConfig
	cb     *gobreaker.CircuitBreaker[*repository.Resources]
}

type resourceRepositoryClusterAwareAccessor struct {
	clusterNames []string
	clusters     map[string]cluster
	repositories map[string]*repository.Resources
	lock         sync.Mutex
}

func (c *resourceRepositoryClusterAwareAccessor) ClusterNames(ctx context.Context) []string {
	return c.clusterNames
}

func (c *resourceRepositoryClusterAwareAccessor) Get(ctx context.Context, clusterName string) (*repository.Resources, error) {

	c.lock.Lock()
	defer c.lock.Unlock()

	if r, ok := c.repositories[clusterName]; ok {
		return r, nil
	}

	cl, ok := c.clusters[clusterName]

	if !ok {
		return nil, errors.New(fmt.Sprintf("cluster not found: %s", clusterName))
	}

	r, err := cl.cb.Execute(func() (*repository.Resources, error) {

		var cfg *rest.Config
		var err error

		if cl.config.Local {
			cfg, err = rest.InClusterConfig()
		} else {
			cfg, err = clientcmd.BuildConfigFromFlags("", cl.config.KubeConfigPath)
		}

		if err != nil {
			return nil, err
		}

		client, err := dynamic.NewForConfig(cfg)

		if err != nil {
			return nil, err
		}

		return NewResources(client), nil
	})

	if err != nil {
		return nil, err
	}

	c.repositories[clusterName] = r

	return r, nil
}

type ResourceRepositoryClusterAwareAccessorParams struct {
	fx.In
	ClustersConfig *config.ClustersConfig
}

func NewResourceRepositoryClusterAwareAccessor(params ResourceRepositoryClusterAwareAccessorParams) repository.ClusterAwareAccessor[*repository.Resources] {

	var clusterNames []string
	clusters := map[string]cluster{}

	for _, c := range params.ClustersConfig.Clusters {
		clusterNames = append(clusterNames, c.Name)
		clusters[c.Name] = cluster{
			config: &c,
			cb: gobreaker.NewCircuitBreaker[*repository.Resources](gobreaker.Settings{
				Name: "factory",
			}),
		}
	}

	return &resourceRepositoryClusterAwareAccessor{
		clusterNames: clusterNames,
		clusters:     clusters,
		repositories: map[string]*repository.Resources{},
	}
}
