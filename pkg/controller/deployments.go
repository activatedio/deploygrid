package controller

import (
	"net/http"
)

type deployments struct{}

func (d *deployments) Get(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"a":"b"}`))
}

func NewDeployments() Deployments {
	return &deployments{}
}
