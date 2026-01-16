package controller

import (
	"net/http"

	"github.com/activatedio/deploygrid/pkg/apiinfra/util"
)

type health struct{}

func (h *health) Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	util.CheckWrite(w.Write([]byte("SERVING")))
}

func NewHealth() Health {
	return &health{}
}
