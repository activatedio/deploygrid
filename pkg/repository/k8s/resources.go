package k8s

import (
	"github.com/activatedio/deploygrid/pkg/repository"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"strings"
)

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

			compName := app.Spec.Source.Chart
			if compName == "" {
				parts := strings.Split(app.Spec.Source.RepoURL, "/")
				compName = parts[len(parts)-1]
			}

			return &repository.Resource{
				Name: ApplicationName(app.Namespace, app.Name),
				ChildrenLocation: []repository.ClusterLocation{
					{
						Server:    app.Spec.Destination.Server,
						Namespace: app.Spec.Destination.Namespace,
					},
				},
				Components: []repository.Component{
					{
						DisplayName: app.Name,
						Version:     app.Spec.Source.TargetRevision,
					},
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

			if mb, ok := dep.Labels["app.kubernetes.io/managed-by"]; ok && mb == "Helm" {
				parent = ApplicationName(dep.Namespace, dep.Labels["app.kubernetes.io/instance"])
			}

			var comps []repository.Component

			for _, c := range dep.Spec.Template.Spec.Containers {

				parts := strings.Split(c.Image, ":")

				version := "latest"

				if len(parts) > 1 {
					version = parts[1]
				}

				comps = append(comps, repository.Component{
					DisplayName: c.Name,
					Version:     version,
				})
			}

			return &repository.Resource{
				Name:       DeploymentName(dep.Namespace, dep.Name),
				Parent:     parent,
				Components: comps,
			}, nil
		},
	})
}
