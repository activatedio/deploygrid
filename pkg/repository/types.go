package repository

import (
	"context"
)

// Resource kinds observed from clusters.
const (
	KindApplication = "argocd-application"
	KindDeployment  = "deployment"
)

// Component (version) kinds.
const (
	VersionKindContainer = "container"
	VersionKindChart     = "chart"
	VersionKindRevision  = "revision"
)

// Health values, coarse enough to roll up across kinds.
const (
	HealthHealthy     = "Healthy"
	HealthProgressing = "Progressing"
	HealthDegraded    = "Degraded"
	HealthUnknown     = "Unknown"
)

// Component is one versioned part of a Resource: a container image, a chart
// or a delivery revision.
type Component struct {
	// Name is unique within the resource (container name, "chart", "revision").
	Name    string
	Kind    string
	Version string
	// Image is the full image reference for containers.
	Image string
}

// ClusterLocation is where a delivery resource (Argo CD Application) places
// its workloads.
type ClusterLocation struct {
	Server    string
	Name      string
	Namespace string
}

// Resource is an observed artifact: a workload or a delivery resource,
// normalised so the grid builder does not need to know about Kubernetes
// kinds.
type Resource struct {
	// Name is the store key, unique within a cluster:
	// applications/<name> or namespaces/<ns>/deployments/<name>.
	Name       string
	Kind       string
	Namespace  string
	ObjectName string
	// Parent is the store key of the delivery resource that manages this one,
	// if known (for Helm-managed workloads: applications/<release>).
	Parent      string
	Labels      map[string]string
	Annotations map[string]string
	// Components are the running (actual) versions.
	Components []Component
	// DesiredVersion is what the delivery source asks for (Argo CD
	// targetRevision).
	DesiredVersion string
	// SyncRevision is the revision the delivery source last applied.
	SyncRevision string
	// ChartVersion is the Helm chart version stamped on a workload.
	ChartVersion string
	Health       string
	Destination  *ClusterLocation
	Hosts        []string
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
