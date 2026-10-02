package service

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/activatedio/deploygrid/pkg/store"
)

// Heartbeat is the last sign of life from a cluster's collector.
type Heartbeat struct {
	LastSeen          time.Time
	CollectorVersion  string
	KubernetesVersion string
	// Pushed is true when the cluster is fed by a collector rather than a
	// server-side watch.
	Pushed bool
}

// SourceRegistry holds the observed state of every cluster, however it
// arrives: server-side watches (pull) register one store per resource kind,
// collectors (push) are given one store per kind by the ingest service.
type SourceRegistry struct {
	// version increments on every change to any store, error or heartbeat
	// that affects a grid; the grid service uses it to invalidate its cache.
	version    atomic.Uint64
	lock       sync.RWMutex
	stores     map[string]map[string]*store.Store // cluster → kind → store
	errors     map[string]string                  // cluster → connection error
	pushErrors map[string][]string                // cluster → collector-reported errors
	heartbeats map[string]Heartbeat
}

func NewSourceRegistry() *SourceRegistry {
	return &SourceRegistry{
		stores:     map[string]map[string]*store.Store{},
		errors:     map[string]string{},
		pushErrors: map[string][]string{},
		heartbeats: map[string]Heartbeat{},
	}
}

// Store returns the store for a cluster and kind, creating it on first use.
// The boolean reports whether it already existed.
func (r *SourceRegistry) Store(cluster, kind string) (*store.Store, bool) {
	r.lock.Lock()
	defer r.lock.Unlock()
	byKind, ok := r.stores[cluster]
	if !ok {
		byKind = map[string]*store.Store{}
		r.stores[cluster] = byKind
	}
	if st, ok := byKind[kind]; ok {
		return st, true
	}
	st := store.NewStore()
	st.OnChange(func() { r.version.Add(1) })
	byKind[kind] = st
	r.version.Add(1)
	return st, false
}

// Version is a change counter for everything that feeds a grid.
func (r *SourceRegistry) Version() uint64 {
	return r.version.Load()
}

// Connected reports whether any store exists for the cluster.
func (r *SourceRegistry) Connected(cluster string) bool {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return len(r.stores[cluster]) > 0
}

// SetError records a cluster-level problem (connection refused, bad
// kubeconfig). ClearError removes it once the cluster is reachable.
func (r *SourceRegistry) SetError(cluster string, err error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.errors[cluster] != err.Error() {
		r.errors[cluster] = err.Error()
		r.version.Add(1)
	}
}

func (r *SourceRegistry) ClearError(cluster string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if _, ok := r.errors[cluster]; ok {
		delete(r.errors, cluster)
		r.version.Add(1)
	}
}

// HasError reports whether a cluster-level error is recorded.
func (r *SourceRegistry) HasError(cluster string) bool {
	r.lock.RLock()
	defer r.lock.RUnlock()
	_, ok := r.errors[cluster]
	return ok
}

// SetPushErrors replaces the collector-reported errors for a cluster.
func (r *SourceRegistry) SetPushErrors(cluster string, errs []string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if len(errs) == 0 {
		if _, ok := r.pushErrors[cluster]; ok {
			delete(r.pushErrors, cluster)
			r.version.Add(1)
		}
		return
	}
	r.pushErrors[cluster] = errs
	r.version.Add(1)
}

func (r *SourceRegistry) RecordHeartbeat(cluster string, hb Heartbeat) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.heartbeats[cluster] = hb
}

func (r *SourceRegistry) Heartbeats() map[string]Heartbeat {
	r.lock.RLock()
	defer r.lock.RUnlock()
	out := make(map[string]Heartbeat, len(r.heartbeats))
	for k, v := range r.heartbeats {
		out[k] = v
	}
	return out
}

// Snapshot merges every store of every cluster and collects errors.
func (r *SourceRegistry) Snapshot() (map[string]*store.StoreData, []string) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	errs := make([]string, 0, len(r.errors)+len(r.pushErrors))
	for cluster, msg := range r.errors {
		errs = append(errs, fmt.Sprintf("[Connect to cluster %s]: %s", cluster, msg))
	}
	for cluster, msgs := range r.pushErrors {
		for _, m := range msgs {
			errs = append(errs, fmt.Sprintf("[collector %s]: %s", cluster, m))
		}
	}

	data := map[string]*store.StoreData{}
	for cluster, byKind := range r.stores {
		merged := store.NewStoreData()
		for kind, st := range byKind {
			d, err := st.GetData()
			if err != nil {
				errs = append(errs, fmt.Sprintf("[cluster %s %s]: %s", cluster, kind, err.Error()))
			}
			merged.AddAll(d)
		}
		data[cluster] = merged
	}
	return data, errs
}
