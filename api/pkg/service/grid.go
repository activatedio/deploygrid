package service

import (
	"context"
	"fmt"
	"github.com/activatedio/deploygrid/pkg/apiinfra/util"
	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/deploygrid"
	"github.com/activatedio/deploygrid/pkg/repository"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
	"slices"
	"strings"
	"sync"
)

type resourcesOrError struct {
	Resources *repository.Resources
	Error     error
}

type gridService struct {
	clusters     map[string]resourcesOrError
	accessor     repository.ClusterAwareAccessor[*repository.Resources]
	stores       map[string]stores
	lock         sync.RWMutex
	addressMap   map[string]string
	environments []string
}

func (g *gridService) updateClusters(ctx context.Context) {

	g.lock.Lock()
	defer g.lock.Unlock()

	// Refresh the map - TODO - make this only as needed
	for _, cn := range g.accessor.ClusterNames(ctx) {
		if roe, ok := g.clusters[cn]; !ok || roe.Error != nil {
			res, err := g.accessor.Get(ctx, cn)
			if err != nil {
				g.clusters[cn] = resourcesOrError{
					Error: err,
				}
			} else {
				st := stores{
					applications: NewStore(),
					deployments:  NewStore(),
				}
				res.Applications.Watch(ctx, st.applications)
				res.Deployment.Watch(ctx, st.deployments)
				g.stores[cn] = st
				g.clusters[cn] = resourcesOrError{
					Resources: res,
				}
			}
		}
	}

}

type gridNode struct {
	simpleName    string
	displayName   string
	pathElement   string
	version       string
	componentType string
	children      []gridNode
}

type gridCell struct {
	nodeMap map[string]gridNode
	nodes   []gridNode
}

func (g *gridCell) mapNodes() {

	var doMap func(prefix string, in map[string]gridNode, nodes []gridNode)
	doMap = func(prefix string, in map[string]gridNode, nodes []gridNode) {
		for _, n := range nodes {
			path := prefix + n.pathElement
			in[path] = n
			doMap(path+"/", in, n.children)
		}
	}

	nodeMap := map[string]gridNode{}
	doMap("", nodeMap, g.nodes)
	g.nodeMap = nodeMap
}

func newGridCell() *gridCell {
	return &gridCell{}
}

type gridRow struct {
	level int
	group string
	cells map[string]*gridCell
	name  string
}

func (g *gridRow) expand() []*deploygrid.Component {

	type gridNodeEnvs struct {
		// first is used to set names of the component
		first *gridNode
		nodes map[string]*gridNode
	}

	type withParent struct {
		parentPath string
		node       *[]*deploygrid.Component
	}

	pathMap := map[string]bool{}
	// Map path to environment
	rowEnvMap := map[string]gridNodeEnvs{}

	for ck, cv := range g.cells {
		cv.mapNodes()
		for p, n := range cv.nodeMap {
			pathMap[p] = true

			if envs, ok := rowEnvMap[p]; !ok {
				envs = gridNodeEnvs{
					first: &n,
					nodes: map[string]*gridNode{
						ck: &n,
					},
				}
				rowEnvMap[p] = envs
			} else {
				envs.nodes[ck] = &n
			}

		}
	}

	var paths []string

	for k, _ := range pathMap {
		paths = append(paths, k)
	}

	slices.Sort(paths)

	st := util.NewStack[withParent]()

	var res []*deploygrid.Component

	cur := &withParent{
		parentPath: "",
		node:       &res,
	}
	st.Push(cur)

	for _, path := range paths {

		envs := rowEnvMap[path]
		comp := &deploygrid.Component{
			Name:          envs.first.simpleName,
			ComponentType: envs.first.componentType,
			Deployments: func() map[string]*deploygrid.Deployment {

				ds := make(map[string]*deploygrid.Deployment)

				for k, v := range envs.nodes {
					ds[k] = &deploygrid.Deployment{
						Version: v.version,
					}
				}

				return ds
			}(),
		}

		for cur.parentPath != "" && !strings.HasPrefix(path, cur.parentPath+"/") {
			cur = st.Pop()
		}

		*cur.node = append(*cur.node, comp)
		st.Push(cur)
		cur = &withParent{
			parentPath: path,
			node:       &comp.Children,
		}
		st.Push(cur)
	}

	return res
}

func newGridRow(group string, name string) *gridRow {
	return &gridRow{
		name:  name,
		group: group,
		cells: map[string]*gridCell{},
	}
}

