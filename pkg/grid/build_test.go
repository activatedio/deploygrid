package grid_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/activatedio/deploygrid/pkg/apis/deploygrid.activated.io/v1alpha1"
	"github.com/activatedio/deploygrid/pkg/deploygrid"
	"github.com/activatedio/deploygrid/pkg/grid"
	"github.com/activatedio/deploygrid/pkg/repository"
	"github.com/activatedio/deploygrid/pkg/store"
)

const (
	opsCluster  = "ops"
	app1Cluster = "app-1"
	app2Cluster = "app-2"
	app1Addr    = "https://127.0.0.1:6444"
	app2Addr    = "https://127.0.0.1:6445"
)

func application(name, component, env, group, server, namespace, chart, target string) *repository.Resource {
	return &repository.Resource{
		Name:       "applications/" + name,
		Kind:       repository.KindApplication,
		Namespace:  "argocd",
		ObjectName: name,
		Annotations: map[string]string{
			grid.LegacyAnnotationName:        component,
			grid.LegacyAnnotationEnvironment: env,
			grid.LegacyAnnotationGroup:       group,
		},
		DesiredVersion: target,
		Health:         repository.HealthUnknown,
		Destination:    &repository.ClusterLocation{Server: server, Namespace: namespace},
		Labels:         map[string]string{"chart": chart},
	}
}

func deployment(namespace, name, instance, chartVersion, image string, labels map[string]string) *repository.Resource {
	l := map[string]string{
		"app.kubernetes.io/managed-by": "Helm",
		"app.kubernetes.io/instance":   instance,
	}
	for k, v := range labels {
		l[k] = v
	}
	parent := ""
	if instance != "" {
		parent = "applications/" + instance
	}
	return &repository.Resource{
		Name:         "namespaces/" + namespace + "/deployments/" + name,
		Kind:         repository.KindDeployment,
		Namespace:    namespace,
		ObjectName:   name,
		Parent:       parent,
		Labels:       l,
		Components:   []repository.Component{{Name: "nginx", Kind: repository.VersionKindContainer, Version: image, Image: "nginx:" + image}},
		ChartVersion: chartVersion,
		Health:       repository.HealthHealthy,
	}
}

func ingress(namespace, name, instance string, labels map[string]string, hosts ...string) *repository.Resource {
	l := map[string]string{}
	parent := ""
	if instance != "" {
		l["app.kubernetes.io/managed-by"] = "Helm"
		l["app.kubernetes.io/instance"] = instance
		parent = "applications/" + instance
	}
	for k, v := range labels {
		l[k] = v
	}
	return &repository.Resource{
		Name:       "namespaces/" + namespace + "/ingresses/" + name,
		Kind:       repository.KindIngress,
		Namespace:  namespace,
		ObjectName: name,
		Parent:     parent,
		Labels:     l,
		Hosts:      hosts,
	}
}

func snapshot(t *testing.T, rs ...*repository.Resource) *store.StoreData {
	t.Helper()
	s := store.NewStore()
	require.NoError(t, s.Replace(rs))
	d, err := s.GetData()
	require.NoError(t, err)
	return d
}

