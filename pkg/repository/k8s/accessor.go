package k8s

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/sony/gobreaker/v2"
	"go.uber.org/fx"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"k8s.io/client-go/discovery"
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
	kinds        []config.ApplicationKindConfig
	lock         sync.Mutex
}

func (c *resourceRepositoryClusterAwareAccessor) ClusterNames(_ context.Context) []string {
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
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	r, err := cl.cb.Execute(func() (*repository.Resources, error) {

		cfg, err := restConfigFor(ctx, cl.config)
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
		disc, err := discovery.NewDiscoveryClientForConfig(cfg)
		if err != nil {
			return nil, err
		}

		return NewResources(client, disc, c.kinds), nil
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
	Sources        *config.SourcesConfig
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
		kinds:        params.Sources.ApplicationKinds,
	}
}

// restConfigFor builds the client configuration of a pull-mode cluster: the
// in-cluster service account, a kubeconfig file, or Address with a Google
// Application Default Credentials bearer token (GKE workload identity).
func restConfigFor(ctx context.Context, c *config.ClusterConfig) (*rest.Config, error) {
	switch {
	case c.EffectiveMode() == config.ClusterModeLocal:
		return rest.InClusterConfig()
	case c.Auth == config.ClusterAuthGoogle:
		ts, err := google.DefaultTokenSource(ctx, "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return nil, fmt.Errorf("cluster %s: google credentials: %w", c.Name, err)
		}
		cfg := &rest.Config{
			Host:            c.Address,
			TLSClientConfig: rest.TLSClientConfig{CAFile: c.CAFile},
		}
		cfg.Wrap(func(rt http.RoundTripper) http.RoundTripper {
			return &oauth2.Transport{Source: ts, Base: rt}
		})
		return cfg, nil
	default:
		rules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: c.KubeConfigPath}
		overrides := &clientcmd.ConfigOverrides{CurrentContext: c.ContextName}
		return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	}
}
