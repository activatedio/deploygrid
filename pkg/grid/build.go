package grid

import (
	"bytes"
	"fmt"
	"path"
	"slices"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/activatedio/deploygrid/pkg/apis/deploygrid.activated.io/v1alpha1"
	"github.com/activatedio/deploygrid/pkg/deploygrid"
	"github.com/activatedio/deploygrid/pkg/repository"
	"github.com/activatedio/deploygrid/pkg/store"
)

// ClusterInfo is what the builder needs to know about a cluster: how other
// resources refer to it and how its namespaces map to environments.
type ClusterInfo struct {
	Name           string
	DisplayName    string
	Addresses      []string
	Environment    string
	NamespaceRules []v1alpha1.ClusterNamespaceRule
}

// ClusterInfoFromCR converts a Cluster custom resource.
func ClusterInfoFromCR(c *v1alpha1.Cluster) ClusterInfo {
	return ClusterInfo{
		Name:           c.Name,
		DisplayName:    c.Spec.DisplayName,
		Addresses:      c.Spec.Addresses,
		Environment:    c.Spec.Environment,
		NamespaceRules: c.Spec.NamespaceRules,
	}
}

// Input is everything needed to build one System's grid.
type Input struct {
	System *v1alpha1.System
	// Components may include components of other systems; they are filtered
	// by spec.system.
	Components []*v1alpha1.Component
	Clusters   []ClusterInfo
	// Observed holds the store snapshot of each cluster, keyed by cluster name.
	Observed map[string]*store.StoreData
	// Errors are cluster-level problems to surface on the grid.
	Errors []string
	Now    time.Time
}

// Result is the built grid plus derived data for other consumers.
type Result struct {
	Grid *deploygrid.Grid
	// Unassigned are top-level observed resources that matched no component.
	Unassigned []*deploygrid.Artifact
	// Statuses holds the per-environment cell summary for each declared
	// component, keyed by component name.
	Statuses map[string][]v1alpha1.ComponentEnvironmentStatus
}

// placement is one top-level observed resource (and the workloads it
// manages) resolved to a component and an environment.
type placement struct {
	component  string
	group      string
	env        string
	cluster    string
	namespace  string
	root       *repository.Resource
	rootHost   string // cluster the root object lives in
	children   []*repository.Resource
	discovered v1alpha1.ComponentKind
	display    string
}

type builder struct {
	in            Input
	envByKey      map[string]*v1alpha1.SystemEnvironment
	clusterByName map[string]*ClusterInfo
	clusterByAddr map[string]*ClusterInfo
	declared      map[string]*v1alpha1.Component
	warnings      map[string]struct{}
	legacyCount   int
}

// newBuilder indexes the declared inputs for fast lookups.
func newBuilder(in Input) *builder {
	b := &builder{
		in:            in,
		envByKey:      map[string]*v1alpha1.SystemEnvironment{},
		clusterByName: map[string]*ClusterInfo{},
		clusterByAddr: map[string]*ClusterInfo{},
		declared:      map[string]*v1alpha1.Component{},
		warnings:      map[string]struct{}{},
	}
	for i := range in.System.Spec.Environments {
		e := &in.System.Spec.Environments[i]
		b.envByKey[strings.ToLower(e.Name)] = e
		if e.DisplayName != "" {
			b.envByKey[strings.ToLower(e.DisplayName)] = e
		}
	}
	for i := range in.Clusters {
		c := &in.Clusters[i]
		b.clusterByName[c.Name] = c
		if c.DisplayName != "" {
			b.clusterByName[c.DisplayName] = c
		}
		for _, a := range c.Addresses {
			b.clusterByAddr[strings.TrimRight(a, "/")] = c
		}
	}
	for _, c := range in.Components {
		if c.Spec.System == in.System.Name {
			b.declared[c.Name] = c
		}
	}
	return b
}

