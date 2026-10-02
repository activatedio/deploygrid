package service

import (
	"context"

	"github.com/activatedio/deploygrid/pkg/apis/deploygrid.activated.io/v1alpha1"
	"github.com/activatedio/deploygrid/pkg/deploygrid"
	"github.com/activatedio/deploygrid/pkg/grid"
)

// Catalog supplies the declared half of the model: Systems, Components and
// Clusters. It is backed by custom resources when a control cluster is
// configured, otherwise synthesised from configuration.
type Catalog interface {
	Systems() ([]*v1alpha1.System, error)
	// System returns apiinframux.ErrNotFound (wrapped) when unknown.
	System(name string) (*v1alpha1.System, error)
	Components() ([]*v1alpha1.Component, error)
	Clusters() ([]grid.ClusterInfo, error)
}

// SystemService lists the Systems (grids) known to this server.
type SystemService interface {
	List(ctx context.Context) ([]*deploygrid.System, error)
	// Get returns apiinframux.ErrNotFound (wrapped) when the system is unknown.
	Get(ctx context.Context, name string) (*deploygrid.System, error)
}

// GridService computes grids from declared and observed state.
type GridService interface {
	Init()
	Grid(ctx context.Context, system string) (*deploygrid.Grid, error)
	Rows(ctx context.Context, system string) ([]*deploygrid.GridRow, error)
	Row(ctx context.Context, system, component string) (*deploygrid.GridRow, error)
	Unassigned(ctx context.Context, system string) ([]*deploygrid.Artifact, error)
	// BuildAll builds every System; used by the status writer.
	BuildAll(ctx context.Context) (map[string]*grid.Result, error)
}

type Metadata map[string]any

type MetadataCriteria struct {
	System      string
	Environment string
	Path        string
}

type MetadataService interface {
	Set(ctx context.Context, criteria MetadataCriteria, metadata Metadata) error
	Get(ctx context.Context, criteria MetadataCriteria) (Metadata, error)
}

// ResponseWriter abstracts simple operations for an http response writer
type ResponseWriter interface {
	Write([]byte) (int, error)
	SetContentType(string)
}

// MetadataRenderer renders metadata for the given view name
type MetadataRenderer interface {
	Render(ctxt context.Context, rw ResponseWriter, criteria MetadataCriteria, viewName string) error
}
