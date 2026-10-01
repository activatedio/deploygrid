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

			g := &deploygrid.Grid{}
			e := &ErrorResponse{}
			resp, err := json(r.R()).SetError(e).SetResult(g).Get("/api/grid")
			require.NoError(c, err)
			require.True(c, resp.IsSuccess())
			require.Len(c, g.Components, 1)
			require.Len(c, g.Environments, 3)

			// v2: systems are read from custom resources on the control cluster
			sl := &deploygrid.SystemList{}
			resp, err = json(r.R()).SetError(e).SetResult(sl).Get("/api/systems")
			require.NoError(c, err)
			require.True(c, resp.IsSuccess(), resp.String())
			require.Len(c, sl.Items, 1)
			require.Equal(c, "apps", sl.Items[0].Name)
			require.Len(c, sl.Items[0].Environments, 3)

			sg := &deploygrid.Grid{}
			resp, err = json(r.R()).SetError(e).SetResult(sg).Get("/api/systems/apps/grid")
			require.NoError(c, err)
			require.True(c, resp.IsSuccess(), resp.String())
			require.Equal(c, "dev", sg.Environments[0].Name)

			resp, err = json(r.R()).SetError(e).Get("/api/systems/missing")
			require.NoError(c, err)
			require.Equal(c, 404, resp.StatusCode())

			log.Info().Msg("test succeeded")

		}, 5*time.Second, time.Second)

	})

}
