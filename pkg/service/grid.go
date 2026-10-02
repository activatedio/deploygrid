package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/fx"

	apiinframux "github.com/activatedio/deploygrid/pkg/apiinfra/mux"
	"github.com/activatedio/deploygrid/pkg/apis/deploygrid.activated.io/v1alpha1"
	"github.com/activatedio/deploygrid/pkg/deploygrid"
	"github.com/activatedio/deploygrid/pkg/grid"
	"github.com/activatedio/deploygrid/pkg/repository"
	"github.com/activatedio/deploygrid/pkg/store"
)

type resourcesOrError struct {
	Resources *repository.Resources
	Error     error
}

type clusterStores struct {
	applications *store.Store
	deployments  *store.Store
}

type gridService struct {
	catalog  Catalog
	accessor repository.ClusterAwareAccessor[*repository.Resources]
	clusters map[string]resourcesOrError
	stores   map[string]clusterStores
	lock     sync.RWMutex
}

// updateClusters connects to any cluster not yet watched (or whose last
// connection attempt failed) and starts its watches.
func (g *gridService) updateClusters(ctx context.Context) {

	g.lock.Lock()
	defer g.lock.Unlock()

	for _, cn := range g.accessor.ClusterNames(ctx) {
		if roe, ok := g.clusters[cn]; ok && roe.Error == nil {
			continue
		}
		res, err := g.accessor.Get(ctx, cn)
		if err != nil {
			g.clusters[cn] = resourcesOrError{Error: err}
			continue
		}
		st := clusterStores{
			applications: store.NewStore(),
			deployments:  store.NewStore(),
		}
		// Watches live for the process lifetime, not the request.
		res.Applications.Watch(context.WithoutCancel(ctx), st.applications)
		res.Deployment.Watch(context.WithoutCancel(ctx), st.deployments)
		g.stores[cn] = st
		g.clusters[cn] = resourcesOrError{Resources: res}
	}
}

func (g *gridService) Init() {
	g.updateClusters(context.Background())
}

// snapshot merges the application and deployment snapshots of every cluster
// and collects cluster-level errors.
func (g *gridService) snapshot(ctx context.Context) (map[string]*store.StoreData, []string) {
	g.updateClusters(ctx)

	g.lock.RLock()
	defer g.lock.RUnlock()

	var errs []string
	for k, v := range g.clusters {
		if v.Error != nil {
			errs = append(errs, fmt.Sprintf("[Connect to cluster %s]: %s", k, v.Error.Error()))
		}
	}

	data := map[string]*store.StoreData{}
	for k, v := range g.stores {
		merged := store.NewStoreData()
		for _, st := range []*store.Store{v.applications, v.deployments} {
			d, err := st.GetData()
			if err != nil {
				errs = append(errs, fmt.Sprintf("[cluster %s]: %s", k, err.Error()))
			}
			merged.AddAll(d)
		}
		data[k] = merged
	}
	return data, errs
}

func (g *gridService) build(ctx context.Context, system *v1alpha1.System, observed map[string]*store.StoreData, errs []string) (*grid.Result, error) {
	components, err := g.catalog.Components()
	if err != nil {
		return nil, err
	}
	clusters, err := g.catalog.Clusters()
	if err != nil {
		return nil, err
	}
	_ = ctx
	return grid.Build(grid.Input{
		System:     system,
		Components: components,
		Clusters:   clusters,
		Observed:   observed,
		Errors:     errs,
		Now:        time.Now(),
	}), nil
}

func (g *gridService) buildSystem(ctx context.Context, system string) (*grid.Result, error) {
	sys, err := g.catalog.System(system)
	if err != nil {
		return nil, err
	}
	observed, errs := g.snapshot(ctx)
	return g.build(ctx, sys, observed, errs)
}

func (g *gridService) BuildAll(ctx context.Context) (map[string]*grid.Result, error) {
	systems, err := g.catalog.Systems()
	if err != nil {
		return nil, err
	}
	observed, errs := g.snapshot(ctx)
	out := make(map[string]*grid.Result, len(systems))
	for _, sys := range systems {
		res, err := g.build(ctx, sys, observed, errs)
		if err != nil {
			return nil, err
		}
		out[sys.Name] = res
	}
	return out, nil
}

func (g *gridService) Grid(ctx context.Context, system string) (*deploygrid.Grid, error) {
	res, err := g.buildSystem(ctx, system)
	if err != nil {
		return nil, err
	}
	return res.Grid, nil
}

func flattenRows(groups []*deploygrid.GridGroup) []*deploygrid.GridRow {
	var out []*deploygrid.GridRow
	var walk func(rows []*deploygrid.GridRow)
	walk = func(rows []*deploygrid.GridRow) {
		for _, r := range rows {
			out = append(out, r)
			walk(r.Children)
		}
	}
	for _, grp := range groups {
		walk(grp.Rows)
	}
	return out
}

func (g *gridService) Rows(ctx context.Context, system string) ([]*deploygrid.GridRow, error) {
	res, err := g.buildSystem(ctx, system)
	if err != nil {
		return nil, err
	}
	return flattenRows(res.Grid.Groups), nil
}

func (g *gridService) Row(ctx context.Context, system, component string) (*deploygrid.GridRow, error) {
	rows, err := g.Rows(ctx, system)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		if r.Component.Name == component {
			return r, nil
		}
	}
	return nil, fmt.Errorf("component %q in system %q: %w", component, system, apiinframux.ErrNotFound)
}

func (g *gridService) Unassigned(ctx context.Context, system string) ([]*deploygrid.Artifact, error) {
	res, err := g.buildSystem(ctx, system)
	if err != nil {
		return nil, err
	}
	return res.Unassigned, nil
}

type GridServiceParams struct {
	fx.In
	Catalog  Catalog
	Accessor repository.ClusterAwareAccessor[*repository.Resources]
}

func NewGridService(params GridServiceParams) GridService {
	return &gridService{
		catalog:  params.Catalog,
		accessor: params.Accessor,
		clusters: map[string]resourcesOrError{},
		stores:   map[string]clusterStores{},
	}
}
