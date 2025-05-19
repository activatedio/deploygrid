package controller

import (
	apiinframux "github.com/activatedio/deploygrid/pkg/apiinfra/mux"
	"github.com/activatedio/deploygrid/pkg/deploygrid"
	"github.com/activatedio/deploygrid/pkg/service"
	"github.com/swaggest/openapi-go/openapi3"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http"
)

type grid struct {
	GridService service.GridService
}

func (d *grid) OpenapiBuilder() apiinframux.OpenapiBuilder {
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

func (d *grid) Get(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	g, err := d.GridService.Get(r.Context())

	if err != nil {
		apiinframux.HandleError(w, r, err)
		return
	}

	json.NewEncoder(w).Encode(g)
}

func NewGrid(gridService service.GridService) Grid {
	return &grid{
		GridService: gridService,
	}
}
