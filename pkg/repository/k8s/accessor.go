package k8s

import (
	"context"
	"fmt"
	"sync"

	"github.com/sony/gobreaker/v2"
	"go.uber.org/fx"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/repository"
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

func (c *resourceRepositoryClusterAwareAccessor) ClusterNames(_ context.Context) []string {
	return c.clusterNames
}

func (c *resourceRepositoryClusterAwareAccessor) Get(_ context.Context, clusterName string) (*repository.Resources, error) {

	c.lock.Lock()
	defer c.lock.Unlock()

	if r, ok := c.repositories[clusterName]; ok {
		return r, nil
	}

	cl, ok := c.clusters[clusterName]

	if !ok {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	r, err := cl.cb.Execute(func() (*repository.Resources, error) {

		var cfg *rest.Config
		var err error

		if cl.config.EffectiveMode() == config.ClusterModeLocal {
			cfg, err = rest.InClusterConfig()
		} else {
			cfg, err = clientcmd.BuildConfigFromFlags("", cl.config.KubeConfigPath)
		}

		if err != nil {
			return nil, err
		}

		if cl.config.InsecureSkipTLSVerify {
			// Development only: kubeconfigs written by tools such as kind are
			// valid, so this should never be needed against a real cluster.
			cfg.CAData = nil
			cfg.CAFile = ""
			cfg.Insecure = true
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

	clusterNames := make([]string, 0, len(params.ClustersConfig.Clusters))
	clusters := map[string]cluster{}

	for i := range params.ClustersConfig.Clusters {
		c := &params.ClustersConfig.Clusters[i]
		if c.EffectiveMode() == config.ClusterModeAgent {
			// Observations arrive from a collector; nothing to watch here.
			continue
		}
		clusterNames = append(clusterNames, c.Name)
		clusters[c.Name] = cluster{
			config: c,
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
