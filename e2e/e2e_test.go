package e2e

import (
	"testing"
	"time"

	"github.com/activatedio/deploygrid/pkg/deploygrid"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

			log.Info().Msg("test succeeded")

		}, 5*time.Second, time.Second)

	})

}
