package controller

import (
	"net/http"

	apiinframux "github.com/activatedio/deploygrid/pkg/apiinfra/mux"
)

type WithOpenapiBuilder interface {
	OpenapiBuilder() apiinframux.OpenapiBuilder
}

// Grid serves the v1 compatibility endpoint: the grid of the first System.
type Grid interface {
	WithOpenapiBuilder
	Get(w http.ResponseWriter, r *http.Request)
}

// Systems serves the v2 system-scoped endpoints.
type Systems interface {
	WithOpenapiBuilder
	List(w http.ResponseWriter, r *http.Request)
	Get(w http.ResponseWriter, r *http.Request)
	Grid(w http.ResponseWriter, r *http.Request)
	Components(w http.ResponseWriter, r *http.Request)
	Component(w http.ResponseWriter, r *http.Request)
	Unassigned(w http.ResponseWriter, r *http.Request)
}

type Health interface {
	Healthz(w http.ResponseWriter, r *http.Request)
}
