package controller

import "net/http"

type Deployments interface {
	Get(w http.ResponseWriter, r *http.Request)
}
