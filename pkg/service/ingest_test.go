package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/deploygrid"
	"github.com/activatedio/deploygrid/pkg/repository"
	"github.com/activatedio/deploygrid/pkg/service"
)

func newIngest(t *testing.T) (service.ObservationService, *service.SourceRegistry) {
	t.Helper()
	reg := service.NewSourceRegistry()
	resolver := service.NewConfigTokenResolver(&config.ClustersConfig{Clusters: []config.ClusterConfig{
		{Name: "edge", Mode: config.ClusterModeAgent, Token: "secret"},
	}})
	svc := service.NewObservationService(service.ObservationServiceParams{
		Registry:  reg,
		Resolvers: []service.TokenResolver{resolver},
	})
	return svc, reg
}

func dep(ns, name, version string) *repository.Resource {
	return &repository.Resource{
		Name: "namespaces/" + ns + "/deployments/" + name, Kind: repository.KindDeployment,
		Namespace: ns, ObjectName: name,
		Components: []repository.Component{{Name: "app", Kind: repository.VersionKindContainer, Version: version}},
	}
}

func TestObservationService_Authenticate(t *testing.T) {
	svc, _ := newIngest(t)
	ctx := context.Background()

	cluster, err := svc.Authenticate(ctx, "secret")
	require.NoError(t, err)
	assert.Equal(t, "edge", cluster)

	_, err = svc.Authenticate(ctx, "nope")
	require.ErrorIs(t, err, service.ErrUnauthorized)
	_, err = svc.Authenticate(ctx, "")
	require.ErrorIs(t, err, service.ErrUnauthorized)
}

func TestObservationService_SnapshotThenDelta(t *testing.T) {
	svc, reg := newIngest(t)
	ctx := context.Background()

	// a delta before any snapshot is recorded but asks for a resync
	resp, err := svc.Ingest(ctx, "edge", &deploygrid.Observation{Resources: []*repository.Resource{dep("a", "x", "1")}})
	require.NoError(t, err)
	assert.True(t, resp.Resync)

	resp, err = svc.Ingest(ctx, "edge", &deploygrid.Observation{
		Snapshot: true, Kinds: []string{repository.KindDeployment},
		Resources: []*repository.Resource{dep("a", "x", "2"), dep("a", "y", "1")},
		Errors:    []string{"ingress: forbidden"},
	})
	require.NoError(t, err)
	assert.False(t, resp.Resync)
	assert.Equal(t, 2, resp.Accepted)

	data, errs := reg.Snapshot()
	require.Len(t, data["edge"].Entries(), 2)
	assert.Equal(t, "2", data["edge"].Entries()["namespaces/a/deployments/x"].Components[0].Version)
	assert.Equal(t, []string{"[collector edge]: ingress: forbidden"}, errs)

	resp, err = svc.Ingest(ctx, "edge", &deploygrid.Observation{
		Resources: []*repository.Resource{dep("a", "z", "1")},
		Removed:   []*repository.Resource{dep("a", "y", "1")},
	})
	require.NoError(t, err)
	assert.False(t, resp.Resync)
	assert.Equal(t, 2, resp.Accepted)

	data, errs = reg.Snapshot()
	assert.Empty(t, errs, "errors clear when a push reports none")
	entries := data["edge"].Entries()
	require.Len(t, entries, 2)
	assert.Contains(t, entries, "namespaces/a/deployments/x")
	assert.Contains(t, entries, "namespaces/a/deployments/z")

	hb := reg.Heartbeats()["edge"]
	assert.True(t, hb.Pushed)
	assert.False(t, hb.LastSeen.IsZero())

	_, err = svc.Ingest(ctx, "edge", &deploygrid.Observation{Snapshot: true})
	require.Error(t, err, "snapshot must list kinds")
}

func TestObservationService_EmptyClusterAsksForResync(t *testing.T) {
	svc, _ := newIngest(t)
	ctx := context.Background()

	// a heartbeat (no changes) from a cluster the server holds nothing for
	resp, err := svc.Ingest(ctx, "edge", &deploygrid.Observation{})
	require.NoError(t, err)
	assert.True(t, resp.Resync, "a restarted server must ask for a snapshot even when nothing changed")

	resp, err = svc.Ingest(ctx, "edge", &deploygrid.Observation{Snapshot: true, Kinds: []string{repository.KindDeployment}})
	require.NoError(t, err)
	assert.False(t, resp.Resync)

	resp, err = svc.Ingest(ctx, "edge", &deploygrid.Observation{})
	require.NoError(t, err)
	assert.False(t, resp.Resync, "once state exists, heartbeats are plain acknowledgements")
}
