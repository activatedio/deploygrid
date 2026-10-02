package k8s

import (
	"strings"
	"unicode"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/activatedio/deploygrid/pkg/repository"
)

const (
	labelManagedBy = "app.kubernetes.io/managed-by"
	labelInstance  = "app.kubernetes.io/instance"
	labelHelmChart = "helm.sh/chart"
)

// argoHealth maps Argo CD health values onto the coarse health used here.
func argoHealth(status string) string {
	switch status {
	case "Healthy":
		return repository.HealthHealthy
	case "Progressing":
		return repository.HealthProgressing
	case "Degraded", "Missing":
		return repository.HealthDegraded
	default:
		return repository.HealthUnknown
	}
}

// deploymentConditions picks out the conditions health depends on; replica
// failure short-circuits to Degraded.
func deploymentConditions(dep *appsv1.Deployment) (available, progressing *appsv1.DeploymentCondition, failed bool) {
	for i := range dep.Status.Conditions {
		c := &dep.Status.Conditions[i]
		switch c.Type {
		case appsv1.DeploymentAvailable:
			available = c
		case appsv1.DeploymentProgressing:
			progressing = c
		case appsv1.DeploymentReplicaFailure:
			failed = failed || c.Status == corev1.ConditionTrue
		}
	}
	return available, progressing, failed
}

// deploymentHealth derives health from Deployment conditions.
func deploymentHealth(dep *appsv1.Deployment) string {
	available, progressing, failed := deploymentConditions(dep)
	switch {
	case failed:
		return repository.HealthDegraded
	case progressing != nil && progressing.Status == corev1.ConditionFalse:
		return repository.HealthDegraded
	case available == nil:
		return repository.HealthUnknown
	case available.Status != corev1.ConditionTrue:
		return repository.HealthDegraded
	case progressing != nil && progressing.Reason != "NewReplicaSetAvailable":
		return repository.HealthProgressing
	default:
		return repository.HealthHealthy
	}
}

// ChartVersionFromLabel extracts "1.2.3" from a helm.sh/chart label such as
// "sealed-secrets-1.2.3". It returns "" when no version suffix is present.
func ChartVersionFromLabel(label string) string {
	i := strings.LastIndex(label, "-")
	if i < 0 || i == len(label)-1 {
		return ""
	}
	v := label[i+1:]
	if !unicode.IsDigit(rune(v[0])) {
		return ""
	}
	return v
}

func NewApplicationRepository(client dynamic.Interface) repository.ResourceRepository {
	return NewResourceRepository(ResourceRepositoryParams{
		Client: client,
		GroupVersionResource: schema.GroupVersionResource{
			Group:    "argoproj.io",
			Version:  "v1alpha1",
			Resource: "applications",
		},
		ToResource: func(obj *unstructured.Unstructured) (*repository.Resource, error) {

			app := &Application{}

			err := DecodeMap(obj.Object, app)

			if err != nil {
				return nil, err
			}

			var comps []repository.Component
			if rev := app.Status.Sync.Revision; rev != "" {
				comps = append(comps, repository.Component{
					Name:    "revision",
					Kind:    repository.VersionKindRevision,
					Version: rev,
				})
			}

			return &repository.Resource{
				Name:           ApplicationName(app.Name),
				Kind:           repository.KindApplication,
				Namespace:      app.Namespace,
				ObjectName:     app.Name,
				Labels:         app.Labels,
				Annotations:    app.Annotations,
				Components:     comps,
				DesiredVersion: app.Spec.Source.TargetRevision,
				SyncRevision:   app.Status.Sync.Revision,
				Health:         argoHealth(app.Status.Health.Status),
				Destination: &repository.ClusterLocation{
					Server:    app.Spec.Destination.Server,
					Name:      app.Spec.Destination.Name,
					Namespace: app.Spec.Destination.Namespace,
				},
			}, nil
		},
	})
}

func NewDeploymentRepository(client dynamic.Interface) repository.ResourceRepository {
	return NewResourceRepository(ResourceRepositoryParams{
		Client: client,
		GroupVersionResource: schema.GroupVersionResource{
			Group:    "apps",
			Version:  "v1",
			Resource: "deployments",
		},
		ToResource: func(obj *unstructured.Unstructured) (*repository.Resource, error) {

			dep := &appsv1.Deployment{}

			err := DecodeMap(obj.Object, dep)

			if err != nil {
				return nil, err
			}

			parent := ""

			if mb, ok := dep.Labels[labelManagedBy]; ok && mb == "Helm" {
				parent = ApplicationName(dep.Labels[labelInstance])
			}

			comps := make([]repository.Component, 0, len(dep.Spec.Template.Spec.Containers))

			for _, c := range dep.Spec.Template.Spec.Containers {
				comps = append(comps, repository.Component{
					Name:    c.Name,
					Kind:    repository.VersionKindContainer,
					Version: ParseImageReference(c.Image).Version(),
					Image:   c.Image,
				})
			}

			return &repository.Resource{
				Name:         DeploymentName(dep.Namespace, dep.Name),
				Kind:         repository.KindDeployment,
				Namespace:    dep.Namespace,
				ObjectName:   dep.Name,
				Labels:       dep.Labels,
				Annotations:  dep.Annotations,
				Parent:       parent,
				Components:   comps,
				ChartVersion: ChartVersionFromLabel(dep.Labels[labelHelmChart]),
				Health:       deploymentHealth(dep),
			}, nil
		},
	})
}

func NewResources(client dynamic.Interface) *repository.Resources {
	return &repository.Resources{
		Applications: NewApplicationRepository(client),
		Deployment:   NewDeploymentRepository(client),
	}
}
