package controller

import (
	"net/http"
)

type health struct{}

func (h *health) Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("SERVING"))
}

func NewHealth() Health {
	return &health{}
}
