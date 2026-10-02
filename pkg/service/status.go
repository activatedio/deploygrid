package service

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/activatedio/deploygrid/pkg/apis/deploygrid.activated.io/v1alpha1"
	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/grid"
	"github.com/activatedio/deploygrid/pkg/repository/k8s"
)

// StatusWriter periodically persists the grid cells of declared Components
// into Component.status and a summary into System.status, so the grid is
// visible through kubectl and consumable by other controllers.
type StatusWriter struct {
	grids       GridService
	registry    *SourceRegistry
	controllers *k8s.Controllers
	namespace   string
	interval    time.Duration
	// staleAfter is how long without a heartbeat before a pushed cluster is
	// reported as disconnected.
	staleAfter time.Duration
	cancel     context.CancelFunc
}

// ConditionConnected is set on Cluster resources.
const ConditionConnected = "Connected"

func (w *StatusWriter) run(ctx context.Context) {
	t := time.NewTicker(w.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := w.Sync(ctx); err != nil {
				log.Error().Err(err).Msg("status sync failed")
			}
		}
	}
}

// Sync writes one round of status updates.
func (w *StatusWriter) Sync(ctx context.Context) error {
	if err := EnsureTokenSecrets(w.controllers, w.namespace); err != nil {
		log.Error().Err(err).Msg("ensure collector tokens")
	}
	results, err := w.grids.BuildAll(ctx)
	if err != nil {
		return err
	}
	now := metav1.NewTime(time.Now())
	for systemName, res := range results {
		w.syncSystem(systemName, res, now)
		for name, statuses := range res.Statuses {
			w.syncComponent(name, statuses, now)
		}
	}
	w.syncClusters(now)
	return nil
}

// syncClusters records connectivity on every Cluster resource: pushed
// clusters from their heartbeat, pulled clusters from the registry.
func (w *StatusWriter) syncClusters(now metav1.Time) {
	clusters, err := w.controllers.ClustersCache.List(w.namespace, labels.Everything())
	if err != nil {
		log.Error().Err(err).Msg("list clusters")
		return
	}
	heartbeats := w.registry.Heartbeats()
	for _, c := range clusters {
		cond := metav1.Condition{Type: ConditionConnected, ObservedGeneration: c.Generation}
		updated := c.DeepCopy()
		if hb, ok := heartbeats[c.Name]; ok && hb.Pushed {
			updated.Status.LastHeartbeatTime = &metav1.Time{Time: hb.LastSeen}
			updated.Status.CollectorVersion = hb.CollectorVersion
			updated.Status.KubernetesVersion = hb.KubernetesVersion
			if now.Sub(hb.LastSeen) > w.staleAfter {
				cond.Status = metav1.ConditionFalse
				cond.Reason = "HeartbeatStale"
				cond.Message = "no observation received since " + hb.LastSeen.UTC().Format(time.RFC3339)
			} else {
				cond.Status = metav1.ConditionTrue
				cond.Reason = "Receiving"
				cond.Message = "collector is pushing observations"
			}
		} else {
			switch {
			case w.registry.HasError(c.Name):
				cond.Status = metav1.ConditionFalse
				cond.Reason = "ConnectionFailed"
				cond.Message = "the server could not connect; see grid errors"
			case w.registry.Connected(c.Name):
				cond.Status = metav1.ConditionTrue
				cond.Reason = "Watching"
				cond.Message = "the server is watching the cluster"
			default:
				cond.Status = metav1.ConditionUnknown
				cond.Reason = "NoSource"
				cond.Message = "no watch configured and no collector has reported"
			}
		}
		meta.SetStatusCondition(&updated.Status.Conditions, cond)
		updated.Status.ObservedGeneration = c.Generation
		if equality.Semantic.DeepEqual(updated.Status, c.Status) {
			continue
		}
		if _, err := w.controllers.Clusters.UpdateStatus(updated); err != nil {
			log.Error().Err(err).Str("cluster", c.Name).Msg("update cluster status")
		}
	}
}

