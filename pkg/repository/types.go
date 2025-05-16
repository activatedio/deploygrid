package repository

import (
	"context"
)

type Component struct {
	DisplayName string
	Version     string
}

type Resource struct {
	Name       string
	Parent     string
	Components []Component
}

type ResourceStore interface {
	Add(in *Resource) error
	Modify(in *Resource) error
	Delete(in *Resource) error
	Replace(in []*Resource) error
	Error(err error)
}

type ResourceRepository interface {
	Watch(ctx context.Context, store ResourceStore)
}

type ClusterAwareAccessor[R any] interface {
	ClusterNames(ctx context.Context) []string
	Get(ctx context.Context, clusterName string) (R, error)
}
