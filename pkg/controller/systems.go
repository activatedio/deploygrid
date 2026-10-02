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

const (
	PathParamSystem    = "system"
	PathParamComponent = "component"
)

type systemPathParams struct {
	System string `path:"system"`
}

type componentPathParams struct {
	System    string `path:"system"`
	Component string `path:"component"`
}

type systems struct {
	systemService service.SystemService
	gridService   service.GridService
}

func addOperation(r *openapi3.Reflector, method, path, description string, req, resp any) error {
	oc, err := r.NewOperationContext(method, path)
	if err != nil {
		return err
	}
	oc.SetDescription(description)
	if req != nil {
		oc.AddReqStructure(req)
	}
	oc.AddRespStructure(resp, apiinframux.ContentOptionsJSONSuccess...)
	oc.AddRespStructure(&apiinframux.Error{}, apiinframux.ContentOptionsJSONDefault...)
	return r.AddOperation(oc)
}

func (s *systems) OpenapiBuilder() apiinframux.OpenapiBuilder {
	return func(r *openapi3.Reflector) error {
		ops := []struct {
			path, desc string
			req, resp  any
		}{
			{"/systems", "List systems.", nil, &deploygrid.SystemList{}},
			{"/systems/{system}", "Get one system with its environments and groups.", systemPathParams{}, &deploygrid.System{}},
			{"/systems/{system}/grid", "The grid: ordered groups of component rows with one cell per environment.", systemPathParams{}, &deploygrid.Grid{}},
			{"/systems/{system}/components", "All component rows of the system, flattened.", systemPathParams{}, &deploygrid.GridRowList{}},
			{"/systems/{system}/components/{component}", "One component row with its cells.", componentPathParams{}, &deploygrid.GridRow{}},
			{"/systems/{system}/unassigned", "Observed resources that matched no component.", systemPathParams{}, &deploygrid.ArtifactList{}},
		}
		for _, op := range ops {
			if err := addOperation(r, http.MethodGet, op.path, op.desc, op.req, op.resp); err != nil {
				return err
			}
		}
		return nil
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

func (s *systems) Grid(w http.ResponseWriter, r *http.Request) {
	g, err := s.gridService.Grid(r.Context(), mux.Vars(r)[PathParamSystem])
	if err != nil {
		apiinframux.HandleError(w, r, err)
		return
	}
	writeJSON(w, g)
}

func (s *systems) Components(w http.ResponseWriter, r *http.Request) {
	rows, err := s.gridService.Rows(r.Context(), mux.Vars(r)[PathParamSystem])
	if err != nil {
		apiinframux.HandleError(w, r, err)
		return
	}
	if rows == nil {
		rows = []*deploygrid.GridRow{}
	}
	writeJSON(w, &deploygrid.GridRowList{Items: rows})
}

func (s *systems) Component(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	row, err := s.gridService.Row(r.Context(), vars[PathParamSystem], vars[PathParamComponent])
	if err != nil {
		apiinframux.HandleError(w, r, err)
		return
	}
	writeJSON(w, row)
}

func (s *systems) Unassigned(w http.ResponseWriter, r *http.Request) {
	items, err := s.gridService.Unassigned(r.Context(), mux.Vars(r)[PathParamSystem])
	if err != nil {
		apiinframux.HandleError(w, r, err)
		return
	}
	if items == nil {
		items = []*deploygrid.Artifact{}
	}
	writeJSON(w, &deploygrid.ArtifactList{Items: items})
}

func NewSystems(systemService service.SystemService, gridService service.GridService) Systems {
	return &systems{
		systemService: systemService,
		gridService:   gridService,
	}
}