// Build computes the grid for in.System.
func Build(in Input) *Result {
	b := newBuilder(in)
	placements := b.collect()
	rows, rowGroup, unassigned := b.assemble(placements)

	statuses := map[string][]v1alpha1.ComponentEnvironmentStatus{}
	for name, row := range rows {
		if _, ok := b.declared[name]; ok {
			statuses[name] = b.statuses(row)
		}
	}

	if b.legacyCount > 0 {
		b.warn(fmt.Sprintf("%d resource(s) use legacy deploygrid/* annotations; migrate to %s* labels", b.legacyCount, LabelPrefix))
	}

	g := &deploygrid.Grid{
		System: &deploygrid.System{
			Name:        in.System.Name,
			DisplayName: firstNonEmpty(in.System.Spec.DisplayName, in.System.Name),
			Description: in.System.Spec.Description,
		},
		Errors: in.Errors,
	}
	for _, e := range in.System.Spec.Environments {
		g.Environments = append(g.Environments, &deploygrid.Environment{Name: e.Name, DisplayName: firstNonEmpty(e.DisplayName, e.Name)})
	}
	g.System.Environments = g.Environments
	g.Groups = b.groups(rows, rowGroup)
	g.Warnings = b.sortedWarnings()

	sort.Slice(unassigned, func(i, j int) bool {
		a, c := unassigned[i], unassigned[j]
		return a.Cluster+"/"+a.Namespace+"/"+a.Name < c.Cluster+"/"+c.Namespace+"/"+c.Name
	})

	return &Result{Grid: g, Unassigned: unassigned, Statuses: statuses}
}

// assemble creates a row for every declared component and every discovered
// one, fills cells from placements and collects unassigned artifacts.
func (b *builder) assemble(placements []*placement) (rows map[string]*deploygrid.GridRow, rowGroup map[string]string, unassigned []*deploygrid.Artifact) {
	rows = map[string]*deploygrid.GridRow{}
	rowGroup = map[string]string{}
	cells := map[string]map[string][]*placement{} // component → env → placements

	for name, c := range b.declared {
		rows[name] = &deploygrid.GridRow{
			Component: &deploygrid.ComponentRef{
				Name:        name,
				DisplayName: firstNonEmpty(c.Spec.DisplayName, name),
				Description: c.Spec.Description,
				Kind:        string(c.Spec.Kind),
			},
			Cells: map[string]*deploygrid.Cell{},
		}
		rowGroup[name] = firstNonEmpty(c.Spec.Group, GroupDefault)
	}

	for _, p := range placements {
		if p.component == "" {
			unassigned = append(unassigned, b.artifact(p.root, p.rootHost, p.env))
			continue
		}
		if p.env == "" {
			b.warn(fmt.Sprintf("%s %s/%s matched component %q but its environment could not be resolved",
				p.root.Kind, p.rootHost, p.root.ObjectName, p.component))
			if _, declared := rows[p.component]; !declared {
				unassigned = append(unassigned, b.artifact(p.root, p.rootHost, ""))
			}
			continue
		}
		if _, ok := rows[p.component]; !ok {
			rows[p.component] = &deploygrid.GridRow{
				Component: &deploygrid.ComponentRef{
					Name:        p.component,
					DisplayName: firstNonEmpty(p.display, p.component),
					Kind:        string(p.discovered),
					Discovered:  true,
				},
				Cells: map[string]*deploygrid.Cell{},
			}
			rowGroup[p.component] = firstNonEmpty(p.group, GroupDefault)
		}
		if cells[p.component] == nil {
			cells[p.component] = map[string][]*placement{}
		}
		cells[p.component][p.env] = append(cells[p.component][p.env], p)
	}

	for name, row := range rows {
		for env, ps := range cells[name] {
			row.Cells[env] = b.cell(row.Component, b.declared[name], env, ps)
		}
	}
	return rows, rowGroup, unassigned
}

// collect turns observed resources into placements. Delivery resources
// (Applications) claim the workloads they manage on their destination
// cluster; remaining workloads stand on their own.
func (b *builder) collect() []*placement {
	claimed := map[string]map[string]bool{} // cluster → store name
	clusterNames := sortedKeys(b.in.Observed)

	var out []*placement
	for _, cn := range clusterNames {
		out = append(out, b.collectApplications(cn, claimed)...)
	}
	for _, cn := range clusterNames {
		out = append(out, b.collectWorkloads(cn, claimed[cn])...)
	}
	return out
}

