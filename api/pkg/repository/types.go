package repository

import (
	"context"
)

type Component struct {
	// Name Primary name which uniquely identifies the component and can be used for parent relationshipos
	Name string
	// SimpleName which is common for other items in the same ro
	SimpleName string
	// DisplayName is the full name to display when showing the component in a cell
	DisplayName string
	// Type of component
	Type string
	// Version fo the commonent
	Version string
	// PathElement is the location in the tree for the component
	PathElement      string
	ChildrenLocation []ClusterLocation
}

type ClusterLocation struct {
	Server string
}

type Resource struct {
	Name       string
	Parent     string
	Labels     map[string]string
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

type Resources struct {
	Applications ResourceRepository
	Deployment   ResourceRepository
}
