package service

import (
	"context"
	"github.com/activatedio/deploygrid/pkg/deploygrid"
	"go.uber.org/fx"
)

type gridService struct {
}

func (g gridService) Get(ctx context.Context) (*deploygrid.Grid, error) {
	//TODO implement me
	panic("implement me")
}

type GridServiceParams struct {
	fx.In
}

func NewGridService(params GridServiceParams) GridService {

	return &gridService{}
}
