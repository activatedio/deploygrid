package service

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/activatedio/deploygrid/pkg/apis/deploygrid.activated.io/v1alpha1"
	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/configuration"
	"github.com/activatedio/deploygrid/pkg/grid"
	"github.com/activatedio/deploygrid/pkg/history"
	"github.com/activatedio/deploygrid/pkg/repository/k8s"
)

// StatusWriter persists the grid cells of declared Components into
// Component.status, a summary into System.status, connectivity into
// Cluster.status and template validity into ConfigurationView.status, and
// records version changes as Events on the Component. It is driven by the
// Reconciler.
type StatusWriter struct {
	registry    *SourceRegistry
	catalog     Catalog
	controllers *k8s.Controllers
	namespace   string
	// staleAfter is how long without a heartbeat before a pushed cluster is
	// reported as disconnected.
	staleAfter time.Duration
}

// ConditionConnected is set on Cluster resources.
const ConditionConnected = "Connected"

// ConditionTemplateValid is set on ConfigurationView resources.
const ConditionTemplateValid = "TemplateValid"

// Prepare runs before the grids are built in a pass.
func (w *StatusWriter) Prepare() {
	if err := EnsureTokenSecrets(w.controllers, w.namespace); err != nil {
		log.Error().Err(err).Msg("ensure collector tokens")
	}
}

// Apply records one System's results.
func (w *StatusWriter) Apply(systemName string, res *grid.Result, changes []history.Change, now time.Time) {
	ts := metav1.NewTime(now)
	w.syncSystem(systemName, res, ts)
	for name, statuses := range res.Statuses {
		w.syncComponent(name, statuses, ts)
	}
	for _, ch := range changes {
		w.recordEvent(ch)
	}
}

// Finish runs after every System was applied.
func (w *StatusWriter) Finish() {
	w.syncClusters(metav1.NewTime(time.Now()))
	w.syncViews()
}

// recordEvent emits a Kubernetes Event on the Component for a version
// change. Discovered components have no resource to attach to and are
// skipped.
func (w *StatusWriter) recordEvent(ch history.Change) {
	comp, err := w.controllers.ComponentsCache.Get(w.namespace, ch.Component)
	if err != nil {
		return
	}
	var message string
	switch ch.Kind() {
	case "appeared":
		message = fmt.Sprintf("%s: %s deployed", ch.Environment, ch.To)
	case "disappeared":
		message = fmt.Sprintf("%s: %s no longer observed", ch.Environment, ch.From)
	default:
		message = fmt.Sprintf("%s: %s -> %s", ch.Environment, ch.From, ch.To)
	}
	if ch.Cluster != "" {
		message += fmt.Sprintf(" (cluster %s", ch.Cluster)
		if ch.Namespace != "" {
			message += ", namespace " + ch.Namespace
		}
		message += ")"
	}
	ts := metav1.NewTime(ch.ObservedAt)
	ev := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: comp.Name + ".",
			Namespace:    w.namespace,
			Labels: map[string]string{
				grid.LabelSystem:      ch.System,
				grid.LabelComponent:   ch.Component,
				grid.LabelEnvironment: ch.Environment,
			},
		},
		InvolvedObject: corev1.ObjectReference{
			APIVersion:      v1alpha1.SchemeGroupVersion.String(),
			Kind:            "Component",
			Namespace:       comp.Namespace,
			Name:            comp.Name,
			UID:             comp.UID,
			ResourceVersion: comp.ResourceVersion,
		},
		Reason:              "VersionChanged",
		Message:             message,
		Type:                corev1.EventTypeNormal,
		Source:              corev1.EventSource{Component: "deploygrid"},
		FirstTimestamp:      ts,
		LastTimestamp:       ts,
		Count:               1,
		ReportingController: "deploygrid.activated.io/server",
		ReportingInstance:   "deploygrid",
	}
	if _, err := w.controllers.Events.Create(ev); err != nil {
		log.Error().Err(err).Str("component", ch.Component).Msg("record event")
	}
}

// syncViews validates every ConfigurationView template and records the
// result as a condition.
func (w *StatusWriter) syncViews() {
	views, err := w.catalog.ConfigurationViews()
	if err != nil {
		log.Error().Err(err).Msg("list views")
		return
	}
	for _, v := range views {
		cond := metav1.Condition{Type: ConditionTemplateValid, ObservedGeneration: v.Generation}
		if _, err := configuration.Parse(v); err != nil {
			cond.Status = metav1.ConditionFalse
			cond.Reason = "ParseError"
			cond.Message = err.Error()
		} else {
			cond.Status = metav1.ConditionTrue
			cond.Reason = "Parsed"
			cond.Message = "template parses"
		}
		updated := v.DeepCopy()
		meta.SetStatusCondition(&updated.Status.Conditions, cond)
		updated.Status.ObservedGeneration = v.Generation
		if equality.Semantic.DeepEqual(updated.Status, v.Status) {
			continue
		}
		if _, err := w.controllers.ConfigurationViews.UpdateStatus(updated); err != nil {
			log.Error().Err(err).Str("view", v.Name).Msg("update view status")
		}
	}
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
	Registry    *SourceRegistry
	Catalog     Catalog
	Controllers *k8s.Controllers
	Control     *config.ControlConfig
}

func NewStatusWriter(params StatusWriterParams) *StatusWriter {
	return &StatusWriter{
		registry:    params.Registry,
		catalog:     params.Catalog,
		controllers: params.Controllers,
		namespace:   params.Control.Namespace,
		staleAfter:  2 * time.Minute,
	}
}
