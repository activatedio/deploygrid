package service

import (
	"context"

	"github.com/activatedio/deploygrid/pkg/deploygrid"
)

type System struct {
	Name        string
	Description string
}

type GridService interface {
	Init()
	ListSystems(ctx context.Context) ([]System, error)
	GetSystem(ctx context.Context, system string) (*deploygrid.Grid, error)
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
