package mux

import (
	"encoding/json"
	"net/http"

	"github.com/activatedio/deploygrid/pkg/apiinfra/util"
	"github.com/rs/zerolog/log"
)

func HandleError(w http.ResponseWriter, r *http.Request, err error) {
	log.Error().Err(err)
	w.Header().Set("Content-Type", "application/json;")
	w.WriteHeader(http.StatusInternalServerError)
	// TODO - let's make this better
	util.Check(json.NewEncoder(w).Encode(&Error{Error: err.Error()}))
}
