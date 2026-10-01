package controller

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/swaggest/openapi-go/openapi3"

	apiinframux "github.com/activatedio/deploygrid/pkg/apiinfra/mux"
	"github.com/activatedio/deploygrid/pkg/apiinfra/util"
	"github.com/activatedio/deploygrid/pkg/deploygrid"
	"github.com/activatedio/deploygrid/pkg/service"
)

const PathParamSystem = "system"

type systemPathParams struct {
	System string `path:"system"`
}

type systems struct {
	systemService service.SystemService
	gridService   service.GridService
}

func (s *systems) OpenapiBuilder() apiinframux.OpenapiBuilder {
	return func(r *openapi3.Reflector) error {

		list, err := r.NewOperationContext(http.MethodGet, "/systems")
		if err != nil {
			return err
		}
		list.AddRespStructure(&deploygrid.SystemList{}, apiinframux.ContentOptionsJSONSuccess...)
		list.AddRespStructure(&apiinframux.Error{}, apiinframux.ContentOptionsJSONDefault...)
		if err = r.AddOperation(list); err != nil {
			return err
		}

		get, err := r.NewOperationContext(http.MethodGet, "/systems/{system}")
		if err != nil {
			return err
		}
		get.AddReqStructure(systemPathParams{})
		get.AddRespStructure(&deploygrid.System{}, apiinframux.ContentOptionsJSONSuccess...)
		get.AddRespStructure(&apiinframux.Error{}, apiinframux.ContentOptionsJSONDefault...)
		if err = r.AddOperation(get); err != nil {
			return err
		}

		grid, err := r.NewOperationContext(http.MethodGet, "/systems/{system}/grid")
		if err != nil {
			return err
		}
		grid.AddReqStructure(systemPathParams{})
		grid.AddRespStructure(&deploygrid.Grid{}, apiinframux.ContentOptionsJSONSuccess...)
		grid.AddRespStructure(&apiinframux.Error{}, apiinframux.ContentOptionsJSONDefault...)
		return r.AddOperation(grid)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	util.Check(json.NewEncoder(w).Encode(v))
}

func (s *systems) List(w http.ResponseWriter, r *http.Request) {
	items, err := s.systemService.List(r.Context())
	if err != nil {
		apiinframux.HandleError(w, r, err)
		return
	}
	writeJSON(w, &deploygrid.SystemList{Items: items})
}

func (s *systems) Get(w http.ResponseWriter, r *http.Request) {
	sys, err := s.systemService.Get(r.Context(), mux.Vars(r)[PathParamSystem])
	if err != nil {
		apiinframux.HandleError(w, r, err)
		return
	}
	writeJSON(w, sys)
}

// Grid returns the grid for a System. Until the v2 grid builder lands, the
// grid is the v1 cluster-wide grid with the System's environments as
// columns.
func (s *systems) Grid(w http.ResponseWriter, r *http.Request) {
	sys, err := s.systemService.Get(r.Context(), mux.Vars(r)[PathParamSystem])
	if err != nil {
		apiinframux.HandleError(w, r, err)
		return
	}
	g, err := s.gridService.Get(r.Context())
	if err != nil {
		apiinframux.HandleError(w, r, err)
		return
	}
	g.Environments = sys.Environments
	writeJSON(w, g)
}

func NewSystems(systemService service.SystemService, gridService service.GridService) Systems {
	return &systems{
		systemService: systemService,
		gridService:   gridService,
	}
}
