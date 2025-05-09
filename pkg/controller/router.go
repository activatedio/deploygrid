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
	Deployments   Deployments
}

func NewRouter(params RouterParams) *mux.Router {

	var (
		SwaggerUIPathPrefix = "/swagger-ui"
	)

	r := mux.NewRouter()

	r.HandleFunc("/grid", params.Deployments.Get).Methods(http.MethodGet)

	_su := params.SwaggerConfig.SwaggerUIURL

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
