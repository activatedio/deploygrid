package mux

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/activatedio/deploygrid/pkg/apiinfra/util"
)

// ErrNotFound is returned by services when a named resource does not exist;
// HandleError maps it to 404.
var ErrNotFound = errors.New("not found")

// ErrBadRequest marks client errors (invalid parameters, template execution
// failures); HandleError maps it to 400.
var ErrBadRequest = errors.New("bad request")

// WriteError writes a JSON error with the given status.
func WriteError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	util.Check(json.NewEncoder(w).Encode(&Error{Error: msg}))
}

func HandleError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, ErrBadRequest):
		status = http.StatusBadRequest
	default:
		log.Error().Err(err).Str("path", r.URL.Path).Msg("request failed")
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	util.Check(json.NewEncoder(w).Encode(&Error{Error: err.Error()}))
}