func fixture(t *testing.T) grid.Input {
	t.Helper()
	order := int32(1)
	return grid.Input{
		System: &v1alpha1.System{
			ObjectMeta: metav1.ObjectMeta{Name: "apps"},
			Spec: v1alpha1.SystemSpec{
				Environments: []v1alpha1.SystemEnvironment{
					{Name: "dev", DisplayName: "Dev"},
					{Name: "qa", DisplayName: "QA"},
					{Name: "stage", DisplayName: "Stage"},
				},
				Groups: []v1alpha1.SystemGroup{{Name: "Apps", DisplayName: "Applications"}},
			},
		},
		Components: []*v1alpha1.Component{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "app-a"},
				Spec: v1alpha1.ComponentSpec{
					System: "apps", DisplayName: "App A", Group: "Apps", Kind: v1alpha1.ComponentKindHelmChart, Order: &order,
					Links: []v1alpha1.ComponentLink{{Name: "Argo CD", URLTemplate: "https://argo/{{ .Environment }}-{{ .Component }}"}},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "worker"},
				Spec: v1alpha1.ComponentSpec{
					System: "apps", Group: "Batch", Kind: v1alpha1.ComponentKindContainer,
					Selector: v1alpha1.ComponentSelector{MatchLabels: map[string]string{"role": "worker"}},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "other-system-comp"},
				Spec:       v1alpha1.ComponentSpec{System: "other"},
			},
		},
		Clusters: []grid.ClusterInfo{
			{Name: app1Cluster, Addresses: []string{app1Addr}, NamespaceRules: []v1alpha1.ClusterNamespaceRule{
				{Match: "dev-*", Environment: "dev"}, {Match: "qa-*", Environment: "qa"},
			}},
			{Name: app2Cluster, Addresses: []string{app2Addr}, Environment: "stage"},
		},
		Observed: map[string]*store.StoreData{
			opsCluster: snapshot(t,
				application("dev-app-a", "app-a", "Dev", "Apps", app1Addr, "dev-app-a", "app-a", "1.16.1"),
				application("qa-app-a", "app-a", "QA", "Apps", app1Addr, "qa-app-a", "app-a", "1.17.0"),
				application("stage-app-a", "app-a", "Stage", "Apps", app2Addr, "stage-app-a", "app-a", "1.16.1"),
				application("dev-app-b", "app-b", "Dev", "APPS", app1Addr, "dev-app-b", "app-b", "2.0.0"), // group matched case-insensitively
				application("lost", "app-c", "Prod", "Apps", "https://nowhere", "lost", "app-c", "1.0.0"),
			),
			app1Cluster: snapshot(t,
				deployment("dev-app-a", "app", "dev-app-a", "1.16.1", "1.14.2", nil),
				deployment("dev-app-a", "app-side", "dev-app-a", "1.16.1", "1.14.2", nil),
				deployment("qa-app-a", "app", "qa-app-a", "1.16.1", "1.14.2", nil), // behind desired 1.17.0
				deployment("dev-app-b", "app", "dev-app-b", "2.0.0", "1.25.0", nil),
				deployment("qa-batch", "worker", "", "", "3.1.0", map[string]string{"role": "worker"}),
				ingress("dev-app-a", "app", "dev-app-a", nil, "app-a.dev.example.com"),
				ingress("qa-batch", "worker", "", map[string]string{grid.LabelComponent: "worker"}, "worker.qa.example.com", "worker-alt.qa.example.com"),
				ingress("dev-misc", "stray", "", nil, "stray.example.com"),
				deployment("dev-misc", "orphan", "", "", "0.1.0", nil),
				deployment("kube-system", "coredns", "", "", "1.11.1", nil),
				deployment("kube-system", "metrics", "", "", "0.7.0", map[string]string{grid.LabelComponent: "metrics", grid.LabelEnvironment: "dev"}),
			),
			app2Cluster: snapshot(t,
				deployment("stage-app-a", "app", "stage-app-a", "1.16.1", "1.14.2", nil),
			),
		},
		Errors: []string{"[Connect to cluster app-3]: refused"},
		Now:    time.Now(),
	}
}

