package e2e

import (
	"github.com/activatedio/deploygrid/pkg/deploygrid"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestE2E(t *testing.T) {

	doTest(t, "./testdata/config-default.yaml", func(t *testing.T, baseURL string) {

		a := assert.New(t)

		r := resty.New().SetBaseURL(baseURL)

		a.EventuallyWithT(func(c *assert.CollectT) {

			g := &deploygrid.Grid{}
			e := &ErrorResponse{}
			resp, err := json(r.R()).SetError(e).SetResult(g).Get("/api/grid")
			assert.Nil(c, err)
			assert.True(c, resp.IsSuccess())
			assert.Len(c, g.Components, 2)
			assert.Len(c, g.Environments, 3)

		}, 5*time.Second, time.Second)

	})

}
