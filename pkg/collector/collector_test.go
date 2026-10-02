package collector_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/activatedio/deploygrid/pkg/collector"
	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/controller"
	"github.com/activatedio/deploygrid/pkg/repository"
	"github.com/activatedio/deploygrid/pkg/service"
)

// fakeRepository replays reflector-style events into whatever store Watch is
// given.
type fakeRepository struct {
	store repository.ResourceStore
	ready chan struct{}
}

func (f *fakeRepository) Watch(_ context.Context, store repository.ResourceStore) {
	f.store = store
	close(f.ready)
}

func dep(ns, name, version string) *repository.Resource {
	return &repository.Resource{
		Name: "namespaces/" + ns + "/deployments/" + name, Kind: repository.KindDeployment,
		Namespace: ns, ObjectName: name,
		Components: []repository.Component{{Name: "app", Kind: repository.VersionKindContainer, Version: version}},
	}
}

func app(name string) *repository.Resource {
	return &repository.Resource{Name: "applications/" + name, Kind: repository.KindApplication, ObjectName: name}
}

func TestCollector_PushesSnapshotsAndDeltas(t *testing.T) {
	// server side: real ingest behind the real HTTP handler
	reg := service.NewSourceRegistry()
	svc := service.NewObservationService(service.ObservationServiceParams{
		Registry: reg,
		Resolvers: []service.TokenResolver{service.NewConfigTokenResolver(&config.ClustersConfig{Clusters: []config.ClusterConfig{
			{Name: "edge", Mode: config.ClusterModeAgent, Token: "secret"},
		}})},
	})
	r := mux.NewRouter()
	r.HandleFunc("/api/observations", controller.NewObservations(svc).Post).Methods(http.MethodPost)
	srv := httptest.NewServer(r)
	defer srv.Close()

	deployments := &fakeRepository{ready: make(chan struct{})}
	applications := &fakeRepository{ready: make(chan struct{})}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := collector.New(collector.Options{
		Server:    srv.URL + "/api",
		Cluster:   "edge",
		Token:     "secret",
		Flush:     50 * time.Millisecond,
		Heartbeat: time.Hour,
		Resources: &repository.Resources{Applications: applications, Deployment: deployments},
	})
	go c.Run(ctx)
	<-deployments.ready
	<-applications.ready

	// initial lists arrive as Replace per kind
	require.NoError(t, deployments.store.Replace([]*repository.Resource{dep("a", "x", "1"), dep("a", "y", "1")}))
	require.NoError(t, applications.store.Replace([]*repository.Resource{app("one")}))

	require.Eventually(t, func() bool {
		data, _ := reg.Snapshot()
		return data["edge"] != nil && len(data["edge"].Entries()) == 3
	}, 5*time.Second, 20*time.Millisecond)

	// incremental changes
	require.NoError(t, deployments.store.Modify(dep("a", "x", "2")))
	require.NoError(t, deployments.store.Delete(dep("a", "y", "1")))
	require.NoError(t, deployments.store.Add(dep("b", "z", "1")))

	require.Eventually(t, func() bool {
		data, _ := reg.Snapshot()
		e := data["edge"].Entries()
		x, ok := e["namespaces/a/deployments/x"]
		return ok && x.Components[0].Version == "2" && e["namespaces/a/deployments/y"] == nil && e["namespaces/b/deployments/z"] != nil
	}, 5*time.Second, 20*time.Millisecond)

	// a collector-side error is surfaced on the grid even without resource
	// changes, and cleared when the kind lists cleanly again
	deployments.store.Error(assert.AnError)
	require.Eventually(t, func() bool {
		_, errs := reg.Snapshot()
		return len(errs) == 1
	}, 5*time.Second, 20*time.Millisecond)
	require.NoError(t, deployments.store.Replace([]*repository.Resource{dep("a", "x", "2"), dep("b", "z", "1")}))
	require.Eventually(t, func() bool {
		_, errs := reg.Snapshot()
		return len(errs) == 0
	}, 5*time.Second, 20*time.Millisecond)

	hb := reg.Heartbeats()["edge"]
	assert.True(t, hb.Pushed)
	assert.Equal(t, "dev", hb.CollectorVersion)
}

func TestCollector_WrongTokenKeepsRetrying(t *testing.T) {
	reg := service.NewSourceRegistry()
	svc := service.NewObservationService(service.ObservationServiceParams{Registry: reg})
	r := mux.NewRouter()
	r.HandleFunc("/api/observations", controller.NewObservations(svc).Post).Methods(http.MethodPost)
	srv := httptest.NewServer(r)
	defer srv.Close()

	deployments := &fakeRepository{ready: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := collector.New(collector.Options{
		Server: srv.URL + "/api", Cluster: "edge", Token: "wrong",
		Flush: 20 * time.Millisecond, Heartbeat: time.Hour,
		Resources: &repository.Resources{Deployment: deployments},
	})
	go c.Run(ctx)
	<-deployments.ready
	require.NoError(t, deployments.store.Replace([]*repository.Resource{dep("a", "x", "1")}))

	time.Sleep(200 * time.Millisecond)
	data, _ := reg.Snapshot()
	assert.Empty(t, data, "nothing is recorded for an unauthenticated collector")
}

// TestCollector_ResendsSnapshotAfterServerRestart simulates a server that
// loses its state (fresh registry) while the collector keeps running with
// nothing changing on the cluster: the next heartbeat must repopulate it.
func TestCollector_ResendsSnapshotAfterServerRestart(t *testing.T) {
	var reg atomic.Pointer[service.SourceRegistry]
	reg.Store(service.NewSourceRegistry())
	resolver := service.NewConfigTokenResolver(&config.ClustersConfig{Clusters: []config.ClusterConfig{
		{Name: "edge", Mode: config.ClusterModeAgent, Token: "secret"},
	}})
	var down atomic.Bool
	r := mux.NewRouter()
	r.HandleFunc("/api/observations", func(w http.ResponseWriter, req *http.Request) {
		if down.Load() {
			http.Error(w, "no healthy upstream", http.StatusServiceUnavailable)
			return
		}
		svc := service.NewObservationService(service.ObservationServiceParams{Registry: reg.Load(), Resolvers: []service.TokenResolver{resolver}})
		controller.NewObservations(svc).Post(w, req)
	}).Methods(http.MethodPost)
	srv := httptest.NewServer(r)
	defer srv.Close()

	deployments := &fakeRepository{ready: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := collector.New(collector.Options{
		Server: srv.URL + "/api", Cluster: "edge", Token: "secret",
		Flush: 30 * time.Millisecond, Heartbeat: 150 * time.Millisecond,
		Resources: &repository.Resources{Deployment: deployments},
	})
	go c.Run(ctx)
	<-deployments.ready
	require.NoError(t, deployments.store.Replace([]*repository.Resource{dep("a", "x", "1")}))
	require.Eventually(t, func() bool {
		data, _ := reg.Load().Snapshot()
		return data["edge"] != nil && len(data["edge"].Entries()) == 1
	}, 5*time.Second, 20*time.Millisecond, "initial snapshot lands")

	// server goes away and comes back with no state; the cluster is quiet
	down.Store(true)
	time.Sleep(400 * time.Millisecond)
	reg.Store(service.NewSourceRegistry())
	down.Store(false)

	require.Eventually(t, func() bool {
		data, _ := reg.Load().Snapshot()
		return data["edge"] != nil && len(data["edge"].Entries()) == 1
	}, 10*time.Second, 50*time.Millisecond, "the restarted server is repopulated without any cluster change or collector restart")
}
