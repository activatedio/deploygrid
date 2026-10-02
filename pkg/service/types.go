package service

import (
	"context"

	"github.com/activatedio/deploygrid/pkg/apis/deploygrid.activated.io/v1alpha1"
	"github.com/activatedio/deploygrid/pkg/deploygrid"
	"github.com/activatedio/deploygrid/pkg/grid"
	"github.com/activatedio/deploygrid/pkg/history"
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
	Configurations() ([]*v1alpha1.Configuration, error)
	ConfigurationViews() ([]*v1alpha1.ConfigurationView, error)
	// Version is a change counter over every declared resource; a static
	// catalog returns a constant.
	Version() uint64
}

// HistoryService answers version-change queries.
type HistoryService interface {
	List(ctx context.Context, q history.Query) ([]history.Change, error)
}

// ConfigurationService merges Configurations and renders views.
type ConfigurationService interface {
	// Names lists the configuration names of a system with the environments
	// that override them.
	Names(ctx context.Context, system string) ([]*deploygrid.ConfigurationInfo, error)
	// Values returns the merged document for one configuration name.
	Values(ctx context.Context, system, name, environment string) (any, error)
	// Views lists the views of a system.
	Views(ctx context.Context, system string) ([]*deploygrid.ViewInfo, error)
	// Render executes a view for an environment.
	Render(ctx context.Context, system, view, environment string) (content []byte, contentType string, err error)
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
