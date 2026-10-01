package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"

	apiinfraconfig "github.com/activatedio/deploygrid/pkg/apiinfra/config"
	"github.com/activatedio/deploygrid/pkg/config"
	deploygridfx "github.com/activatedio/deploygrid/pkg/fx"
	"github.com/activatedio/deploygrid/pkg/runner"
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

func doTest(t *testing.T, configPath string, callback func(t *testing.T, baseURL string)) {

	fxctx := context.Background()

	var baseURL string

	m := config.NewMainConfig(apiinfraconfig.NewConfig(configPath))

	app := fx.New(deploygridfx.Index(m),
		fx.Invoke(func(server *runner.RunningServer) {
			baseURL = fmt.Sprintf("http://%s:%d", server.Host, server.Port)
			log.Info().Str("host", server.Host).Int("port", server.Port).Msg("Starting server")
		}))
	check(app.Start(fxctx))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = app.Stop(ctx)
	})

	waitForHealth(baseURL)

	callback(t, baseURL)

}

type ErrorResponse struct {
	Error string `json:"error"`
}
