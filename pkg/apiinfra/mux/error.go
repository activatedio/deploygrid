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

// WriteError writes a JSON error with the given status.
func WriteError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	util.Check(json.NewEncoder(w).Encode(&Error{Error: msg}))
}

func HandleError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	if errors.Is(err, ErrNotFound) {
		status = http.StatusNotFound
	} else {
		log.Error().Err(err).Str("path", r.URL.Path).Msg("request failed")
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	util.Check(json.NewEncoder(w).Encode(&Error{Error: err.Error()}))
}
