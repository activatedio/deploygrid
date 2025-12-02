package controller

import (
	"net/http"
	"net/url"

	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/gorilla/mux"
	"go.uber.org/fx"
)

type RouterParams struct {
	fx.In
	SwaggerConfig *config.SwaggerConfig
	Grid          Grid
	Metadata      Metadata
	Health        Health
}

const (
	PathBase     = "/api"
	PathHealth   = PathBase + "/healthz"
	PathGrid     = PathBase + "/grid"
	PathMetadata = PathBase + "/metadata"
)

func NewRouter(params RouterParams) *mux.Router {

	var (
		SwaggerUIPathPrefix = "/api/swagger-ui"
	)

	r := mux.NewRouter()

	r.HandleFunc(PathHealth, params.Health.Healthz).Methods(http.MethodGet)
	r.HandleFunc(PathGrid, params.Grid.Get).Methods(http.MethodGet)
	r.HandleFunc(PathMetadata, params.Metadata.Get).Methods(http.MethodGet)
	r.HandleFunc(PathMetadata, params.Metadata.Post).Methods(http.MethodPost)

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
