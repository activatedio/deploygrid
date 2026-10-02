package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/deploygrid"
	"github.com/activatedio/deploygrid/pkg/grid"
	"github.com/activatedio/deploygrid/pkg/repository"
	"github.com/activatedio/deploygrid/pkg/service"
)

// noClusters is an accessor for a server that watches nothing itself.
type noClusters struct{}

func (noClusters) ClusterNames(context.Context) []string { return nil }
func (noClusters) Get(context.Context, string) (*repository.Resources, error) {
	return nil, nil
}

func TestGridService_IndexInvalidation(t *testing.T) {
	reg := service.NewSourceRegistry()
	catalog := service.NewConfigCatalog(&config.ClustersConfig{
		Environments: []string{"dev"},
		Clusters:     []config.ClusterConfig{{Name: "edge", Mode: config.ClusterModeAgent, Token: "t"}},
	})
	svc := service.NewGridService(service.GridServiceParams{Catalog: catalog, Accessor: noClusters{}, Registry: reg})
	ctx := context.Background()

	first, err := svc.Grid(ctx, service.DefaultSystemName)
	require.NoError(t, err)
	again, err := svc.Grid(ctx, service.DefaultSystemName)
	require.NoError(t, err)
	assert.Same(t, first, again, "nothing changed: the cached grid is served")

	// a pushed observation changes the registry and invalidates the index
	st, _ := reg.Store("edge", repository.KindDeployment)
	require.NoError(t, st.Replace([]*repository.Resource{{
		Name: "namespaces/a/deployments/api", Kind: repository.KindDeployment, Namespace: "a", ObjectName: "api",
		Labels:     map[string]string{grid.LabelComponent: "api", grid.LabelEnvironment: "dev"},
		Components: []repository.Component{{Name: "api", Kind: repository.VersionKindContainer, Version: "1.0"}},
	}}))

	rebuilt, err := svc.Grid(ctx, service.DefaultSystemName)
	require.NoError(t, err)
	assert.NotSame(t, first, rebuilt)
	require.Len(t, rebuilt.Groups, 1)
	assert.Equal(t, "1.0", rebuilt.Groups[0].Rows[0].Cells["dev"].Version)

	cached, err := svc.Grid(ctx, service.DefaultSystemName)
	require.NoError(t, err)
	assert.Same(t, rebuilt, cached)

	// errors are part of the inputs too
	reg.SetError("edge", assert.AnError)
	withErr, err := svc.Grid(ctx, service.DefaultSystemName)
	require.NoError(t, err)
	assert.NotSame(t, cached, withErr)
	assert.Len(t, withErr.Errors, 1)

	rows, err := svc.Rows(ctx, service.DefaultSystemName)
	require.NoError(t, err)
	assert.Len(t, rows, 1)
	_, err = svc.Grid(ctx, "nope")
	require.Error(t, err)
	_ = deploygrid.Grid{}
}
