package controller

import (
	"fmt"
	"net/http"

	"github.com/swaggest/openapi-go/openapi3"

	apiinframux "github.com/activatedio/deploygrid/pkg/apiinfra/mux"
	"github.com/activatedio/deploygrid/pkg/deploygrid"
	"github.com/activatedio/deploygrid/pkg/service"
)

type grid struct {
	systemService service.SystemService
	gridService   service.GridService
}

func (d *grid) OpenapiBuilder() apiinframux.OpenapiBuilder {
	return func(r *openapi3.Reflector) error {

		oc, err := r.NewOperationContext(http.MethodGet, "/grid")

		if err != nil {
			return err
		}
		oc.SetDescription("Grid of the first System; kept for v1 compatibility.")
		oc.AddRespStructure(&deploygrid.Grid{}, apiinframux.ContentOptionsJSONSuccess...)
		oc.AddRespStructure(&apiinframux.Error{}, apiinframux.ContentOptionsJSONDefault...)

		return r.AddOperation(oc)
	}
}

func (d *grid) Get(w http.ResponseWriter, r *http.Request) {
	systems, err := d.systemService.List(r.Context())
	if err != nil {
		apiinframux.HandleError(w, r, err)
		return
	}
	if len(systems) == 0 {
		apiinframux.HandleError(w, r, fmt.Errorf("no systems defined: %w", apiinframux.ErrNotFound))
		return
	}
	g, err := d.gridService.Grid(r.Context(), systems[0].Name)
	if err != nil {
		apiinframux.HandleError(w, r, err)
		return
	}
	writeJSON(w, g)
}

func NewGrid(systemService service.SystemService, gridService service.GridService) Grid {
	return &grid{
		systemService: systemService,
		gridService:   gridService,
	}
}