func (b *builder) collectApplications(cn string, claimed map[string]map[string]bool) []*placement {
	var out []*placement
	data := b.in.Observed[cn]
	for _, name := range sortedKeys(data.Entries()) {
		r := data.Entries()[name]
		if r.Kind != repository.KindApplication {
			continue
		}
		p := &placement{root: r, rootHost: cn, cluster: cn}
		b.claimChildren(p, claimed)
		if b.resolve(p) {
			out = append(out, p)
		}
	}
	return out
}

// claimChildren attaches the workloads an Application manages on its
// destination cluster and marks them as claimed.
func (b *builder) claimChildren(p *placement, claimed map[string]map[string]bool) {
	r := p.root
	if r.Destination == nil {
		return
	}
	p.namespace = r.Destination.Namespace
	dest := b.resolveCluster(r.Destination)
	if dest == nil {
		if r.Destination.Server != "" || r.Destination.Name != "" {
			b.warn(fmt.Sprintf("application %s/%s targets unknown cluster %q", p.rootHost, r.ObjectName,
				firstNonEmpty(r.Destination.Name, r.Destination.Server)))
		}
		return
	}
	p.cluster = dest.Name
	dd, ok := b.in.Observed[dest.Name]
	if !ok {
		return
	}
	for _, childName := range dd.Children(r.Name) {
		child, ok := dd.Entries()[childName]
		if !ok {
			continue
		}
		p.children = append(p.children, child)
		if claimed[dest.Name] == nil {
			claimed[dest.Name] = map[string]bool{}
		}
		claimed[dest.Name][childName] = true
	}
}

func (b *builder) collectWorkloads(cn string, claimed map[string]bool) []*placement {
	var out []*placement
	data := b.in.Observed[cn]
	for _, name := range sortedKeys(data.Entries()) {
		r := data.Entries()[name]
		if r.Kind == repository.KindApplication || claimed[name] {
			continue
		}
		if SystemNamespaces[r.Namespace] && !labelled(r) {
			continue
		}
		p := &placement{root: r, rootHost: cn, cluster: cn, namespace: r.Namespace}
		if b.resolve(p) {
			out = append(out, p)
		}
	}
	return out
}

// labelled reports whether a resource carries any deploygrid identity.
func labelled(r *repository.Resource) bool {
	return r.Labels[LabelComponent] != "" || r.Annotations[LabelComponent] != "" ||
		r.Annotations[LegacyAnnotationName] != ""
}

// resolve fills component, environment and group. It returns false when the
// resource belongs to another System.
func (b *builder) resolve(p *placement) bool {
	r := p.root
	if sys := r.Labels[LabelSystem]; sys != "" && sys != b.in.System.Name {
		return false
	}
	p.component = b.resolveComponent(p)
	p.env = b.resolveEnvironment(p)
	p.group = firstNonEmpty(r.Annotations[AnnotationGroup], r.Annotations[LegacyAnnotationGroup])
	p.display = r.Annotations[AnnotationDisplayNam]
	p.discovered = discoveredKind(p)
	return true
}

func (b *builder) resolveComponent(p *placement) string {
	r := p.root
	switch {
	case r.Labels[LabelComponent] != "":
		return r.Labels[LabelComponent]
	case r.Annotations[LabelComponent] != "":
		return r.Annotations[LabelComponent]
	case r.Annotations[LegacyAnnotationName] != "":
		b.legacyCount++
		return r.Annotations[LegacyAnnotationName]
	default:
		return b.matchSelector(p)
	}
}

