package mux

import (
	"github.com/gorilla/mux"
	"github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"
	"net/http"
)

var (
	contentTypeApplicationJSON = "application/json"
	ContentOptionsJsonSuccess  = []openapi.ContentOption{openapi.WithContentType(contentTypeApplicationJSON), openapi.WithHTTPStatus(http.StatusOK)}
	ContentOptionsJsonDefault  = []openapi.ContentOption{openapi.WithContentType(contentTypeApplicationJSON), func(cu *openapi.ContentUnit) {
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
