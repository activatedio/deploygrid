package controller

import (
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	"go.uber.org/fx"

	"github.com/activatedio/deploygrid/pkg/config"
)

type RouterParams struct {
	fx.In
	SwaggerConfig *config.SwaggerConfig
	Grid          Grid
	Systems       Systems
	Observations  Observations
	Health        Health
}

const (
	PathBase             = "/api"
	PathHealth           = PathBase + "/healthz"
	PathGrid             = PathBase + "/grid"
	PathSystems          = PathBase + "/systems"
	PathSystem           = PathSystems + "/{" + PathParamSystem + "}"
	PathSystemGrid       = PathSystem + "/grid"
	PathSystemComponents = PathSystem + "/components"
	PathSystemComponent  = PathSystemComponents + "/{" + PathParamComponent + "}"
	PathSystemUnassigned = PathSystem + "/unassigned"
	PathObservations     = PathBase + "/observations"
)

func NewRouter(params RouterParams) *mux.Router {

	var (
		SwaggerUIPathPrefix = "/api/swagger-ui"
	)

	r := mux.NewRouter()

	r.HandleFunc(PathHealth, params.Health.Healthz).Methods(http.MethodGet)
	// v1 compatibility: the grid of the first/default system
	r.HandleFunc(PathGrid, params.Grid.Get).Methods(http.MethodGet)
	r.HandleFunc(PathSystems, params.Systems.List).Methods(http.MethodGet)
	r.HandleFunc(PathSystem, params.Systems.Get).Methods(http.MethodGet)
	r.HandleFunc(PathSystemGrid, params.Systems.Grid).Methods(http.MethodGet)
	r.HandleFunc(PathSystemComponents, params.Systems.Components).Methods(http.MethodGet)
	r.HandleFunc(PathSystemComponent, params.Systems.Component).Methods(http.MethodGet)
	r.HandleFunc(PathSystemUnassigned, params.Systems.Unassigned).Methods(http.MethodGet)
	r.HandleFunc(PathObservations, params.Observations.Post).Methods(http.MethodPost)

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
