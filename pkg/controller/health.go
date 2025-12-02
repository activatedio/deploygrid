package controller

import (
	"github.com/activatedio/deploygrid/pkg/apiinfra/util"
	"net/http"
)

type health struct{}

func (h *health) Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	util.CheckWrite(w.Write([]byte("SERVING")))
}

func NewHealth() Health {
	return &health{}
}
