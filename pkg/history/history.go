// Package history records version changes per grid cell. The in-memory ring
// answers the API; durable records are Kubernetes Events emitted by the
// reconciler (see service.EventRecorder).
package history

import (
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/activatedio/deploygrid/pkg/deploygrid"
)

// DefaultCapacity is how many changes are kept per cell.
const DefaultCapacity = 100

// Change is one observed transition of a cell.
type Change struct {
	System      string    `json:"system"`
	Component   string    `json:"component"`
	Environment string    `json:"environment"`
	Cluster     string    `json:"cluster,omitempty"`
	Namespace   string    `json:"namespace,omitempty"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	Desired     string    `json:"desired,omitempty"`
	Health      string    `json:"health,omitempty"`
	ObservedAt  time.Time `json:"observed_at"`
}

// Kind classifies a change for consumers that render it.
func (c Change) Kind() string {
	switch {
	case c.From == "" && c.To != "":
		return "appeared"
	case c.To == "" && c.From != "":
		return "disappeared"
	default:
		return "changed"
	}
}

// Query filters history reads.
type Query struct {
	System      string
	Component   string // empty: every component of the system
	Environment string // empty: every environment
	Since       time.Time
	Limit       int
}

type cellKey struct {
	system, component, env string
}

type cellState struct {
	version string
	known   bool
	ring    []Change
}

// Ring tracks the last known version of each cell and keeps a bounded log
// of its changes.
type Ring struct {
	lock     sync.RWMutex
	capacity int
	cells    map[cellKey]*cellState
}

func New(capacity int) *Ring {
	if capacity <= 0 {
		capacity = DefaultCapacity
	}
	return &Ring{capacity: capacity, cells: map[cellKey]*cellState{}}
}

// Seed records the current versions of a grid without producing changes. The
// reconciler uses it while watches are still syncing after start-up, so
// cells that appear as their cluster connects are not reported as deployed.
func (r *Ring) Seed(g *deploygrid.Grid) {
	r.lock.Lock()
	defer r.lock.Unlock()
	for _, row := range flatten(g.Groups) {
		for env, cell := range row.Cells {
			k := cellKey{g.System.Name, row.Component.Name, env}
			if st := r.cells[k]; st != nil {
				st.version = cell.Version
			} else {
				r.cells[k] = &cellState{version: cell.Version, known: true}
			}
		}
	}
}

// Observe compares a grid against the last known versions and records every
// difference. It returns the changes it recorded. The first observation of
// a cell seeds its version without recording a change, so a server restart
// does not produce a flood of "appeared" entries.
func (r *Ring) Observe(g *deploygrid.Grid, now time.Time) []Change {
	r.lock.Lock()
	defer r.lock.Unlock()

	seen := map[cellKey]bool{}
	out := make([]Change, 0, 4)
	for _, row := range flatten(g.Groups) {
		for env, cell := range row.Cells {
			k := cellKey{g.System.Name, row.Component.Name, env}
			seen[k] = true
			st := r.cells[k]
			if st == nil {
				r.cells[k] = &cellState{version: cell.Version, known: true}
				continue
			}
			if st.version == cell.Version {
				continue
			}
			ch := Change{
				System: k.system, Component: k.component, Environment: k.env,
				Cluster: cell.Cluster, Namespace: cell.Namespace,
				From: st.version, To: cell.Version, Desired: cell.DesiredVersion, Health: cell.Health,
				ObservedAt: now,
			}
			st.version = cell.Version
			st.push(ch, r.capacity)
			out = append(out, ch)
		}
	}
	// cells that vanished from the grid
	for k, st := range r.cells {
		if k.system != g.System.Name || seen[k] || st.version == "" {
			continue
		}
		ch := Change{System: k.system, Component: k.component, Environment: k.env, From: st.version, ObservedAt: now}
		st.version = ""
		st.push(ch, r.capacity)
		out = append(out, ch)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Component+out[i].Environment < out[j].Component+out[j].Environment
	})
	return out
}

func (s *cellState) push(ch Change, capacity int) {
	s.ring = append(s.ring, ch)
	if len(s.ring) > capacity {
		s.ring = s.ring[len(s.ring)-capacity:]
	}
}

// List returns matching changes, newest first.
func (r *Ring) List(q Query) []Change {
	r.lock.RLock()
	defer r.lock.RUnlock()

	var out []Change
	for k, st := range r.cells {
		if !q.matches(k) {
			continue
		}
		for _, ch := range st.ring {
			if q.Since.IsZero() || !ch.ObservedAt.Before(q.Since) {
				out = append(out, ch)
			}
		}
	}
	slices.SortFunc(out, newestFirst)
	if q.Limit > 0 && len(out) > q.Limit {
		out = out[:q.Limit]
	}
	return out
}

func (q Query) matches(k cellKey) bool {
	return k.system == q.System &&
		(q.Component == "" || k.component == q.Component) &&
		(q.Environment == "" || k.env == q.Environment)
}

func newestFirst(a, b Change) int {
	switch {
	case a.ObservedAt.After(b.ObservedAt):
		return -1
	case a.ObservedAt.Before(b.ObservedAt):
		return 1
	default:
		return strings.Compare(a.Component+a.Environment, b.Component+b.Environment)
	}
}

func flatten(groups []*deploygrid.GridGroup) []*deploygrid.GridRow {
	var out []*deploygrid.GridRow
	var walk func(rows []*deploygrid.GridRow)
	walk = func(rows []*deploygrid.GridRow) {
		for _, r := range rows {
			out = append(out, r)
			walk(r.Children)
		}
	}
	for _, g := range groups {
		walk(g.Rows)
	}
	return out
}
