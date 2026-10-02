package e2e

import (
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/activatedio/deploygrid/pkg/apis/deploygrid.activated.io/v1alpha1"
	"github.com/activatedio/deploygrid/pkg/deploygrid"
)

func TestE2E(t *testing.T) {

	doTest(t, "./testdata/config-default.yaml", func(t *testing.T, baseURL string) {

		a := assert.New(t)

		r := resty.New().SetBaseURL(baseURL)

		// kind-app-cluster-2 is not watched by the server: run a collector
		// against it, authenticating with the token the server generated for
		// the agent-mode Cluster resource.
		token := waitForToken(t, "../.kind/kubeconfig-ops-cluster-1.yaml", "deploygrid", "kind-app-cluster-2-collector-token")
		startCollector(t, baseURL+"/api", "kind-app-cluster-2", token, "../.kind/kubeconfig-app-cluster-2.yaml")

		a.EventuallyWithT(func(c *assert.CollectT) {

			// v1 compatibility endpoint: grid of the first system
			g := &deploygrid.Grid{}
			e := &ErrorResponse{}
			resp, err := json(r.R()).SetError(e).SetResult(g).Get("/api/grid")
			require.NoError(c, err)
			require.True(c, resp.IsSuccess(), resp.String())
			require.Equal(c, "apps", g.System.Name)
			require.Len(c, g.Environments, 3)

			// systems are read from custom resources on the control cluster
			sl := &deploygrid.SystemList{}
			resp, err = json(r.R()).SetError(e).SetResult(sl).Get("/api/systems")
			require.NoError(c, err)
			require.True(c, resp.IsSuccess(), resp.String())
			require.Len(c, sl.Items, 1)
			require.Equal(c, "apps", sl.Items[0].Name)

			// the grid: one declared group with the two declared components,
			// cells resolved through the Argo CD applications on the ops
			// cluster down to the Helm-labelled deployments on the app clusters
			sg := &deploygrid.Grid{}
			resp, err = json(r.R()).SetError(e).SetResult(sg).Get("/api/systems/apps/grid")
			require.NoError(c, err)
			require.True(c, resp.IsSuccess(), resp.String())
			require.Equal(c, "dev", sg.Environments[0].Name)
			require.Len(c, sg.Groups, 1)
			require.Equal(c, "apps", sg.Groups[0].Name)
			require.Len(c, sg.Groups[0].Rows, 2)
			appA := sg.Groups[0].Rows[0]
			require.Equal(c, "app-a", appA.Component.Name)
			require.False(c, appA.Component.Discovered)
			dev := appA.Cells["dev"]
			require.NotNil(c, dev, "dev cell: %v", appA.Cells)
			require.Equal(c, "1.16.1", dev.Version)
			require.Equal(c, "1.16.1", dev.DesiredVersion)
			require.False(c, dev.Drifted)
			require.Equal(c, "kind-app-cluster-1", dev.Cluster)
			require.Len(c, dev.Artifacts, 3)
			require.Len(c, dev.Links, 1)
			// stage arrives only through the collector
			stage := appA.Cells["stage"]
			require.NotNil(c, stage, "stage cell: %v", appA.Cells)
			require.Equal(c, "kind-app-cluster-2", stage.Cluster)
			require.Equal(c, "1.16.1", stage.Version)

			rows := &deploygrid.GridRowList{}
			resp, err = json(r.R()).SetError(e).SetResult(rows).Get("/api/systems/apps/components")
			require.NoError(c, err)
			require.True(c, resp.IsSuccess(), resp.String())
			require.Len(c, rows.Items, 2)

			// app-b is not declared: discovery shows it and the reconciler
			// materialises a Component resource for it
			row := &deploygrid.GridRow{}
			resp, err = json(r.R()).SetError(e).SetResult(row).Get("/api/systems/apps/components/app-b")
			require.NoError(c, err)
			require.True(c, resp.IsSuccess(), resp.String())
			require.Equal(c, "app-b", row.Component.Name)
			require.True(c, row.Component.Discovered)
			comp := getComponent(t, "../.kind/kubeconfig-ops-cluster-1.yaml", "deploygrid", "app-b")
			require.NotNil(c, comp, "Component app-b not materialised yet")
			require.True(c, comp.Status.Discovered)
			require.Equal(c, "apps", comp.Spec.System)
			require.Equal(c, "apps", comp.Spec.Group, "group canonicalised against the System")
			require.Equal(c, v1alpha1.ComponentKindHelmChart, comp.Spec.Kind)

			ul := &deploygrid.ArtifactList{}
			resp, err = json(r.R()).SetError(e).SetResult(ul).Get("/api/systems/apps/unassigned")
			require.NoError(c, err)
			require.True(c, resp.IsSuccess(), resp.String())
			// kind ships local-path-provisioner outside kube-system; nothing
			// from the fixture namespaces may be unassigned
			for _, u := range ul.Items {
				require.Equal(c, "local-path-provisioner", u.Name, "unexpected unassigned %s/%s", u.Namespace, u.Name)
			}

			resp, err = json(r.R()).SetError(e).Get("/api/systems/missing")
			require.NoError(c, err)
			require.Equal(c, 404, resp.StatusCode())

			// configurations: the qa document deep-merges over the system-wide one
			cl := &deploygrid.ConfigurationList{}
			resp, err = json(r.R()).SetError(e).SetResult(cl).Get("/api/systems/apps/configurations")
			require.NoError(c, err)
			require.True(c, resp.IsSuccess(), resp.String())
			require.Len(c, cl.Items, 1)
			require.Equal(c, "endpoints", cl.Items[0].Name)
			require.Equal(c, []string{"qa"}, cl.Items[0].Environments)

			cv := &deploygrid.ConfigurationValues{}
			resp, err = json(r.R()).SetError(e).SetResult(cv).Get("/api/systems/apps/configurations/endpoints?environment=qa")
			require.NoError(c, err)
			require.True(c, resp.IsSuccess(), resp.String())
			require.Equal(c, map[string]any{"domain": "qa.example.com"}, cv.Values)

			// views render with the grid of the environment
			vl := &deploygrid.ViewList{}
			resp, err = json(r.R()).SetError(e).SetResult(vl).Get("/api/systems/apps/views")
			require.NoError(c, err)
			require.True(c, resp.IsSuccess(), resp.String())
			require.Len(c, vl.Items, 1)
			require.True(c, vl.Items[0].Valid)

			resp, err = r.R().Get("/api/systems/apps/views/hosts?environment=dev")
			require.NoError(c, err)
			require.True(c, resp.IsSuccess(), resp.String())
			require.Equal(c, "text/plain", resp.Header().Get("Content-Type"))
			require.Contains(c, resp.String(), "app-a.example.com -> 1.16.1")

			// history is empty until something changes, but the endpoint works
			hl := map[string]any{}
			resp, err = json(r.R()).SetError(e).SetResult(&hl).Get("/api/systems/apps/components/app-a/history")
			require.NoError(c, err)
			require.True(c, resp.IsSuccess(), resp.String())
			require.Contains(c, hl, "items")
			resp, err = json(r.R()).SetError(e).Get("/api/systems/apps/history?since=garbage")
			require.NoError(c, err)
			require.Equal(c, 400, resp.StatusCode())

			log.Info().Msg("test succeeded")

		}, 20*time.Second, time.Second)

		// the server records the collector heartbeat on the Cluster resource
		a.EventuallyWithT(func(c *assert.CollectT) {
			cl := getCluster(t, "../.kind/kubeconfig-ops-cluster-1.yaml", "deploygrid", "kind-app-cluster-2")
			require.NotNil(c, cl.Status.LastHeartbeatTime)
			require.Equal(c, "kind-app-cluster-2", cl.Name)
			connected := false
			for _, cond := range cl.Status.Conditions {
				if cond.Type == "Connected" && cond.Status == "True" {
					connected = true
				}
			}
			require.True(c, connected, "conditions: %v", cl.Status.Conditions)
		}, 40*time.Second, 2*time.Second)

	})

}
