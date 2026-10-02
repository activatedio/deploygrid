package k8s

import (
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"

	"github.com/activatedio/deploygrid/pkg/repository"
)

// IngressName is the store key of an ingress.
func IngressName(namespace, name string) string {
	return "namespaces/" + namespace + "/ingresses/" + name
}

// NewIngressRepository observes networking.k8s.io/v1 Ingresses. An ingress
// contributes its hosts to the component it belongs to, either through the
// Helm release that manages it or through deploygrid labels.
func NewIngressRepository(client dynamic.Interface, disc discovery.ServerResourcesInterface) repository.ResourceRepository {
	return NewResourceRepository(ResourceRepositoryParams{
		Client:    client,
		Discovery: disc,
		GroupVersionResource: schema.GroupVersionResource{
			Group:    "networking.k8s.io",
			Version:  "v1",
			Resource: "ingresses",
		},
		ToResource: func(obj *unstructured.Unstructured) (*repository.Resource, error) {
			ing := &networkingv1.Ingress{}
			if err := DecodeMap(obj.Object, ing); err != nil {
				return nil, err
			}

			parent := parentOf(ing.ObjectMeta)

			seen := map[string]bool{}
			var hosts []string
			for _, rule := range ing.Spec.Rules {
				if rule.Host != "" && !seen[rule.Host] {
					seen[rule.Host] = true
					hosts = append(hosts, rule.Host)
				}
			}

			return &repository.Resource{
				Name:        IngressName(ing.Namespace, ing.Name),
				Kind:        repository.KindIngress,
				Namespace:   ing.Namespace,
				ObjectName:  ing.Name,
				Labels:      ing.Labels,
				Annotations: ing.Annotations,
				Parent:      parent,
				Hosts:       hosts,
			}, nil
		},
	})
}
