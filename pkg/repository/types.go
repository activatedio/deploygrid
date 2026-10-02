package repository

import (
	"context"
	"strings"
)

// Resource kinds observed from clusters.
const (
	KindApplication = "argocd-application"
	KindDeployment  = "deployment"
	KindIngress     = "ingress"
	// KindApplicationPrefix starts the kind of every operator application
	// resource: application/<group>/<resource>.
	KindApplicationPrefix = "application/"
)

// IsOperatorApplication reports whether a kind is an operator application
// custom resource.
func IsOperatorApplication(kind string) bool {
	return strings.HasPrefix(kind, KindApplicationPrefix)
}

// CustomResourceName is the store key of a custom resource, also used as
// the Parent of workloads that carry a controller ownerReference to it.
func CustomResourceName(group, kind, namespace, name string) string {
	return "customresources/" + group + "/" + kind + "/" + namespace + "/" + name
}

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
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Version string `json:"version"`
	// Image is the full image reference for containers.
	Image string `json:"image,omitempty"`
}

// ClusterLocation is where a delivery resource (Argo CD Application) places
// its workloads.
type ClusterLocation struct {
	Server    string `json:"server,omitempty"`
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

// Resource is an observed artifact: a workload or a delivery resource,
// normalised so the grid builder does not need to know about Kubernetes
// kinds.
type Resource struct {
	// Name is the store key, unique within a cluster:
	// applications/<name> or namespaces/<ns>/deployments/<name>.
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	Namespace  string `json:"namespace,omitempty"`
	ObjectName string `json:"objectName"`
	// Parent is the store key of the delivery resource that manages this one,
	// if known (for Helm-managed workloads: applications/<release>).
	Parent      string            `json:"parent,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	// Components are the running (actual) versions.
	Components []Component `json:"components,omitempty"`
	// DesiredVersion is what the delivery source asks for (Argo CD
	// targetRevision).
	DesiredVersion string `json:"desiredVersion,omitempty"`
	// SyncRevision is the revision the delivery source last applied.
	SyncRevision string `json:"syncRevision,omitempty"`
	// ChartVersion is the Helm chart version stamped on a workload.
	ChartVersion string `json:"chartVersion,omitempty"`
	// StampedVersion is the app.kubernetes.io/version label of a workload.
	StampedVersion string           `json:"stampedVersion,omitempty"`
	Health         string           `json:"health,omitempty"`
	Destination    *ClusterLocation `json:"destination,omitempty"`
	Hosts          []string         `json:"hosts,omitempty"`
	// DefaultComponent and DefaultEnvironment are identities the collector
	// derived from the resource itself (for operator applications); labels
	// and declared selectors take precedence over them.
	DefaultComponent   string `json:"defaultComponent,omitempty"`
	DefaultEnvironment string `json:"defaultEnvironment,omitempty"`
	// PinnedVersions are versions an operator application intentionally
	// pins for some of its workloads; they do not make a cell inconsistent.
	PinnedVersions []string `json:"pinnedVersions,omitempty"`
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

// Resources are the repositories collected from one cluster, keyed by the
// resource kind they produce.
type Resources struct {
	Applications ResourceRepository
	Deployment   ResourceRepository
	Ingress      ResourceRepository
	// Custom holds operator application repositories keyed by the kind
	// they emit (application/<group>/<resource>).
	Custom map[string]ResourceRepository
}

// ByKind returns the repositories keyed by the kind each one emits.
func (r *Resources) ByKind() map[string]ResourceRepository {
	out := map[string]ResourceRepository{}
	for k, repo := range r.Custom {
		out[k] = repo
	}
	if r.Applications != nil {
		out[KindApplication] = r.Applications
	}
	if r.Deployment != nil {
		out[KindDeployment] = r.Deployment
	}
	if r.Ingress != nil {
		out[KindIngress] = r.Ingress
	}
	return out
}
