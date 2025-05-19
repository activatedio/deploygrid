package e2e

import (
	"context"
	"fmt"
	"github.com/activatedio/deploygrid/pkg/apiinfra/viper"
	deploygridfx "github.com/activatedio/deploygrid/pkg/fx"
	"github.com/activatedio/deploygrid/pkg/runner"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
	"testing"
	"time"
)

func json(r *resty.Request) *resty.Request {
	return r.SetHeader("Content-Type", "application/json")
}

func waitForHealth(url string) {

	for i := 0; i < 30; i++ {
		resp, err := resty.New().SetBaseURL(url).R().Get("/api/healthz")
		if err == nil && resp.IsSuccess() {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	panic("health check timed out")
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func checkResp(resp *resty.Response, err error) {
	if err != nil {
		panic(err)
	}
	if resp == nil {
		panic("response is nil")
	}
	if resp.IsError() {
		panic(resp.String())
	}
}

func doTest(t *testing.T, configPath string, callback func(t *testing.T, baseURL string)) {

	fxctx := context.Background()

	var baseUrl string

	v := viper.NewViper(viper.WithConfigPath(configPath))

	app := fx.New(deploygridfx.Index(v),
		fx.Invoke(func(server *runner.RunningServer) {
			baseUrl = fmt.Sprintf("http://%s:%d", server.Host, server.Port)
			log.Info().Str("host", server.Host).Int("port", server.Port).Msg("Starting server")
		}))
	check(app.Start(fxctx))

	waitForHealth(baseUrl)

	callback(t, baseUrl)

}

type ErrorResponse struct {
	Error string `json:"error"`
}
