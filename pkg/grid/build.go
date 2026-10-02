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

// DiscoveredComponent describes a row that no Component resource declares,
// with what was learned about it from observed resources.
type DiscoveredComponent struct {
	Name        string
	DisplayName string
	Group       string
	Kind        v1alpha1.ComponentKind
}

// Result is the built grid plus derived data for other consumers.
type Result struct {
	Grid *deploygrid.Grid
	// Unassigned are top-level observed resources that matched no component.
	Unassigned []*deploygrid.Artifact
	// Statuses holds the per-environment cell summary for each declared
	// component, keyed by component name.
	Statuses map[string][]v1alpha1.ComponentEnvironmentStatus
	// Discovered lists rows without a Component resource, sorted by name,
	// so the control plane can materialise them.
	Discovered []DiscoveredComponent
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
	groupByKey    map[string]string // lower(name|displayName) → declared group name
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
		groupByKey:    map[string]string{},
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
	for _, g := range in.System.Spec.Groups {
		b.groupByKey[strings.ToLower(g.Name)] = g.Name
		if g.DisplayName != "" {
			b.groupByKey[strings.ToLower(g.DisplayName)] = g.Name
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
	discovered := make([]DiscoveredComponent, 0, len(rows)-len(b.declared))
	for _, name := range sortedKeys(rows) {
		row := rows[name]
		if _, ok := b.declared[name]; ok {
			statuses[name] = b.statuses(row)
			continue
		}
		discovered = append(discovered, DiscoveredComponent{
			Name:        name,
			DisplayName: row.Component.DisplayName,
			Group:       rowGroup[name],
			Kind:        v1alpha1.ComponentKind(row.Component.Kind),
		})
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

	return &Result{Grid: g, Unassigned: unassigned, Statuses: statuses, Discovered: discovered}
}

// assemble creates a row for every declared component and every discovered
// one, fills cells from placements and collects unassigned artifacts.
func (b *builder) assemble(placements []*placement) (rows map[string]*deploygrid.GridRow, rowGroup map[string]string, unassigned []*deploygrid.Artifact) {
	rows, rowGroup = b.declaredRows()
	cells := map[string]map[string][]*placement{} // component → env → placements

	var hostOnly []*placement
	for _, p := range placements {
		if p.root.Kind == repository.KindIngress {
			// Ingresses never form rows; they add hosts to the cell of the
			// component they belong to.
			if p.component != "" && p.env != "" {
				hostOnly = append(hostOnly, p)
			}
			continue
		}
		if a := b.place(p, rows, rowGroup, cells); a != nil {
			unassigned = append(unassigned, a)
		}
	}

	for name, row := range rows {
		for env, ps := range cells[name] {
			row.Cells[env] = b.cell(row.Component, b.declared[name], env, ps)
		}
	}
	b.addHosts(rows, hostOnly)
	return rows, rowGroup, unassigned
}

// place files a placement under its row and environment, creating a
// discovered row when needed. It returns an artifact when the placement
// cannot be shown on the grid.
func (b *builder) place(p *placement, rows map[string]*deploygrid.GridRow, rowGroup map[string]string, cells map[string]map[string][]*placement) *deploygrid.Artifact {
	if p.component == "" {
		return b.artifact(p.root, p.rootHost, p.env)
	}
	if p.env == "" {
		b.warn(fmt.Sprintf("%s %s/%s matched component %q but its environment could not be resolved",
			p.root.Kind, p.rootHost, p.root.ObjectName, p.component))
		if _, declared := rows[p.component]; !declared {
			return b.artifact(p.root, p.rootHost, "")
		}
		return nil
	}
	if _, ok := rows[p.component]; !ok {
		rows[p.component] = discoveredRow(p)
		rowGroup[p.component] = b.group(p.group)
	}
	if cells[p.component] == nil {
		cells[p.component] = map[string][]*placement{}
	}
	cells[p.component][p.env] = append(cells[p.component][p.env], p)
	return nil
}

// declaredRows creates an empty row for every Component declared in the
// system.
func (b *builder) declaredRows() (map[string]*deploygrid.GridRow, map[string]string) {
	rows := map[string]*deploygrid.GridRow{}
	rowGroup := map[string]string{}
	for name, c := range b.declared {
		rows[name] = &deploygrid.GridRow{
			Component: &deploygrid.ComponentRef{
				Name:        name,
				DisplayName: firstNonEmpty(c.Spec.DisplayName, name),
				Description: c.Spec.Description,
				Kind:        string(c.Spec.Kind),
				// materialised by discovery and not yet touched by a person
				Discovered: c.Status.Discovered && c.Generation <= 1,
			},
			Cells: map[string]*deploygrid.Cell{},
		}
		rowGroup[name] = b.group(c.Spec.Group)
	}
	return rows, rowGroup
}

// group canonicalises a group name against the System's declared groups
// (matching name or display name case-insensitively); unknown groups are
// kept as written and an empty group falls back to GroupDefault.
func (b *builder) group(name string) string {
	if name == "" {
		return GroupDefault
	}
	if canonical, ok := b.groupByKey[strings.ToLower(name)]; ok {
		return canonical
	}
	return name
}

// discoveredRow creates a row for a component no resource declares.
func discoveredRow(p *placement) *deploygrid.GridRow {
	return &deploygrid.GridRow{
		Component: &deploygrid.ComponentRef{
			Name:        p.component,
			DisplayName: firstNonEmpty(p.display, p.component),
			Kind:        string(p.discovered),
			Discovered:  true,
		},
		Cells: map[string]*deploygrid.Cell{},
	}
}

// addHosts merges standalone ingress hosts into existing cells and
// re-renders their links.
func (b *builder) addHosts(rows map[string]*deploygrid.GridRow, hostOnly []*placement) {
	for _, p := range hostOnly {
		row, ok := rows[p.component]
		if !ok {
			continue
		}
		cell, ok := row.Cells[p.env]
		if !ok {
			continue
		}
		cell.Hosts = append(cell.Hosts, p.root.Hosts...)
		slices.Sort(cell.Hosts)
		cell.Hosts = slices.Compact(cell.Hosts)
		if c, declared := b.declared[p.component]; declared {
			cell.Links = b.links(c, p.env, cell)
		}
	}
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
		switch {
		case r.Kind == repository.KindApplication:
			p := &placement{root: r, rootHost: cn, cluster: cn}
			b.claimChildren(p, claimed)
			if b.resolve(p) {
				out = append(out, p)
			}
		case repository.IsOperatorApplication(r.Kind):
			// an operator reconciles in its own cluster: the workloads it
			// owns live next to the custom resource
			p := &placement{root: r, rootHost: cn, cluster: cn, namespace: r.Namespace}
			b.claimOwned(p, data, claimed)
			if b.resolve(p) {
				out = append(out, p)
			}
		}
	}
	return out
}

// claimOwned attaches the workloads whose Parent is the given resource in
// the same cluster store.
func (b *builder) claimOwned(p *placement, data *store.StoreData, claimed map[string]map[string]bool) {
	for _, childName := range data.Children(p.root.Name) {
		child, ok := data.Entries()[childName]
		if !ok {
			continue
		}
		p.children = append(p.children, child)
		if claimed[p.cluster] == nil {
			claimed[p.cluster] = map[string]bool{}
		}
		claimed[p.cluster][childName] = true
	}
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
		if r.Kind == repository.KindApplication || repository.IsOperatorApplication(r.Kind) || claimed[name] {
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
		return firstNonEmpty(b.matchSelector(p), r.DefaultComponent)
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
		envKey = r.DefaultEnvironment
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
	case repository.IsOperatorApplication(p.root.Kind):
		return v1alpha1.ComponentKindOperatorApplication
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

	agg := b.aggregate(kind, ref.Name, ps, cell)
	versions, healths, mixed := agg.versions, agg.healths, agg.mixed
	if cell.Version == "" {
		for v := range versions {
			cell.Version = v
			break
		}
	}
	cell.Inconsistent = len(versions) > 1 || mixed
	cell.Drifted = cell.DesiredVersion != "" && cell.Version != "" && cell.DesiredVersion != cell.Version
	cell.Health = rollupHealth(healths)
	slices.Sort(cell.Hosts)
	cell.Hosts = slices.Compact(cell.Hosts)

	if declared != nil {
		cell.Links = b.links(declared, env, cell)
	}
	return cell
}

// cellAggregate is what the placements of one cell contribute.
type cellAggregate struct {
	versions map[string]bool
	healths  []string
	mixed    bool
}

// aggregate folds every placement into the cell: the first one fixes the
// headline fields, all of them contribute versions, health, artifacts and
// hosts.
func (b *builder) aggregate(kind v1alpha1.ComponentKind, component string, ps []*placement, cell *deploygrid.Cell) cellAggregate {
	agg := cellAggregate{versions: map[string]bool{}}
	for i, p := range ps {
		actual := actualVersion(kind, component, p)
		if i == 0 {
			cell.Version = actual
			cell.DesiredVersion = p.root.DesiredVersion
			cell.Cluster = p.cluster
			cell.Namespace = p.namespace
		}
		if !slices.Contains(cell.Clusters, p.cluster) {
			cell.Clusters = append(cell.Clusters, p.cluster)
		}
		if actual != "" {
			agg.versions[actual] = true
		}
		if sv := scanVersions(component, p); kind == v1alpha1.ComponentKindOperatorApplication && sv.mixed() {
			agg.mixed = true
		}
		agg.healths = append(agg.healths, placementHealth(p)...)
		cell.Artifacts = append(cell.Artifacts, b.placementArtifacts(p)...)
		cell.Hosts = append(cell.Hosts, placementHosts(p)...)
	}
	return agg
}

// placementHealth lists the health of the root and its workloads; ingresses
// carry no health.
func placementHealth(p *placement) []string {
	out := []string{p.root.Health}
	for _, c := range p.children {
		if c.Kind != repository.KindIngress {
			out = append(out, c.Health)
		}
	}
	return out
}

func (b *builder) placementArtifacts(p *placement) []*deploygrid.Artifact {
	out := []*deploygrid.Artifact{b.artifact(p.root, p.rootHost, "")}
	for _, c := range p.children {
		out = append(out, b.artifact(c, p.cluster, ""))
	}
	return out
}

func placementHosts(p *placement) []string {
	out := append([]string(nil), p.root.Hosts...)
	for _, c := range p.children {
		out = append(out, c.Hosts...)
	}
	return out
}

// observedVersions summarises the versions found on a placement.
type observedVersions struct {
	chart     string   // chart version stamped on workloads
	stamped   string   // app.kubernetes.io/version stamped on workloads
	container string   // first container version
	named     string   // container (or workload) named after the component
	distinct  []string // distinct container versions across root and children
}

func scanVersions(component string, p *placement) observedVersions {
	v := observedVersions{chart: p.root.ChartVersion, stamped: p.root.StampedVersion}
	seen := map[string]bool{}
	for _, r := range append([]*repository.Resource{p.root}, p.children...) {
		v.chart = firstNonEmpty(v.chart, r.ChartVersion)
		v.stamped = firstNonEmpty(v.stamped, r.StampedVersion)
		v.scanContainers(component, r, seen)
	}
	return v
}

func (v *observedVersions) scanContainers(component string, r *repository.Resource, seen map[string]bool) {
	for _, c := range r.Components {
		if c.Kind != repository.VersionKindContainer {
			continue
		}
		if !seen[c.Version] {
			seen[c.Version] = true
			v.distinct = append(v.distinct, c.Version)
		}
		v.container = firstNonEmpty(v.container, c.Version)
		if v.named == "" && (c.Name == component || r.ObjectName == component) {
			v.named = c.Version
		}
	}
}

// mixed reports whether an operator application runs more than one version
// across its workloads, which the cell reports as inconsistent.
func (v *observedVersions) mixed() bool {
	return len(v.distinct) > 1
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
	case v1alpha1.ComponentKindOperatorApplication:
		// what the operator reports as running, else what its workloads run
		// (when they agree), else the version they were stamped with
		if p.root.SyncRevision != "" {
			return p.root.SyncRevision
		}
		if len(v.distinct) == 1 {
			return v.distinct[0]
		}
		return firstNonEmpty(v.stamped, v.container)
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
