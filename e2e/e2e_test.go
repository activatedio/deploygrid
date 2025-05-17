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

		g := &deploygrid.Grid{}
		e := &ErrorResponse{}

		time.Sleep(5 * time.Second)
		checkResp(json(r.R()).SetError(e).SetResult(g).Get("/grid"))
		a.NotNil(g.Components)

	})

}
