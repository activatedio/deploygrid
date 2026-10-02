package service

import (
	"context"

	"github.com/activatedio/deploygrid/pkg/apis/deploygrid.activated.io/v1alpha1"
	"github.com/activatedio/deploygrid/pkg/deploygrid"
)

// FromSystemCR converts a System custom resource into the API model.
func FromSystemCR(in *v1alpha1.System) *deploygrid.System {
	out := &deploygrid.System{
		Name:        in.Name,
		DisplayName: in.Spec.DisplayName,
		Description: in.Spec.Description,
	}
	if out.DisplayName == "" {
		out.DisplayName = in.Name
	}
	for _, e := range in.Spec.Environments {
		dn := e.DisplayName
		if dn == "" {
			dn = e.Name
		}
		out.Environments = append(out.Environments, &deploygrid.Environment{Name: e.Name, DisplayName: dn})
	}
	for _, g := range in.Spec.Groups {
		dn := g.DisplayName
		if dn == "" {
			dn = g.Name
		}
		out.Groups = append(out.Groups, &deploygrid.Group{Name: g.Name, DisplayName: dn})
	}
	return out
}

type systemService struct {
	catalog Catalog
}

func (s *systemService) List(_ context.Context) ([]*deploygrid.System, error) {
	items, err := s.catalog.Systems()
	if err != nil {
		return nil, err
	}
	res := make([]*deploygrid.System, 0, len(items))
	for _, item := range items {
		res = append(res, FromSystemCR(item))
	}
	return res, nil
}

func (s *systemService) Get(_ context.Context, name string) (*deploygrid.System, error) {
	item, err := s.catalog.System(name)
	if err != nil {
		return nil, err
	}
	return FromSystemCR(item), nil
}

func NewSystemService(catalog Catalog) SystemService {
	return &systemService{catalog: catalog}
}
