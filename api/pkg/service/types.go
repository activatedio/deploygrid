package service

import (
	"context"
	"github.com/activatedio/deploygrid/pkg/deploygrid"
)

type GridService interface {
	Init()
	Get(ctx context.Context) (*deploygrid.Grid, error)
}
