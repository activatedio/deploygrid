package service

import (
	"context"

	"github.com/activatedio/deploygrid/pkg/deploygrid"
)

// SystemService lists the Systems (grids) known to this server.
type SystemService interface {
	List(ctx context.Context) ([]*deploygrid.System, error)
	// Get returns apiinframux.ErrNotFound (wrapped) when the system is unknown.
	Get(ctx context.Context, name string) (*deploygrid.System, error)
}

type GridService interface {
	Init()
	Get(ctx context.Context) (*deploygrid.Grid, error)
}

type Metadata map[string]any

type MetadataCriteria struct {
	System      string
	Environment string
	Path        string
	// TODO - eventually we can have a selector
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
