package service

import (
	"context"

	"github.com/activatedio/deploygrid/pkg/deploygrid"
)

type GridService interface {
	Init()
	Get(ctx context.Context) (*deploygrid.Grid, error)
}

type Metadata map[string]any

type MetadataCriteria struct {
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