// resolveEnvironment returns the canonical environment name, or "" when it
// cannot be determined or is not declared in the System.
func (b *builder) resolveEnvironment(p *placement) string {
	r := p.root
	envKey := firstNonEmpty(r.Labels[LabelEnvironment], r.Annotations[LabelEnvironment])
	if envKey == "" {
		if legacy := r.Annotations[LegacyAnnotationEnvironment]; legacy != "" {
			envKey = legacy
			b.legacyCount++
		}
	}
	if envKey == "" {
		if c, ok := b.clusterByName[p.cluster]; ok {
			envKey = b.environmentFromCluster(c, p.namespace)
		}
	}
	if envKey == "" {
		return ""
	}
	if e, ok := b.envByKey[strings.ToLower(envKey)]; ok {
		return e.Name
	}
	b.warn(fmt.Sprintf("environment %q on %s %s/%s is not declared in system %q", envKey, r.Kind, p.rootHost, r.ObjectName, b.in.System.Name))
	return ""
}

// discoveredKind guesses the component kind of a row that no Component
// resource declares.
func discoveredKind(p *placement) v1alpha1.ComponentKind {
	switch {
	case p.root.Kind == repository.KindApplication && len(p.children) > 0:
		return v1alpha1.ComponentKindHelmChart
	case p.root.Kind == repository.KindApplication:
		return v1alpha1.ComponentKindArgoCDApplication
	default:
		return v1alpha1.ComponentKindContainer
	}
}

func (b *builder) environmentFromCluster(c *ClusterInfo, namespace string) string {
	for _, rule := range c.NamespaceRules {
		if ok, _ := path.Match(rule.Match, namespace); ok {
			return rule.Environment
		}
	}
	return c.Environment
}

func (b *builder) resolveCluster(loc *repository.ClusterLocation) *ClusterInfo {
	if loc.Name != "" {
		if c, ok := b.clusterByName[loc.Name]; ok {
			return c
		}
	}
	if loc.Server != "" {
		if c, ok := b.clusterByAddr[strings.TrimRight(loc.Server, "/")]; ok {
			return c
		}
	}
	return nil
}

// matchSelector returns the first declared component whose selector matches
// the placement's root resource.
func (b *builder) matchSelector(p *placement) string {
	names := sortedKeys(b.declared)
	for _, name := range names {
		sel := b.declared[name].Spec.Selector
		if len(sel.MatchLabels) == 0 && len(sel.Namespaces) == 0 && len(sel.Names) == 0 {
			continue
		}
		if !labelsMatch(sel.MatchLabels, p.root.Labels) {
			continue
		}
		if len(sel.Namespaces) > 0 && !slices.Contains(sel.Namespaces, p.namespace) {
			continue
		}
		if len(sel.Names) > 0 && !slices.Contains(sel.Names, p.root.ObjectName) {
			continue
		}
		return name
	}
	return ""
}

func labelsMatch(want, have map[string]string) bool {
	for k, v := range want {
		if have[k] != v {
			return false
		}
	}
	return true
}

// cell aggregates the placements of one component in one environment.
func (b *builder) cell(ref *deploygrid.ComponentRef, declared *v1alpha1.Component, env string, ps []*placement) *deploygrid.Cell {
	kind := v1alpha1.ComponentKind(ref.Kind)
	cell := &deploygrid.Cell{Health: repository.HealthUnknown}

	versions := map[string]bool{}
	healths := []string{}
	for i, p := range ps {
		actual := actualVersion(kind, ref.Name, p)
		if i == 0 {
			cell.Version = actual
			cell.DesiredVersion = p.root.DesiredVersion
			cell.Cluster = p.cluster
			cell.Namespace = p.namespace
		}
		if actual != "" {
			versions[actual] = true
		}
		healths = append(healths, p.root.Health)
		cell.Artifacts = append(cell.Artifacts, b.artifact(p.root, p.rootHost, ""))
		for _, c := range p.children {
			healths = append(healths, c.Health)
			cell.Artifacts = append(cell.Artifacts, b.artifact(c, p.cluster, ""))
		}
		cell.Hosts = append(cell.Hosts, p.root.Hosts...)
	}
	if cell.Version == "" {
		for v := range versions {
			cell.Version = v
			break
		}
	}
	cell.Inconsistent = len(versions) > 1
	cell.Drifted = cell.DesiredVersion != "" && cell.Version != "" && cell.DesiredVersion != cell.Version
	cell.Health = rollupHealth(healths)
	slices.Sort(cell.Hosts)
	cell.Hosts = slices.Compact(cell.Hosts)

	if declared != nil {
		cell.Links = b.links(declared, env, cell)
	}
	return cell
}

