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
		arrange func() (context.Context, repository.ComponentCriteria)
		assert  func(got []repository.Component, err error)
	}

	cases := map[string]s{
		"argocd applications": {
			arrange: func() (context.Context, repository.ComponentCriteria) {
				return context.TODO(), repository.ComponentCriteria{
					Namespace:  "argocd",
					ApiVersion: "argoproj.io/v1alpha1",
					Kind:       "applications",
				}
			},
			assert: func(got []repository.Component, err error) {
				a.Nil(err)
				// TODO - actually validate this
				a.Len(got, 4)
			},
		},
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", "../../../.kind/kubeconfig-ops-cluster-1.yaml")

	util.Check(err)

	cl, err := dynamic.NewForConfig(cfg)

	util.Check(err)

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {

			unit := k8s.NewComponentRepository("test", cl)
			v.assert(unit.List(v.arrange()))

		})
	}
}
