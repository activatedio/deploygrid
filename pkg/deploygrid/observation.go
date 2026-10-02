package deploygrid

import (
	"time"

	"github.com/activatedio/deploygrid/pkg/repository"
)

// Observation is what a collector pushes to POST /api/observations. A
// snapshot replaces every resource of the listed kinds for the cluster; a
// delta upserts Resources and deletes Removed. An observation with no
// changes is a heartbeat.
type Observation struct {
	// Cluster is informational; the authenticated token decides which
	// cluster the observation is recorded against.
	Cluster  string `json:"cluster,omitempty"`
	Snapshot bool   `json:"snapshot"`
	// Kinds covered by a snapshot. Required when Snapshot is true.
	Kinds     []string               `json:"kinds,omitempty"`
	Resources []*repository.Resource `json:"resources,omitempty"`
	// Removed carries the full resource so parent links can be unwound.
	Removed []*repository.Resource `json:"removed,omitempty"`
	// Errors are collector-side problems (for example a watch that cannot
	// list), shown on the grid.
	Errors    []string             `json:"errors,omitempty"`
	Heartbeat ObservationHeartbeat `json:"heartbeat"`
}

type ObservationHeartbeat struct {
	CollectorVersion  string    `json:"collector_version,omitempty"`
	KubernetesVersion string    `json:"kubernetes_version,omitempty"`
	SentAt            time.Time `json:"sent_at"`
}

// ObservationResponse acknowledges an observation. Resync asks the collector
// to send a full snapshot of every kind, for example after a server restart.
type ObservationResponse struct {
	Cluster  string `json:"cluster"`
	Accepted int    `json:"accepted"`
	Resync   bool   `json:"resync"`
}
