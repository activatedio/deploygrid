package controller

import (
	apiinframux "github.com/activatedio/deploygrid/pkg/apiinfra/mux"
	"github.com/activatedio/deploygrid/pkg/deploygrid"
	"github.com/swaggest/openapi-go/openapi3"
	"net/http"
)

type deployments struct{}

func (d *deployments) OpenapiBuilder() apiinframux.OpenapiBuilder {
	return func(r *openapi3.Reflector) error {

		oc, err := r.NewOperationContext(http.MethodGet, "/grid")

		if err != nil {
			return err
		}
		oc.AddRespStructure(&deploygrid.Grid{}, apiinframux.ContentOptionsJsonSuccess...)
		oc.AddRespStructure(&apiinframux.Error{}, apiinframux.ContentOptionsJsonDefault...)

		return r.AddOperation(oc)
	}
}

func (d *deployments) Get(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"a":"b"}`))
}

func NewDeployments() Deployments {
	return &deployments{}
}
