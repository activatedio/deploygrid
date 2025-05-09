package controller

import (
	"net/http"
)

type deployments struct{}

func (d *deployments) Get(w http.ResponseWriter, r *http.Request) {
	//TODO implement me
	panic("implement me")
}

func NewDeployments() Deployments {
	return &deployments{}
}
