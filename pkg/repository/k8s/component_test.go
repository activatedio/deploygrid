package k8s_test

import (
	"context"
	"github.com/activatedio/deploygrid/pkg/apiinfra/util"
	"github.com/activatedio/deploygrid/pkg/repository"
	"github.com/activatedio/deploygrid/pkg/repository/k8s"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"testing"
)

func TestComponentRepository_List(t *testing.T) {

	a := assert.New(t)

	type s struct {
		arrange func() (context.Context, dynamic.Interface, repository.ComponentCriteria)
		assert  func(got []repository.Component, err error)
	}

	makeClient := func(path string) dynamic.Interface {
		cfg, err := clientcmd.BuildConfigFromFlags("", path)
		util.Check(err)
		cl, err := dynamic.NewForConfig(cfg)
		util.Check(err)
		return cl
	}

	opsCl := makeClient("../../../.kind/kubeconfig-ops-cluster-1.yaml")
	apps1Cl := makeClient("../../../.kind/kubeconfig-app-cluster-1.yaml")

	cases := map[string]s{
		"argocd applications": {
			arrange: func() (context.Context, dynamic.Interface, repository.ComponentCriteria) {
				return context.TODO(), opsCl, repository.ComponentCriteria{
					Namespace:  "argocd",
					ApiVersion: "argoproj.io/v1alpha1",
					Kind:       "applications",
				}
			},
			assert: func(got []repository.Component, err error) {
				a.Nil(err)
				// TODO - actually validate this
				a.Len(got, 6)
			},
		},
		"deployments": {
			arrange: func() (context.Context, dynamic.Interface, repository.ComponentCriteria) {
				return context.TODO(), apps1Cl, repository.ComponentCriteria{
					Namespace:     "dev-app-a",
					ApiVersion:    "apps/v1",
					Kind:          "deployments",
					LabelSelector: "app.kubernetes.io/instance=dev-app-a,app.kubernetes.io/managed-by=Helm",
				}
			},
			assert: func(got []repository.Component, err error) {
				a.Nil(err)
				// TODO - actually validate this
				a.Len(got, 2)
			},
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {

			ctx, cl, c := v.arrange()

			unit := k8s.NewComponentRepository("test", cl)
			v.assert(unit.List(ctx, c))

		})
	}
}
