package mux

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"
)

var (
	contentTypeApplicationJSON = "application/json"
	ContentOptionsJSONSuccess  = []openapi.ContentOption{openapi.WithContentType(contentTypeApplicationJSON), openapi.WithHTTPStatus(http.StatusOK)}
	ContentOptionsJSONDefault  = []openapi.ContentOption{openapi.WithContentType(contentTypeApplicationJSON), func(cu *openapi.ContentUnit) {
		cu.IsDefault = true
		cu.Description = "Error"
	}}
)

type OpenapiBuilder func(reflector *openapi3.Reflector) error

type Openapi interface {
	Mount(router *mux.Router, builders ...OpenapiBuilder) error
}

type Error struct {
	Error string `json:"error"`
}
