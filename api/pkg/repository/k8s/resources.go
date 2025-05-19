package k8s

import (
	"fmt"
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

			simpleName := app.Spec.Source.Chart

			return &repository.Resource{
				Name:   ApplicationName(app.Name),
				Labels: app.Labels,
				Components: []repository.Component{
					{
						Name:        ApplicationName(app.Name),
						SimpleName:  simpleName,
						DisplayName: app.Name,
						Type:        "Application Helm Chart",
						Version:     app.Spec.Source.TargetRevision,
						PathElement: fmt.Sprintf("applicationcharts/%s", simpleName),
						ChildrenLocation: []repository.ClusterLocation{
							{
								Server: app.Spec.Destination.Server,
							},
						},
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
				parent = ApplicationName(dep.Labels["app.kubernetes.io/instance"])
			}

			var comps []repository.Component

			for _, c := range dep.Spec.Template.Spec.Containers {

				parts := strings.Split(c.Image, ":")

				version := "latest"

				if len(parts) > 1 {
					version = parts[1]
				}

				pathElement := fmt.Sprintf("deployments/%s/containers/%s", dep.Name, c.Name)
				name := fmt.Sprintf("namespaces/%s/%s", dep.Namespace, pathElement)
				simpleName := fmt.Sprintf("%s/%s", dep.Name, c.Name)

				comps = append(comps, repository.Component{
					Name:        name,
					SimpleName:  simpleName,
					DisplayName: simpleName,
					Type:        "Container",
					Version:     version,
					PathElement: pathElement,
				})
			}

			return &repository.Resource{
				Name:       DeploymentName(dep.Namespace, dep.Name),
				Labels:     dep.Labels,
				Parent:     parent,
				Components: comps,
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
