package controller

import (
	"net/http"

	apiinframux "github.com/activatedio/deploygrid/pkg/apiinfra/mux"
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

type Metadata interface {
	WithOpenapiBuilder
	Get(w http.ResponseWriter, r *http.Request)
	Post(w http.ResponseWriter, r *http.Request)
}