// observedVersions summarises the versions found on a placement.
type observedVersions struct {
	chart     string // chart version stamped on workloads
	container string // first container version
	named     string // container (or workload) named after the component
}

func scanVersions(component string, p *placement) observedVersions {
	v := observedVersions{chart: p.root.ChartVersion}
	for _, r := range append([]*repository.Resource{p.root}, p.children...) {
		if v.chart == "" {
			v.chart = r.ChartVersion
		}
		for _, c := range r.Components {
			if c.Kind != repository.VersionKindContainer {
				continue
			}
			if v.container == "" {
				v.container = c.Version
			}
			if v.named == "" && (c.Name == component || r.ObjectName == component) {
				v.named = c.Version
			}
		}
	}
	return v
}

// actualVersion picks the running version of a placement for the component
// kind.
func actualVersion(kind v1alpha1.ComponentKind, component string, p *placement) string {
	v := scanVersions(component, p)
	switch kind {
	case v1alpha1.ComponentKindHelmChart:
		return firstNonEmpty(v.chart, p.root.SyncRevision)
	case v1alpha1.ComponentKindContainer:
		return firstNonEmpty(v.named, v.container)
	case v1alpha1.ComponentKindArgoCDApplication:
		return firstNonEmpty(p.root.SyncRevision, v.chart)
	case v1alpha1.ComponentKindService, v1alpha1.ComponentKindCustom:
		return firstNonEmpty(v.chart, v.named, v.container, p.root.SyncRevision)
	default:
		return firstNonEmpty(v.chart, v.named, v.container, p.root.SyncRevision)
	}
}

var healthRank = map[string]int{
	repository.HealthDegraded:    0,
	repository.HealthProgressing: 1,
	repository.HealthUnknown:     2,
	repository.HealthHealthy:     3,
}

// rollupHealth returns the worst health among the inputs; Unknown only when
// nothing is known.
func rollupHealth(hs []string) string {
	best := ""
	for _, h := range hs {
		if h == "" {
			h = repository.HealthUnknown
		}
		if best == "" || healthRank[h] < healthRank[best] {
			best = h
		}
	}
	if best == "" {
		return repository.HealthUnknown
	}
	return best
}

func (b *builder) artifact(r *repository.Resource, cluster, env string) *deploygrid.Artifact {
	a := &deploygrid.Artifact{
		Kind:        r.Kind,
		Cluster:     cluster,
		Namespace:   r.Namespace,
		Name:        r.ObjectName,
		Health:      r.Health,
		Environment: env,
	}
	for _, c := range r.Components {
		a.Versions = append(a.Versions, &deploygrid.Version{Name: c.Name, Kind: c.Kind, Value: c.Version, Image: c.Image})
	}
	if r.ChartVersion != "" {
		a.Versions = append(a.Versions, &deploygrid.Version{Name: "chart", Kind: repository.VersionKindChart, Value: r.ChartVersion})
	}
	if r.DesiredVersion != "" {
		a.Versions = append(a.Versions, &deploygrid.Version{Name: "desired", Kind: repository.VersionKindRevision, Value: r.DesiredVersion})
	}
	return a
}

type linkContext struct {
	System      string
	Component   string
	Environment string
	Cluster     string
	Namespace   string
	Version     string
	Hosts       []string
}

