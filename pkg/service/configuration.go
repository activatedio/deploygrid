package service

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	apiinframux "github.com/activatedio/deploygrid/pkg/apiinfra/mux"
	"github.com/activatedio/deploygrid/pkg/apis/deploygrid.activated.io/v1alpha1"
	"github.com/activatedio/deploygrid/pkg/configuration"
	"github.com/activatedio/deploygrid/pkg/deploygrid"
)

type configurationService struct {
	catalog Catalog
	grids   GridService
}

func (s *configurationService) forSystem(system string) ([]*v1alpha1.Configuration, error) {
	if _, err := s.catalog.System(system); err != nil {
		return nil, err
	}
	all, err := s.catalog.Configurations()
	if err != nil {
		return nil, err
	}
	var out []*v1alpha1.Configuration
	for _, c := range all {
		if c.Spec.System == system {
			out = append(out, c)
		}
	}
	return out, nil
}

func (s *configurationService) Names(_ context.Context, system string) ([]*deploygrid.ConfigurationInfo, error) {
	cfgs, err := s.forSystem(system)
	if err != nil {
		return nil, err
	}
	byName := map[string]*deploygrid.ConfigurationInfo{}
	for _, c := range cfgs {
		name := logicalName(c)
		info := byName[name]
		if info == nil {
			info = &deploygrid.ConfigurationInfo{Name: name, Environments: []string{}}
			byName[name] = info
		}
		if c.Spec.Environment == "" {
			info.SystemWide = true
		} else {
			info.Environments = append(info.Environments, c.Spec.Environment)
		}
	}
	out := make([]*deploygrid.ConfigurationInfo, 0, len(byName))
	for _, info := range byName {
		slices.Sort(info.Environments)
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// documents returns the system-wide and environment documents of one name.
// Configurations are named <name> for the system-wide document and
// <name>-<environment> for overrides; a document is matched by either its
// resource name or that convention.
func (s *configurationService) documents(system, name, environment string) (systemWide, env *v1alpha1.Configuration, err error) {
	cfgs, err := s.forSystem(system)
	if err != nil {
		return nil, nil, err
	}
	for _, c := range cfgs {
		if c.Name == name && c.Spec.Environment == "" {
			systemWide = c
		} else if isOverride(c, name, environment) {
			env = c
		}
	}
	if systemWide == nil && env == nil {
		return nil, nil, fmt.Errorf("configuration %q in system %q: %w", name, system, apiinframux.ErrNotFound)
	}
	return systemWide, env, nil
}

// logicalName strips the "-<environment>" suffix an override document
// carries by convention, so "endpoints-qa" lists under "endpoints".
func logicalName(c *v1alpha1.Configuration) string {
	if c.Spec.Environment != "" {
		return strings.TrimSuffix(c.Name, "-"+c.Spec.Environment)
	}
	return c.Name
}

func isOverride(c *v1alpha1.Configuration, name, environment string) bool {
	return environment != "" && c.Spec.Environment == environment &&
		(c.Name == name || c.Name == name+"-"+environment)
}

func (s *configurationService) Values(_ context.Context, system, name, environment string) (any, error) {
	systemWide, env, err := s.documents(system, name, environment)
	if err != nil {
		return nil, err
	}
	return configuration.Merge(systemWide, env)
}

func (s *configurationService) viewsFor(system string) ([]*v1alpha1.ConfigurationView, error) {
	if _, err := s.catalog.System(system); err != nil {
		return nil, err
	}
	all, err := s.catalog.ConfigurationViews()
	if err != nil {
		return nil, err
	}
	var out []*v1alpha1.ConfigurationView
	for _, v := range all {
		if v.Spec.System == system {
			out = append(out, v)
		}
	}
	return out, nil
}

// ViewInfoFor summarises a view, validating its template.
func ViewInfoFor(v *v1alpha1.ConfigurationView) *deploygrid.ViewInfo {
	info := &deploygrid.ViewInfo{
		Name:          v.Name,
		DisplayName:   v.Spec.DisplayName,
		Configuration: viewConfiguration(v),
		ContentType:   viewContentType(v),
		Valid:         true,
	}
	if _, err := configuration.Parse(v); err != nil {
		info.Valid = false
		info.Error = err.Error()
	}
	return info
}

func viewConfiguration(v *v1alpha1.ConfigurationView) string {
	if v.Spec.Configuration != "" {
		return v.Spec.Configuration
	}
	return v.Name
}

func viewContentType(v *v1alpha1.ConfigurationView) string {
	if v.Spec.ContentType != "" {
		return v.Spec.ContentType
	}
	return "text/plain"
}

func (s *configurationService) Views(_ context.Context, system string) ([]*deploygrid.ViewInfo, error) {
	views, err := s.viewsFor(system)
	if err != nil {
		return nil, err
	}
	out := make([]*deploygrid.ViewInfo, 0, len(views))
	for _, v := range views {
		out = append(out, ViewInfoFor(v))
	}
	return out, nil
}

func (s *configurationService) findView(system, viewName string) (*v1alpha1.ConfigurationView, error) {
	views, err := s.viewsFor(system)
	if err != nil {
		return nil, err
	}
	for _, v := range views {
		if v.Name == viewName {
			return v, nil
		}
	}
	return nil, fmt.Errorf("view %q in system %q: %w", viewName, system, apiinframux.ErrNotFound)
}

// mergedValues returns the merged configuration a view renders, or nil when
// no document exists (a view may render from the grid alone).
func (s *configurationService) mergedValues(system string, view *v1alpha1.ConfigurationView, environment string) (any, error) {
	systemWide, envDoc, err := s.documents(system, viewConfiguration(view), environment)
	if err != nil {
		return nil, nil //nolint:nilerr // absence of a document is not an error for rendering
	}
	return configuration.Merge(systemWide, envDoc)
}

func (s *configurationService) Render(ctx context.Context, system, viewName, environment string) ([]byte, string, error) {
	view, err := s.findView(system, viewName)
	if err != nil {
		return nil, "", err
	}
	g, err := s.grids.Grid(ctx, system)
	if err != nil {
		return nil, "", err
	}
	if environment != "" && !slices.ContainsFunc(g.Environments, func(e *deploygrid.Environment) bool { return e.Name == environment }) {
		return nil, "", fmt.Errorf("environment %q in system %q: %w", environment, system, apiinframux.ErrNotFound)
	}
	vals, err := s.mergedValues(system, view, environment)
	if err != nil {
		return nil, "", err
	}
	out, err := configuration.Render(view, configuration.Context{
		System:      g.System,
		Environment: environment,
		Values:      vals,
		Components:  configuration.ComponentsFor(g, environment),
		Grid:        g,
	})
	if err != nil {
		return nil, "", fmt.Errorf("%w: %w", apiinframux.ErrBadRequest, err)
	}
	return out, viewContentType(view), nil
}

func NewConfigurationService(catalog Catalog, grids GridService) ConfigurationService {
	return &configurationService{catalog: catalog, grids: grids}
}