func (w *StatusWriter) syncSystem(name string, res *grid.Result, now metav1.Time) {
	sys, err := w.controllers.SystemsCache.Get(w.namespace, name)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error().Err(err).Str("system", name).Msg("get system")
		}
		return
	}
	count := int32(len(flattenRows(res.Grid.Groups))) //nolint:gosec // row counts are small
	if sys.Status.Components == count && sys.Status.ObservedGeneration == sys.Generation {
		return
	}
	updated := sys.DeepCopy()
	updated.Status.Components = count
	updated.Status.ObservedGeneration = sys.Generation
	updated.Status.LastObservedTime = &now
	if _, err := w.controllers.Systems.UpdateStatus(updated); err != nil {
		log.Error().Err(err).Str("system", name).Msg("update system status")
	}
}

// statusChanged compares cells ignoring timestamps.
func statusChanged(old, cur []v1alpha1.ComponentEnvironmentStatus) bool {
	if len(old) != len(cur) {
		return true
	}
	byEnv := map[string]v1alpha1.ComponentEnvironmentStatus{}
	for _, o := range old {
		byEnv[o.Environment] = o
	}
	for _, c := range cur {
		o, ok := byEnv[c.Environment]
		if !ok || !cellEqual(o, c) {
			return true
		}
	}
	return false
}

func cellEqual(a, b v1alpha1.ComponentEnvironmentStatus) bool {
	return a.Version == b.Version && a.DesiredVersion == b.DesiredVersion &&
		a.Cluster == b.Cluster && a.Namespace == b.Namespace && a.Health == b.Health &&
		equalStrings(a.Hosts, b.Hosts)
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (w *StatusWriter) syncComponent(name string, cur []v1alpha1.ComponentEnvironmentStatus, now metav1.Time) {
	comp, err := w.controllers.ComponentsCache.Get(w.namespace, name)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error().Err(err).Str("component", name).Msg("get component")
		}
		return
	}
	if !statusChanged(comp.Status.Environments, cur) && comp.Status.ObservedGeneration == comp.Generation {
		return
	}

	old := map[string]v1alpha1.ComponentEnvironmentStatus{}
	for _, o := range comp.Status.Environments {
		old[o.Environment] = o
	}
	for i := range cur {
		c := &cur[i]
		c.LastObservedTime = &now
		if o, ok := old[c.Environment]; ok && o.Version == c.Version {
			c.LastChangeTime = o.LastChangeTime
		} else {
			c.LastChangeTime = &now
		}
	}

	updated := comp.DeepCopy()
	updated.Status.Environments = cur
	updated.Status.ObservedGeneration = comp.Generation
	if _, err := w.controllers.Components.UpdateStatus(updated); err != nil {
		log.Error().Err(err).Str("component", name).Msg("update component status")
	}
}

type StatusWriterParams struct {
	fx.In
	Grids       GridService
	Registry    *SourceRegistry
	Controllers *k8s.Controllers
	Control     *config.ControlConfig
	Lifecycle   fx.Lifecycle
}

// NewStatusWriter starts the writer with the application lifecycle.
func NewStatusWriter(params StatusWriterParams) *StatusWriter {
	w := &StatusWriter{
		grids:       params.Grids,
		registry:    params.Registry,
		controllers: params.Controllers,
		namespace:   params.Control.Namespace,
		interval:    15 * time.Second,
		staleAfter:  2 * time.Minute,
	}
	params.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Tokens must exist before the first collector tries to connect.
			if err := EnsureTokenSecrets(w.controllers, w.namespace); err != nil {
				log.Error().Err(err).Msg("ensure collector tokens")
			}
			runCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
			w.cancel = cancel
			go w.run(runCtx)
			return nil
		},
		OnStop: func(_ context.Context) error {
			if w.cancel != nil {
				w.cancel()
			}
			return nil
		},
	})
	return w
}