func (b *builder) links(c *v1alpha1.Component, env string, cell *deploygrid.Cell) []*deploygrid.Link {
	if len(c.Spec.Links) == 0 {
		return nil
	}
	ctx := linkContext{
		System:      b.in.System.Name,
		Component:   c.Name,
		Environment: env,
		Cluster:     cell.Cluster,
		Namespace:   cell.Namespace,
		Version:     cell.Version,
		Hosts:       cell.Hosts,
	}
	out := make([]*deploygrid.Link, 0, len(c.Spec.Links))
	for _, l := range c.Spec.Links {
		t, err := template.New(l.Name).Option("missingkey=zero").Parse(l.URLTemplate)
		if err != nil {
			b.warn(fmt.Sprintf("component %s link %q: %v", c.Name, l.Name, err))
			continue
		}
		var buf bytes.Buffer
		if err := t.Execute(&buf, ctx); err != nil {
			b.warn(fmt.Sprintf("component %s link %q: %v", c.Name, l.Name, err))
			continue
		}
		out = append(out, &deploygrid.Link{Name: l.Name, URL: buf.String()})
	}
	return out
}

func (b *builder) statuses(row *deploygrid.GridRow) []v1alpha1.ComponentEnvironmentStatus {
	out := make([]v1alpha1.ComponentEnvironmentStatus, 0, len(row.Cells))
	for _, e := range b.in.System.Spec.Environments {
		cell, ok := row.Cells[e.Name]
		if !ok {
			continue
		}
		out = append(out, v1alpha1.ComponentEnvironmentStatus{
			Environment:    e.Name,
			Version:        cell.Version,
			DesiredVersion: cell.DesiredVersion,
			Cluster:        cell.Cluster,
			Namespace:      cell.Namespace,
			Health:         cell.Health,
			Hosts:          cell.Hosts,
		})
	}
	return out
}

// groups orders rows into declared groups first, then the remaining groups
// alphabetically, nesting rows under their declared parent.
func (b *builder) groups(rows map[string]*deploygrid.GridRow, rowGroup map[string]string) []*deploygrid.GridGroup {
	byGroup := map[string][]*deploygrid.GridRow{}
	for name, row := range rows {
		if b.nested(name, rows) {
			continue
		}
		g := rowGroup[name]
		byGroup[g] = append(byGroup[g], row)
	}
	for _, row := range rows {
		b.sortRows(row.Children)
	}

	out := make([]*deploygrid.GridGroup, 0, len(byGroup))
	seen := map[string]bool{}
	for _, g := range b.in.System.Spec.Groups {
		seen[g.Name] = true
		if rs, ok := byGroup[g.Name]; ok {
			b.sortRows(rs)
			out = append(out, &deploygrid.GridGroup{Name: g.Name, DisplayName: firstNonEmpty(g.DisplayName, g.Name), Rows: rs})
		}
	}
	for _, g := range sortedKeys(byGroup) {
		if seen[g] {
			continue
		}
		b.sortRows(byGroup[g])
		out = append(out, &deploygrid.GridGroup{Name: g, DisplayName: g, Rows: byGroup[g]})
	}
	return out
}

// nested attaches the row to its declared parent and reports whether it did.
func (b *builder) nested(name string, rows map[string]*deploygrid.GridRow) bool {
	c, ok := b.declared[name]
	if !ok || c.Spec.Parent == "" || c.Spec.Parent == name {
		return false
	}
	parent, ok := rows[c.Spec.Parent]
	if !ok {
		return false
	}
	parent.Children = append(parent.Children, rows[name])
	return true
}

func (b *builder) order(name string) int {
	if c, ok := b.declared[name]; ok && c.Spec.Order != nil {
		return int(*c.Spec.Order)
	}
	return 1 << 30
}

func (b *builder) sortRows(rs []*deploygrid.GridRow) {
	sort.SliceStable(rs, func(i, j int) bool {
		oi, oj := b.order(rs[i].Component.Name), b.order(rs[j].Component.Name)
		if oi != oj {
			return oi < oj
		}
		return rs[i].Component.Name < rs[j].Component.Name
	})
}

func (b *builder) warn(msg string) {
	b.warnings[msg] = struct{}{}
}

func (b *builder) sortedWarnings() []string {
	return sortedKeys(b.warnings)
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