func TestBuild(t *testing.T) {
	res := grid.Build(fixture(t))
	g := res.Grid
	a := assert.New(t)

	// columns and errors pass through
	require.Len(t, g.Environments, 3)
	a.Equal("Dev", g.Environments[0].DisplayName)
	a.Equal([]string{"[Connect to cluster app-3]: refused"}, g.Errors)

	// groups: declared first, then alphabetical
	require.Len(t, g.Groups, 3)
	a.Equal("Apps", g.Groups[0].Name)
	a.Equal("Applications", g.Groups[0].DisplayName)
	a.Equal("Batch", g.Groups[1].Name)

	apps := g.Groups[0]
	require.Len(t, apps.Rows, 2)
	appA, appB := apps.Rows[0], apps.Rows[1]
	a.Equal("app-a", appA.Component.Name)
	a.Equal("App A", appA.Component.DisplayName)
	a.False(appA.Component.Discovered)
	a.Equal("app-b", appB.Component.Name)
	a.True(appB.Component.Discovered, "app-b has no Component resource")
	a.Equal(string(v1alpha1.ComponentKindHelmChart), appB.Component.Kind)

	// app-a dev: chart version from workloads, desired from the Application
	dev := appA.Cells["dev"]
	require.NotNil(t, dev)
	a.Equal("1.16.1", dev.Version)
	a.Equal("1.16.1", dev.DesiredVersion)
	a.False(dev.Drifted)
	a.Equal(app1Cluster, dev.Cluster)
	a.Equal("dev-app-a", dev.Namespace)
	a.Equal(repository.HealthUnknown, dev.Health, "the Application has unknown health, which dominates")
	a.Equal([]string{"app-a.dev.example.com"}, dev.Hosts, "hosts come from the Helm-managed ingress")
	require.Len(t, dev.Artifacts, 4, "application, two deployments and the ingress")
	a.Equal("nginx", dev.Artifacts[1].Versions[0].Name)
	a.Equal("1.14.2", dev.Artifacts[1].Versions[0].Value)
	require.Len(t, dev.Links, 1)
	a.Equal("https://argo/dev-app-a", dev.Links[0].URL)

	// app-a qa: desired moved ahead of what runs
	qa := appA.Cells["qa"]
	require.NotNil(t, qa)
	a.Equal("1.16.1", qa.Version)
	a.Equal("1.17.0", qa.DesiredVersion)
	a.True(qa.Drifted)

	// app-a stage: resolved through the second cluster
	a.Equal(app2Cluster, appA.Cells["stage"].Cluster)

	// worker: matched by selector, environment from namespace rule, no Application
	batch := g.Groups[1]
	require.Len(t, batch.Rows, 1)
	worker := batch.Rows[0]
	a.Equal("worker", worker.Component.Name)
	require.NotNil(t, worker.Cells["qa"])
	a.Equal("3.1.0", worker.Cells["qa"].Version)
	a.Empty(worker.Cells["qa"].DesiredVersion)
	a.Equal(repository.HealthHealthy, worker.Cells["qa"].Health)
	a.Equal([]string{"worker-alt.qa.example.com", "worker.qa.example.com"}, worker.Cells["qa"].Hosts, "labelled standalone ingress adds hosts")

	// statuses only for declared components of this system
	require.Len(t, res.Statuses["app-a"], 3)
	a.Equal("qa", res.Statuses["app-a"][1].Environment)
	a.Equal("1.17.0", res.Statuses["app-a"][1].DesiredVersion)
	a.Len(res.Statuses["worker"], 1)
	a.NotContains(res.Statuses, "app-b")
	a.NotContains(res.Statuses, "other-system-comp")
	require.Len(t, res.Discovered, 2, "app-b and metrics have no Component resource")
	a.Equal("app-b", res.Discovered[0].Name)
	a.Equal("Apps", res.Discovered[0].Group)
	a.Equal(v1alpha1.ComponentKindHelmChart, res.Discovered[0].Kind)
	a.Equal("metrics", res.Discovered[1].Name)
	a.Equal([]string{app1Cluster}, dev.Clusters)

	// labelled resources in system namespaces are still rows
	require.Len(t, g.Groups, 3, "Apps, Batch and Default (metrics)")
	a.Equal("metrics", g.Groups[2].Rows[0].Component.Name)

	// unassigned: the orphan deployment, plus the lost application which has
	// a component but no resolvable environment; coredns in kube-system is
	// ignored and ingresses never appear
	require.Len(t, res.Unassigned, 2)
	a.Equal("orphan", res.Unassigned[0].Name)
	a.Equal("dev", res.Unassigned[0].Environment, "environment still resolved through the namespace rule")
	a.Equal("lost", res.Unassigned[1].Name)

	// warnings
	a.Contains(g.Warnings, "application ops/lost targets unknown cluster \"https://nowhere\"")
	a.Contains(g.Warnings, "environment \"Prod\" on argocd-application ops/lost is not declared in system \"apps\"")
	found := false
	for _, w := range g.Warnings {
		if len(w) > 0 && w[0] >= '0' && w[0] <= '9' {
			found = true
			a.Contains(w, "legacy deploygrid/* annotations")
		}
	}
	a.True(found, "legacy annotation warning expected: %v", g.Warnings)
}

func TestBuild_SystemLabelFiltersAndParentNesting(t *testing.T) {
	in := fixture(t)
	in.Components = append(in.Components, &v1alpha1.Component{
		ObjectMeta: metav1.ObjectMeta{Name: "app-a-db"},
		Spec:       v1alpha1.ComponentSpec{System: "apps", Group: "Apps", Parent: "app-a", Kind: v1alpha1.ComponentKindContainer},
	})
	other := deployment("dev-other", "svc", "", "", "9.9.9", map[string]string{grid.LabelSystem: "other", grid.LabelComponent: "svc"})
	mine := deployment("dev-db", "db", "", "", "15.2", map[string]string{grid.LabelSystem: "apps", grid.LabelComponent: "app-a-db"})
	in.Observed[app1Cluster] = snapshot(t, other, mine)

	res := grid.Build(in)
	apps := res.Grid.Groups[0]
	require.Len(t, apps.Rows, 2, "app-a and discovered app-b; app-a-db is nested; svc belongs to another system")
	appA := apps.Rows[0]
	require.Len(t, appA.Children, 1)
	a := assert.New(t)
	a.Equal("app-a-db", appA.Children[0].Component.Name)
	a.Equal("15.2", appA.Children[0].Cells["dev"].Version)
	for _, u := range res.Unassigned {
		a.NotEqual("svc", u.Name)
	}
}

