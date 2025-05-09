package controller

import (
	apiinframux "github.com/activatedio/deploygrid/pkg/apiinfra/mux"
	"net/http"
)

type WithOpenapiBuilder interface {
	OpenapiBuilder() apiinframux.OpenapiBuilder
}

type Deployments interface {
	WithOpenapiBuilder
	Get(w http.ResponseWriter, r *http.Request)
}
