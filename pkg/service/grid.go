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

type gridService struct {
	catalog   Catalog
	accessor  repository.ClusterAwareAccessor[*repository.Resources]
	registry  *SourceRegistry
	connected map[string]bool
	lock      sync.Mutex
}

// updateClusters connects to any pull-mode cluster not yet watched (or whose
// last connection attempt failed) and starts its watches.
func (g *gridService) updateClusters(ctx context.Context) {

	g.lock.Lock()
	defer g.lock.Unlock()

	for _, cn := range g.accessor.ClusterNames(ctx) {
		if g.connected[cn] {
			continue
		}
		res, err := g.accessor.Get(ctx, cn)
		if err != nil {
			g.registry.SetError(cn, err)
			continue
		}
		// Watches live for the process lifetime, not the request.
		watchCtx := context.WithoutCancel(ctx)
		for kind, repo := range res.ByKind() {
			st, _ := g.registry.Store(cn, kind)
			repo.Watch(watchCtx, st)
		}
		g.registry.ClearError(cn)
		g.connected[cn] = true
	}
}

func (g *gridService) Init() {
	g.updateClusters(context.Background())
}

// snapshot refreshes pull-mode connections and returns the merged observed
// state of every cluster.
func (g *gridService) snapshot(ctx context.Context) (map[string]*store.StoreData, []string) {
	g.updateClusters(ctx)
	return g.registry.Snapshot()
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
	Registry *SourceRegistry
}

func NewGridService(params GridServiceParams) GridService {
	return &gridService{
		catalog:   params.Catalog,
		accessor:  params.Accessor,
		registry:  params.Registry,
		connected: map[string]bool{},
	}
}
