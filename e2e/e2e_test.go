package e2e

import (
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/activatedio/deploygrid/pkg/deploygrid"
)

func TestE2E(t *testing.T) {

	doTest(t, "./testdata/config-default.yaml", func(t *testing.T, baseURL string) {

		a := assert.New(t)

		r := resty.New().SetBaseURL(baseURL)

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
			stage := appA.Cells["stage"]
			require.NotNil(c, stage, "stage cell: %v", appA.Cells)
			require.Equal(c, "kind-app-cluster-2", stage.Cluster)

			rows := &deploygrid.GridRowList{}
			resp, err = json(r.R()).SetError(e).SetResult(rows).Get("/api/systems/apps/components")
			require.NoError(c, err)
			require.True(c, resp.IsSuccess(), resp.String())
			require.Len(c, rows.Items, 2)

			row := &deploygrid.GridRow{}
			resp, err = json(r.R()).SetError(e).SetResult(row).Get("/api/systems/apps/components/app-b")
			require.NoError(c, err)
			require.True(c, resp.IsSuccess(), resp.String())
			require.Equal(c, "app-b", row.Component.Name)

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

			log.Info().Msg("test succeeded")

		}, 5*time.Second, time.Second)

	})

}
