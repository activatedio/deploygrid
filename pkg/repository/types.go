package repository

import "context"

type ComponentCriteria struct {
	Namespace  string
	ApiVersion string
	Kind       string
}

type Component struct {
	Namespace    string
	ApiVersion   string
	Kind         string
	SubComponent string
	Name         string
	Labels       map[string]string
	Version      string
}

type ComponentRepository interface {
	List(ctx context.Context, criteria ComponentCriteria) ([]Component, error)
}

type ClusterAwareAccessor[R any] interface {
	Get(ctx context.Context, clusterName string) (R, error)
}
