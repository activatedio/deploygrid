package service

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	apiinframux "github.com/activatedio/deploygrid/pkg/apiinfra/mux"
	"github.com/activatedio/deploygrid/pkg/apis/deploygrid.activated.io/v1alpha1"
	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/grid"
	"github.com/activatedio/deploygrid/pkg/repository/k8s"
)

// DefaultSystemName is the System synthesised from v1 configuration when no
// control cluster is configured.
const DefaultSystemName = "default"

// InClusterAddress is how Argo CD refers to the cluster it runs in.
const InClusterAddress = "https://kubernetes.default.svc"

// clusterInfosFromConfig converts the v1 cluster list. These supply the
// connection and the address; Cluster custom resources add environment
// mapping on top.
func clusterInfosFromConfig(cfg *config.ClustersConfig) []grid.ClusterInfo {
	out := make([]grid.ClusterInfo, 0, len(cfg.Clusters))
	for _, c := range cfg.Clusters {
		addr := c.Address
		if c.EffectiveMode() == config.ClusterModeLocal && addr == "" {
			addr = InClusterAddress
		}
		ci := grid.ClusterInfo{Name: c.Name}
		if addr != "" {
			ci.Addresses = []string{addr}
		}
		out = append(out, ci)
	}
	return out
}

// mergeClusters overlays Cluster custom resources on the configured
// clusters, matching by name.
func mergeClusters(configured []grid.ClusterInfo, crs []*v1alpha1.Cluster) []grid.ClusterInfo {
	byName := map[string]*grid.ClusterInfo{}
	out := make([]grid.ClusterInfo, 0, len(configured)+len(crs))
	for _, c := range configured {
		out = append(out, c)
		byName[c.Name] = &out[len(out)-1]
	}
	for _, cr := range crs {
		ci := grid.ClusterInfoFromCR(cr)
		if existing, ok := byName[cr.Name]; ok {
			for _, a := range existing.Addresses {
				if !slices.Contains(ci.Addresses, a) {
					ci.Addresses = append(ci.Addresses, a)
				}
			}
			*existing = ci
			continue
		}
		out = append(out, ci)
	}
	return out
}

// configCatalog serves a single System built from the v1 clusters
// configuration so the v2 API shape is available before a control cluster is
// configured.
type configCatalog struct {
	system   *v1alpha1.System
	clusters []grid.ClusterInfo
}

func (c *configCatalog) Systems() ([]*v1alpha1.System, error) {
	return []*v1alpha1.System{c.system}, nil
}

func (c *configCatalog) System(name string) (*v1alpha1.System, error) {
	if name != c.system.Name {
		return nil, fmt.Errorf("system %q: %w", name, apiinframux.ErrNotFound)
	}
	return c.system, nil
}

func (c *configCatalog) Components() ([]*v1alpha1.Component, error) {
	return nil, nil
}

func (c *configCatalog) Clusters() ([]grid.ClusterInfo, error) {
	return c.clusters, nil
}

func (c *configCatalog) Configurations() ([]*v1alpha1.Configuration, error) {
	return nil, nil
}

func (c *configCatalog) Version() uint64 { return 0 }

func (c *configCatalog) ConfigurationViews() ([]*v1alpha1.ConfigurationView, error) {
	return nil, nil
}

func NewConfigCatalog(clusters *config.ClustersConfig) Catalog {
	s := &v1alpha1.System{
		ObjectMeta: metav1.ObjectMeta{Name: DefaultSystemName},
		Spec: v1alpha1.SystemSpec{
			DisplayName: "Default",
			Description: "Synthesised from the clusters configuration",
		},
	}
	for _, e := range clusters.Environments {
		s.Spec.Environments = append(s.Spec.Environments, v1alpha1.SystemEnvironment{Name: e, DisplayName: e})
	}
	return &configCatalog{system: s, clusters: clusterInfosFromConfig(clusters)}
}

// controlCatalog reads custom resources from the control cluster's informer
// caches.
type controlCatalog struct {
	controllers *k8s.Controllers
	namespace   string
	configured  []grid.ClusterInfo
	version     atomic.Uint64
}

func (c *controlCatalog) Version() uint64 { return c.version.Load() }

// watch bumps the version on every informer event of the given controllers.
// Handlers are registered before the factory starts, so the initial list
// counts too.
func (c *controlCatalog) watch(ctx context.Context) {
	bump := func(_ string, obj runtime.Object) (runtime.Object, error) {
		c.version.Add(1)
		return obj, nil
	}
	const name = "deploygrid-catalog-version"
	c.controllers.Systems.AddGenericHandler(ctx, name, bump)
	c.controllers.Components.AddGenericHandler(ctx, name, bump)
	c.controllers.Clusters.AddGenericHandler(ctx, name, bump)
	c.controllers.Configurations.AddGenericHandler(ctx, name, bump)
	c.controllers.ConfigurationViews.AddGenericHandler(ctx, name, bump)
}

func (c *controlCatalog) Systems() ([]*v1alpha1.System, error) {
	items, err := c.controllers.SystemsCache.List(c.namespace, labels.Everything())
	if err != nil {
		return nil, err
	}
	slices.SortFunc(items, func(a, b *v1alpha1.System) int { return strings.Compare(a.Name, b.Name) })
	return items, nil
}

func (c *controlCatalog) System(name string) (*v1alpha1.System, error) {
	item, err := c.controllers.SystemsCache.Get(c.namespace, name)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("system %q: %w", name, apiinframux.ErrNotFound)
		}
		return nil, err
	}
	return item, nil
}

func (c *controlCatalog) Components() ([]*v1alpha1.Component, error) {
	items, err := c.controllers.ComponentsCache.List(c.namespace, labels.Everything())
	if err != nil {
		return nil, err
	}
	slices.SortFunc(items, func(a, b *v1alpha1.Component) int { return strings.Compare(a.Name, b.Name) })
	return items, nil
}

func (c *controlCatalog) Clusters() ([]grid.ClusterInfo, error) {
	crs, err := c.controllers.ClustersCache.List(c.namespace, labels.Everything())
	if err != nil {
		return nil, err
	}
	slices.SortFunc(crs, func(a, b *v1alpha1.Cluster) int { return strings.Compare(a.Name, b.Name) })
	return mergeClusters(c.configured, crs), nil
}

func (c *controlCatalog) Configurations() ([]*v1alpha1.Configuration, error) {
	items, err := c.controllers.ConfigurationsCache.List(c.namespace, labels.Everything())
	if err != nil {
		return nil, err
	}
	slices.SortFunc(items, func(a, b *v1alpha1.Configuration) int { return strings.Compare(a.Name, b.Name) })
	return items, nil
}

func (c *controlCatalog) ConfigurationViews() ([]*v1alpha1.ConfigurationView, error) {
	items, err := c.controllers.ConfigurationViewsCache.List(c.namespace, labels.Everything())
	if err != nil {
		return nil, err
	}
	slices.SortFunc(items, func(a, b *v1alpha1.ConfigurationView) int { return strings.Compare(a.Name, b.Name) })
	return items, nil
}

func NewControlCatalog(controllers *k8s.Controllers, control *config.ControlConfig, clusters *config.ClustersConfig) Catalog {
	c := &controlCatalog{
		controllers: controllers,
		namespace:   control.Namespace,
		configured:  clusterInfosFromConfig(clusters),
	}
	c.watch(context.Background())
	return c
}