func TestBuild_Inconsistent(t *testing.T) {
	in := fixture(t)
	in.Observed[app1Cluster] = snapshot(t,
		deployment("dev-w", "w1", "", "", "1.0.0", map[string]string{grid.LabelComponent: "w", grid.LabelEnvironment: "dev"}),
		deployment("dev-w", "w2", "", "", "1.1.0", map[string]string{grid.LabelComponent: "w", grid.LabelEnvironment: "dev"}),
	)
	in.Observed[opsCluster] = snapshot(t)
	in.Observed[app2Cluster] = snapshot(t)

	res := grid.Build(in)
	var cell *deploygrid.Cell
	for _, g := range res.Grid.Groups {
		for _, r := range g.Rows {
			if r.Component.Name == "w" {
				cell = r.Cells["dev"]
			}
		}
	}
	require.NotNil(t, cell)
	assert.True(t, cell.Inconsistent)
	assert.Equal(t, []string{app1Cluster}, cell.Clusters, "both workloads run on one cluster")
	assert.Empty(t, res.Grid.Warnings, "fully labelled resources raise no warnings: %v", res.Grid.Warnings)
}

func operatorApp(ns, name, desired, running string, health string) *repository.Resource {
	return &repository.Resource{
		Name:             repository.CustomResourceName("platform.example.com", "Suite", ns, name),
		Kind:             repository.KindApplicationPrefix + "platform.example.com/suites",
		Namespace:        ns,
		ObjectName:       name,
		DesiredVersion:   desired,
		SyncRevision:     running,
		Health:           health,
		DefaultComponent: "suite",
	}
}

func ownedDeployment(ns, owner, name, image, stamped string) *repository.Resource {
	return &repository.Resource{
		Name:           "namespaces/" + ns + "/deployments/" + name,
		Kind:           repository.KindDeployment,
		Namespace:      ns,
		ObjectName:     name,
		Parent:         repository.CustomResourceName("platform.example.com", "Suite", ns, owner),
		Labels:         map[string]string{"app.kubernetes.io/managed-by": "suite-operator", "app.kubernetes.io/instance": owner, "app.kubernetes.io/name": name},
		Components:     []repository.Component{{Name: name, Kind: repository.VersionKindContainer, Version: image, Image: "registry/" + name + ":" + image}},
		StampedVersion: stamped,
		Health:         repository.HealthHealthy,
	}
}

func TestBuild_OperatorApplication(t *testing.T) {
	in := fixture(t)
	in.Clusters = append(in.Clusters, grid.ClusterInfo{Name: "ops-dev", Environment: "dev"})
	in.Observed["ops-dev"] = snapshot(t,
		// the operator reports what runs; workloads agree
		operatorApp("suite", "dev", "0.2.0", "0.2.0", repository.HealthDegraded),
		ownedDeployment("suite", "dev", "management", "0.2.0", "0.2.0"),
		ownedDeployment("suite", "dev", "console", "0.2.0", "0.2.0"),
	)
	in.Observed[app2Cluster] = snapshot(t,
		// no running version reported; workloads disagree mid-rollout
		operatorApp("suite", "stage", "0.3.0", "", repository.HealthProgressing),
		ownedDeployment("suite", "stage", "management", "0.3.0", "0.3.0"),
		ownedDeployment("suite", "stage", "console", "0.2.0", "0.3.0"),
	)

	res := grid.Build(in)
	var row *deploygrid.GridRow
	for _, g := range res.Grid.Groups {
		for _, r := range g.Rows {
			if r.Component.Name == "suite" {
				row = r
			}
		}
	}
	require.NotNil(t, row, "operator application discovered as component 'suite'")
	a := assert.New(t)
	a.True(row.Component.Discovered)
	a.Equal(string(v1alpha1.ComponentKindOperatorApplication), row.Component.Kind)

	dev := row.Cells["dev"]
	require.NotNil(t, dev, "environment from the cluster default: %v", row.Cells)
	a.Equal("0.2.0", dev.Version)
	a.Equal("0.2.0", dev.DesiredVersion)
	a.False(dev.Drifted)
	a.False(dev.Inconsistent)
	a.Equal(repository.HealthDegraded, dev.Health, "the operator's own condition dominates")
	a.Len(dev.Artifacts, 3, "custom resource plus two owned deployments")

	stage := row.Cells["stage"]
	require.NotNil(t, stage)
	a.True(stage.Inconsistent, "workloads run different versions")
	a.Equal("0.3.0", stage.Version, "the stamped version when workloads disagree")
	a.Equal("0.3.0", stage.DesiredVersion)

	for _, u := range res.Unassigned {
		a.NotEqual("management", u.Name, "owned workloads are claimed, not unassigned")
	}
	require.Len(t, res.Discovered, 3)
	a.Equal(v1alpha1.ComponentKindOperatorApplication, res.Discovered[2].Kind)
}
