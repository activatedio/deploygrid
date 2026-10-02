package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiinframux "github.com/activatedio/deploygrid/pkg/apiinfra/mux"
	"github.com/activatedio/deploygrid/pkg/apis/deploygrid.activated.io/v1alpha1"
	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/service"
)

func TestConfigCatalogAndSystemService(t *testing.T) {
	cat := service.NewConfigCatalog(&config.ClustersConfig{
		Environments: []string{"Dev", "QA"},
		Clusters: []config.ClusterConfig{
			{Name: "remote", Address: "https://remote:6443"},
			{Name: "here", Local: true},
		},
	})
	unit := service.NewSystemService(cat)
	ctx := context.Background()

	list, err := unit.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, service.DefaultSystemName, list[0].Name)
	require.Len(t, list[0].Environments, 2)
	assert.Equal(t, "QA", list[0].Environments[1].Name)

	_, err = unit.Get(ctx, "nope")
	require.ErrorIs(t, err, apiinframux.ErrNotFound)

	comps, err := cat.Components()
	require.NoError(t, err)
	assert.Empty(t, comps)

	clusters, err := cat.Clusters()
	require.NoError(t, err)
	require.Len(t, clusters, 2)
	assert.Equal(t, []string{"https://remote:6443"}, clusters[0].Addresses)
	assert.Equal(t, []string{service.InClusterAddress}, clusters[1].Addresses, "local clusters answer to the in-cluster address")
}

func TestFromSystemCR(t *testing.T) {
	in := &v1alpha1.System{
		ObjectMeta: metav1.ObjectMeta{Name: "apps"},
		Spec: v1alpha1.SystemSpec{
			Environments: []v1alpha1.SystemEnvironment{{Name: "dev"}, {Name: "qa", DisplayName: "QA"}},
			Groups:       []v1alpha1.SystemGroup{{Name: "core"}},
		},
	}
	out := service.FromSystemCR(in)
	a := assert.New(t)
	a.Equal("apps", out.DisplayName, "display name defaults to the resource name")
	require.Len(t, out.Environments, 2)
	a.Equal("dev", out.Environments[0].DisplayName)
	a.Equal("QA", out.Environments[1].DisplayName)
	require.Len(t, out.Groups, 1)
	a.Equal("core", out.Groups[0].DisplayName)
}
