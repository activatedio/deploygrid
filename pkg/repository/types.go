package repository

import "context"

type ComponentCriteria struct {
	Namespace  string
	ApiVersion string
	Kind       string
}

type Component struct {
	Namespace        string
	ApiVersion       string
	Kind             string
	ComponentName    string
	SubComponentName string
	Name             string
	Labels           map[string]string
	Version          string
}

type ComponentRepository interface {
	List(ctx context.Context, criteria ComponentCriteria) ([]Component, error)
}

type ClusterAwareAccessor[R any] interface {
	ClusterNames(ctx context.Context) []string
	Get(ctx context.Context, clusterName string) (R, error)
}
