package mux

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"net/http"
)

func HandleError(w http.ResponseWriter, r *http.Request, err error) {
	log.Error().Err(err)
	w.Header().Set("Content-Type", "application/json;")
	w.WriteHeader(http.StatusInternalServerError)
	// TODO - let's make this better
	json.NewEncoder(w).Encode(&Error{Error: err.Error()})
}
