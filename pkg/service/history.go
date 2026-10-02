package service

import (
	"context"

	"github.com/activatedio/deploygrid/pkg/history"
)

type historyService struct {
	catalog Catalog
	ring    *history.Ring
}

func (h *historyService) List(_ context.Context, q history.Query) ([]history.Change, error) {
	if _, err := h.catalog.System(q.System); err != nil {
		return nil, err
	}
	out := h.ring.List(q)
	if out == nil {
		out = []history.Change{}
	}
	return out, nil
}

func NewHistoryRing() *history.Ring {
	return history.New(history.DefaultCapacity)
}

func NewHistoryService(catalog Catalog, ring *history.Ring) HistoryService {
	return &historyService{catalog: catalog, ring: ring}
}
