package controller

import (
	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/gorilla/mux"
	"go.uber.org/fx"
	"net/http"
	"net/url"
)

type RouterParams struct {
	fx.In
	SwaggerConfig *config.SwaggerConfig
	Grid          Grid
	Health        Health
}

func NewRouter(params RouterParams) *mux.Router {

	var (
		SwaggerUIPathPrefix = "/api/swagger-ui"
	)

	r := mux.NewRouter()

	r.HandleFunc("/api/healthz", params.Health.Healthz).Methods(http.MethodGet)
	r.HandleFunc("/api/grid", params.Grid.Get).Methods(http.MethodGet)

	_su := params.SwaggerConfig.SwaggerUiUrl

	if _su != "" {
		su, err := url.Parse(_su)
		if err != nil {
			panic(err)
		}

		sh := NewSwaggerUIHandler(su, SwaggerUIPathPrefix)

		r.PathPrefix(SwaggerUIPathPrefix).Handler(sh)
	}

	return r
}
