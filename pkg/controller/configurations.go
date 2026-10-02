package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"

	apiinframux "github.com/activatedio/deploygrid/pkg/apiinfra/mux"
	"github.com/activatedio/deploygrid/pkg/apiinfra/util"
	"github.com/activatedio/deploygrid/pkg/deploygrid"
	"github.com/activatedio/deploygrid/pkg/history"
	"github.com/activatedio/deploygrid/pkg/service"
)

const (
	PathParamConfiguration = "configuration"
	PathParamView          = "view"
	QueryEnvironment       = "environment"
	QuerySince             = "since"
	QueryLimit             = "limit"
)

// Configurations serves configuration documents, rendered views and version
// history for a system.
type Configurations interface {
	WithOpenapiBuilder
	Names(w http.ResponseWriter, r *http.Request)
	Values(w http.ResponseWriter, r *http.Request)
	Views(w http.ResponseWriter, r *http.Request)
	Render(w http.ResponseWriter, r *http.Request)
	SystemHistory(w http.ResponseWriter, r *http.Request)
	ComponentHistory(w http.ResponseWriter, r *http.Request)
}

type configurations struct {
	configs service.ConfigurationService
	history service.HistoryService
}

type configurationPathParams struct {
	System        string `path:"system"`
	Configuration string `path:"configuration"`
	Environment   string `query:"environment"`
}

type viewPathParams struct {
	System      string `path:"system"`
	View        string `path:"view"`
	Environment string `query:"environment"`
}

type historyParams struct {
	System      string `path:"system"`
	Environment string `query:"environment"`
	Since       string `query:"since" description:"RFC 3339 timestamp"`
	Limit       int    `query:"limit"`
}

type componentHistoryParams struct {
	historyParams
	Component string `path:"component"`
}

type HistoryList struct {
	Items []history.Change `json:"items"`
}

func (c *configurations) OpenapiBuilder() apiinframux.OpenapiBuilder {
	return func(r *openapi3.Reflector) error {
		ops := []struct {
			path, desc string
			req, resp  any
		}{
			{"/systems/{system}/configurations", "Configuration names of the system and the environments that override them.", systemPathParams{}, &deploygrid.ConfigurationList{}},
			{"/systems/{system}/configurations/{configuration}", "Merged values of one configuration; environment documents deep-merge over the system-wide one.", configurationPathParams{}, &deploygrid.ConfigurationValues{}},
			{"/systems/{system}/views", "Views of the system with template validity.", systemPathParams{}, &deploygrid.ViewList{}},
			{"/systems/{system}/history", "Version changes across the system, newest first.", historyParams{}, &HistoryList{}},
			{"/systems/{system}/components/{component}/history", "Version changes of one component, newest first.", componentHistoryParams{}, &HistoryList{}},
		}
		for _, op := range ops {
			if err := addOperation(r, http.MethodGet, op.path, op.desc, op.req, op.resp); err != nil {
				return err
			}
		}
		oc, err := r.NewOperationContext(http.MethodGet, "/systems/{system}/views/{view}")
		if err != nil {
			return err
		}
		oc.SetDescription("Render a view for an environment; the response carries the view's content type.")
		oc.AddReqStructure(viewPathParams{})
		oc.AddRespStructure("", openapi.WithContentType("text/plain"), openapi.WithHTTPStatus(http.StatusOK))
		oc.AddRespStructure(&apiinframux.Error{}, apiinframux.ContentOptionsJSONDefault...)
		return r.AddOperation(oc)
	}
}

func (c *configurations) Names(w http.ResponseWriter, r *http.Request) {
	items, err := c.configs.Names(r.Context(), mux.Vars(r)[PathParamSystem])
	if err != nil {
		apiinframux.HandleError(w, r, err)
		return
	}
	writeJSON(w, &deploygrid.ConfigurationList{Items: items})
}

func (c *configurations) Values(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	env := r.URL.Query().Get(QueryEnvironment)
	vals, err := c.configs.Values(r.Context(), vars[PathParamSystem], vars[PathParamConfiguration], env)
	if err != nil {
		apiinframux.HandleError(w, r, err)
		return
	}
	writeJSON(w, &deploygrid.ConfigurationValues{Name: vars[PathParamConfiguration], Environment: env, Values: vals})
}

func (c *configurations) Views(w http.ResponseWriter, r *http.Request) {
	items, err := c.configs.Views(r.Context(), mux.Vars(r)[PathParamSystem])
	if err != nil {
		apiinframux.HandleError(w, r, err)
		return
	}
	writeJSON(w, &deploygrid.ViewList{Items: items})
}

func (c *configurations) Render(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	content, contentType, err := c.configs.Render(r.Context(), vars[PathParamSystem], vars[PathParamView], r.URL.Query().Get(QueryEnvironment))
	if err != nil {
		apiinframux.HandleError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", contentType)
	util.CheckWrite(w.Write(content))
}

func historyQuery(r *http.Request) (history.Query, error) {
	vars := mux.Vars(r)
	q := history.Query{
		System:      vars[PathParamSystem],
		Component:   vars[PathParamComponent],
		Environment: r.URL.Query().Get(QueryEnvironment),
		Limit:       200,
	}
	if since := r.URL.Query().Get(QuerySince); since != "" {
		t, err := time.Parse(time.RFC3339, since)
		if err != nil {
			return q, apiinframux.ErrBadRequest
		}
		q.Since = t
	}
	if limit := r.URL.Query().Get(QueryLimit); limit != "" {
		n, err := strconv.Atoi(limit)
		if err != nil || n < 1 {
			return q, apiinframux.ErrBadRequest
		}
		q.Limit = n
	}
	return q, nil
}

func (c *configurations) listHistory(w http.ResponseWriter, r *http.Request) {
	q, err := historyQuery(r)
	if err != nil {
		apiinframux.HandleError(w, r, err)
		return
	}
	items, err := c.history.List(r.Context(), q)
	if err != nil {
		apiinframux.HandleError(w, r, err)
		return
	}
	writeJSON(w, &HistoryList{Items: items})
}

func (c *configurations) SystemHistory(w http.ResponseWriter, r *http.Request) { c.listHistory(w, r) }
func (c *configurations) ComponentHistory(w http.ResponseWriter, r *http.Request) {
	c.listHistory(w, r)
}

func NewConfigurations(configs service.ConfigurationService, hist service.HistoryService) Configurations {
	return &configurations{configs: configs, history: hist}
}
