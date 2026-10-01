package service

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"

	apiinframux "github.com/activatedio/deploygrid/pkg/apiinfra/mux"
	"github.com/activatedio/deploygrid/pkg/apis/deploygrid.activated.io/v1alpha1"
	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/deploygrid"
	"github.com/activatedio/deploygrid/pkg/repository/k8s"
)

// DefaultSystemName is the System synthesised from v1 configuration when no
// control cluster is configured.
const DefaultSystemName = "default"

// FromSystemCR converts a System custom resource into the API model.
func FromSystemCR(in *v1alpha1.System) *deploygrid.System {
	out := &deploygrid.System{
		Name:        in.Name,
		DisplayName: in.Spec.DisplayName,
		Description: in.Spec.Description,
	}
	if out.DisplayName == "" {
		out.DisplayName = in.Name
	}
	for _, e := range in.Spec.Environments {
		dn := e.DisplayName
		if dn == "" {
			dn = e.Name
		}
		out.Environments = append(out.Environments, &deploygrid.Environment{Name: e.Name, DisplayName: dn})
	}
	for _, g := range in.Spec.Groups {
		dn := g.DisplayName
		if dn == "" {
			dn = g.Name
		}
		out.Groups = append(out.Groups, &deploygrid.Group{Name: g.Name, DisplayName: dn})
	}
	return out
}

// configSystemService serves a single System built from the v1 clusters
// configuration so the v2 API shape is available before a control cluster is
// configured.
type configSystemService struct {
	system *deploygrid.System
}

func (c *configSystemService) List(_ context.Context) ([]*deploygrid.System, error) {
	return []*deploygrid.System{c.system}, nil
}

func (c *configSystemService) Get(_ context.Context, name string) (*deploygrid.System, error) {
	if name != c.system.Name {
		return nil, fmt.Errorf("system %q: %w", name, apiinframux.ErrNotFound)
	}
	return c.system, nil
}

func NewConfigSystemService(clusters *config.ClustersConfig) SystemService {
	s := &deploygrid.System{
		Name:        DefaultSystemName,
		DisplayName: "Default",
		Description: "Synthesised from the clusters configuration",
	}
	for _, e := range clusters.Environments {
		s.Environments = append(s.Environments, &deploygrid.Environment{Name: e, DisplayName: e})
	}
	return &configSystemService{system: s}
}

// controlSystemService reads Systems from the control cluster's informer
// cache.
type controlSystemService struct {
	controllers *k8s.Controllers
	namespace   string
}

func (c *controlSystemService) List(_ context.Context) ([]*deploygrid.System, error) {
	items, err := c.controllers.SystemsCache.List(c.namespace, labels.Everything())
	if err != nil {
		return nil, err
	}
	res := make([]*deploygrid.System, 0, len(items))
	for _, item := range items {
		res = append(res, FromSystemCR(item))
	}
	slices.SortFunc(res, func(a, b *deploygrid.System) int {
		return strings.Compare(a.Name, b.Name)
	})
	return res, nil
}

func (c *controlSystemService) Get(_ context.Context, name string) (*deploygrid.System, error) {
	item, err := c.controllers.SystemsCache.Get(c.namespace, name)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("system %q: %w", name, apiinframux.ErrNotFound)
		}
		return nil, err
	}
	return FromSystemCR(item), nil
}

func NewControlSystemService(controllers *k8s.Controllers, control *config.ControlConfig) SystemService {
	return &controlSystemService{
		controllers: controllers,
		namespace:   control.Namespace,
	}
}
