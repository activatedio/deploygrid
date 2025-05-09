package service

import (
	"context"
	"github.com/activatedio/deploygrid/pkg/deploygrid"
)

type GridService interface {
	Get(ctx context.Context) (*deploygrid.Grid, error)
}