func buildGrid(grid *deploygrid.Grid, rows map[string]*gridRow, columns []string) {

	envs := map[string]bool{}

	var sorted []*gridRow

	for _, rv := range rows {
		sorted = append(sorted, rv)
	}

	slices.SortFunc(sorted, func(a, b *gridRow) int {
		return strings.Compare(a.name, b.name)
	})

	grouped := map[string]*deploygrid.Component{}

	for _, rv := range sorted {
		for ck, _ := range rv.cells {
			envs[ck] = true
		}

		groupName := rv.group
		if groupName == "" {
			groupName = "Default"
		}
		comps := rv.expand()

		if grp, ok := grouped[groupName]; ok {
			grp.Children = append(grp.Children, comps...)
		} else {
			grp = &deploygrid.Component{
				Name:          groupName,
				ComponentType: "Group",
				Children:      comps,
			}
			grouped[groupName] = grp
		}

	}

	for _, c := range columns {
		grid.Environments = append(grid.Environments, &deploygrid.Environment{
			Name: c,
		})
	}

	var comps []*deploygrid.Component

	for _, v := range grouped {
		comps = append(comps, v)
	}

	slices.SortFunc(comps, func(a, b *deploygrid.Component) int {
		return strings.Compare(a.Name, b.Name)
	})

	grid.Components = comps
}

func (g *gridService) Init() {
	g.updateClusters(context.Background())
}

func (g *gridService) Get(ctx context.Context) (*deploygrid.Grid, error) {

	// We do this first before we acquire a read lock
	g.updateClusters(context.Background())

	g.lock.RLock()
	defer g.lock.RUnlock()

	res := &deploygrid.Grid{}

	for k, v := range g.clusters {
		if v.Error != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("[Connect to cluster %s]: %s ", k, v.Error.Error()))
		}
	}

	data := map[string]*StoreData{}

	for k, v := range g.stores {
		_data := NewStoreData()
		var d *StoreData
		var err error
		d, err = v.applications.GetData()
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("[cluster %s]: %s ", k, err.Error()))
		}
		_data.addAll(d)
		d, err = v.deployments.GetData()
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("[cluster %s]: %s ", k, err.Error()))
		}
		_data.addAll(d)
		data[k] = _data
	}

	rowMap := map[string]*gridRow{}

	// Check recursion limit
	// TODO - first encountered group defines the group container - this should change
	for _, v := range data {
		for _, e := range v.entries {
			group := e.Annotations[AnnotationDeployGridGroup]

			if group == "" {
				group = GroupNoGroup
			}

			name, nameOk := e.Annotations[AnnotationDeployGridName]
			envs, envsOk := e.Annotations[AnnotationDeployGridEnvironment]

			if nameOk && envsOk && e.Parent == "" {

				row, ok := rowMap[name]
				if !ok {
					row = newGridRow(group, name)
					rowMap[name] = row
				}

				for _, env := range strings.Split(envs, ",") {
					cell, ok := row.cells[env]
					if !ok {
						cell = newGridCell()
						row.cells[env] = cell
					}

					err := g.buildNodes(ctx, data, &cell.nodes, &e.Components)
					if err != nil {
						return nil, err
					}
				}

			}
		}
	}

	buildGrid(res, rowMap, g.environments)

	return res, nil
}

func (g *gridService) buildNodes(ctx context.Context, data map[string]*StoreData, nodes *[]gridNode, comps *[]repository.Component) error {
	for _, c := range *comps {
		n := gridNode{
			simpleName:    c.SimpleName,
			displayName:   c.DisplayName,
			pathElement:   c.PathElement,
			version:       c.Version,
			componentType: c.Type,
		}
		for _, cl := range c.ChildrenLocation {
			clusterName := g.addressMap[cl.Server]
			if clusterName == "" {
				log.Warn().Str("address", cl.Server).Msg("cluster name not found for address")
				continue
			}
			if cd, ok := data[clusterName]; ok {
				if childs, ok := cd.parentMap[c.Name]; ok {
					for childName, _ := range childs {
						if res, ok := cd.entries[childName]; ok {
							err := g.buildNodes(ctx, data, &n.children, &res.Components)
							if err != nil {
								return err
							}
						}
					}
				}
			}
		}
		*nodes = append(*nodes, n)
	}

	return nil
}

type GridServiceParams struct {
	fx.In
	ClustersConfig *config.ClustersConfig
	Accessor       repository.ClusterAwareAccessor[*repository.Resources]
}

func NewGridService(params GridServiceParams) GridService {

	addressMap := map[string]string{}

	for _, cc := range params.ClustersConfig.Clusters {
		if cc.Local {
			addressMap["https://kubernetes.default.svc"] = cc.Name
		} else {
			addressMap[cc.Address] = cc.Name
		}
	}
	return &gridService{
		accessor:     params.Accessor,
		addressMap:   addressMap,
		environments: params.ClustersConfig.Environments,
		clusters:     map[string]resourcesOrError{},
		stores:       map[string]stores{},
	}
}
