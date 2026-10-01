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

func TestFromSystemCR(t *testing.T) {
	in := &v1alpha1.System{
		ObjectMeta: metav1.ObjectMeta{Name: "apps"},
		Spec: v1alpha1.SystemSpec{
			Description: "desc",
			Environments: []v1alpha1.SystemEnvironment{
				{Name: "dev"},
				{Name: "qa", DisplayName: "QA"},
			},
			Groups: []v1alpha1.SystemGroup{{Name: "core", DisplayName: "Core"}},
		},
	}

	out := service.FromSystemCR(in)

	a := assert.New(t)
	a.Equal("apps", out.Name)
	a.Equal("apps", out.DisplayName, "display name defaults to the resource name")
	a.Equal("desc", out.Description)
	require.Len(t, out.Environments, 2)
	a.Equal("dev", out.Environments[0].DisplayName, "environment display name defaults to its name")
	a.Equal("QA", out.Environments[1].DisplayName)
	require.Len(t, out.Groups, 1)
	a.Equal("Core", out.Groups[0].DisplayName)
}

func TestConfigSystemService(t *testing.T) {
	unit := service.NewConfigSystemService(&config.ClustersConfig{Environments: []string{"Dev", "QA"}})
	ctx := context.Background()

	list, err := unit.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, service.DefaultSystemName, list[0].Name)
	require.Len(t, list[0].Environments, 2)
	assert.Equal(t, "QA", list[0].Environments[1].Name)

	got, err := unit.Get(ctx, service.DefaultSystemName)
	require.NoError(t, err)
	assert.Same(t, list[0], got)

	_, err = unit.Get(ctx, "nope")
	assert.ErrorIs(t, err, apiinframux.ErrNotFound)
}
