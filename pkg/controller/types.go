package controller

import (
	apiinframux "github.com/activatedio/deploygrid/pkg/apiinfra/mux"
	"net/http"
)

type WithOpenapiBuilder interface {
	OpenapiBuilder() apiinframux.OpenapiBuilder
}

type Grid interface {
	WithOpenapiBuilder
	Get(w http.ResponseWriter, r *http.Request)
}

type Health interface {
	Healthz(w http.ResponseWriter, r *http.Request)
}
