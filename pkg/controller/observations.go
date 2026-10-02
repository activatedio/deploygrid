package controller

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/swaggest/openapi-go/openapi3"

	apiinframux "github.com/activatedio/deploygrid/pkg/apiinfra/mux"
	"github.com/activatedio/deploygrid/pkg/deploygrid"
	"github.com/activatedio/deploygrid/pkg/service"
)

// maxObservationBytes bounds a single push; snapshots of large clusters are
// well under this.
const maxObservationBytes = 64 << 20

type Observations interface {
	WithOpenapiBuilder
	Post(w http.ResponseWriter, r *http.Request)
}

type observations struct {
	service service.ObservationService
}

func (o *observations) OpenapiBuilder() apiinframux.OpenapiBuilder {
	return func(r *openapi3.Reflector) error {
		oc, err := r.NewOperationContext(http.MethodPost, "/observations")
		if err != nil {
			return err
		}
		oc.SetDescription("Collector push. Authenticate with a bearer token issued for one Cluster; a snapshot replaces the listed kinds, otherwise resources are upserted and removed incrementally.")
		oc.AddReqStructure(&deploygrid.Observation{})
		oc.AddRespStructure(&deploygrid.ObservationResponse{}, apiinframux.ContentOptionsJSONSuccess...)
		oc.AddRespStructure(&apiinframux.Error{}, apiinframux.ContentOptionsJSONDefault...)
		return r.AddOperation(oc)
	}
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}

func (o *observations) Post(w http.ResponseWriter, r *http.Request) {
	cluster, err := o.service.Authenticate(r.Context(), bearerToken(r))
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="deploygrid"`)
			apiinframux.WriteError(w, http.StatusUnauthorized, "invalid or missing bearer token")
			return
		}
		apiinframux.HandleError(w, r, err)
		return
	}

	obs := &deploygrid.Observation{}
	if err := json.NewDecoder(io.LimitReader(r.Body, maxObservationBytes)).Decode(obs); err != nil {
		apiinframux.WriteError(w, http.StatusBadRequest, "invalid observation: "+err.Error())
		return
	}
	if obs.Cluster != "" && obs.Cluster != cluster {
		apiinframux.WriteError(w, http.StatusForbidden, "token is not valid for cluster "+obs.Cluster)
		return
	}

	resp, err := o.service.Ingest(r.Context(), cluster, obs)
	if err != nil {
		apiinframux.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, resp)
}

func NewObservations(svc service.ObservationService) Observations {
	return &observations{service: svc}
}
