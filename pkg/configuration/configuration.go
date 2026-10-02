// Package configuration merges Configuration documents and renders
// ConfigurationView templates.
package configuration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"

	"github.com/activatedio/deploygrid/pkg/apis/deploygrid.activated.io/v1alpha1"
	"github.com/activatedio/deploygrid/pkg/deploygrid"
)

// Merge deep-merges the system-wide document and the environment document of
// one Configuration name. Maps merge recursively; any other value in the
// environment document replaces the system-wide one. Either may be nil.
func Merge(systemWide, environment *v1alpha1.Configuration) (any, error) {
	base, err := values(systemWide)
	if err != nil {
		return nil, err
	}
	over, err := values(environment)
	if err != nil {
		return nil, err
	}
	return merge(base, over), nil
}

func values(c *v1alpha1.Configuration) (any, error) {
	if c == nil || c.Spec.Values == nil || len(c.Spec.Values.Raw) == 0 {
		return nil, nil
	}
	return decode(c.Spec.Values)
}

func decode(j *apiextensionsv1.JSON) (any, error) {
	var out any
	if err := json.Unmarshal(j.Raw, &out); err != nil {
		return nil, fmt.Errorf("decode configuration values: %w", err)
	}
	return out, nil
}

func merge(base, over any) any {
	if over == nil {
		return base
	}
	bm, bok := base.(map[string]any)
	om, ook := over.(map[string]any)
	if !bok || !ook {
		return over
	}
	out := make(map[string]any, len(bm)+len(om))
	for k, v := range bm {
		out[k] = v
	}
	for k, v := range om {
		if existing, ok := out[k]; ok {
			out[k] = merge(existing, v)
		} else {
			out[k] = v
		}
	}
	return out
}

// ComponentView is one grid row flattened for one environment.
type ComponentView struct {
	Name           string
	DisplayName    string
	Kind           string
	Group          string
	Version        string
	DesiredVersion string
	Drifted        bool
	Health         string
	Cluster        string
	Namespace      string
	Hosts          []string
	Links          []*deploygrid.Link
}

// Context is what templates see.
type Context struct {
	System      *deploygrid.System
	Environment string
	Values      any
	Components  []ComponentView
	Grid        *deploygrid.Grid
}

// ComponentsFor flattens the grid rows of one environment.
func ComponentsFor(g *deploygrid.Grid, environment string) []ComponentView {
	var out []ComponentView
	var walk func(group string, rows []*deploygrid.GridRow)
	walk = func(group string, rows []*deploygrid.GridRow) {
		for _, r := range rows {
			cv := ComponentView{
				Name:        r.Component.Name,
				DisplayName: r.Component.DisplayName,
				Kind:        r.Component.Kind,
				Group:       group,
			}
			if cell, ok := r.Cells[environment]; ok {
				cv.Version = cell.Version
				cv.DesiredVersion = cell.DesiredVersion
				cv.Drifted = cell.Drifted
				cv.Health = cell.Health
				cv.Cluster = cell.Cluster
				cv.Namespace = cell.Namespace
				cv.Hosts = cell.Hosts
				cv.Links = cell.Links
			}
			out = append(out, cv)
			walk(group, r.Children)
		}
	}
	for _, grp := range g.Groups {
		walk(grp.Name, grp.Rows)
	}
	return out
}

// Funcs are the template functions available to views.
var Funcs = template.FuncMap{
	"default": func(def, val any) any {
		if val == nil || val == "" || val == false || val == 0 {
			return def
		}
		return val
	},
	"upper":   strings.ToUpper,
	"lower":   strings.ToLower,
	"title":   func(s string) string { return strings.ToUpper(s[:min(1, len(s))]) + s[min(1, len(s)):] },
	"trim":    strings.TrimSpace,
	"replace": func(old, newS, s string) string { return strings.ReplaceAll(s, old, newS) },
	"join":    func(sep string, items []string) string { return strings.Join(items, sep) },
	"split":   func(sep, s string) []string { return strings.Split(s, sep) },
	"quote":   func(s string) string { return fmt.Sprintf("%q", s) },
	"indent": func(n int, s string) string {
		pad := strings.Repeat(" ", n)
		return pad + strings.ReplaceAll(s, "\n", "\n"+pad)
	},
	"nindent": func(n int, s string) string {
		pad := strings.Repeat(" ", n)
		return "\n" + pad + strings.ReplaceAll(s, "\n", "\n"+pad)
	},
	"toYaml": func(v any) (string, error) {
		bs, err := yaml.Marshal(v)
		return strings.TrimSuffix(string(bs), "\n"), err
	},
	"toJson": func(v any) (string, error) {
		bs, err := json.Marshal(v)
		return string(bs), err
	},
}

// Parse compiles a view template, reporting syntax errors.
func Parse(view *v1alpha1.ConfigurationView) (*template.Template, error) {
	return template.New(view.Name).Funcs(Funcs).Option("missingkey=zero").Parse(view.Spec.Template)
}

// Render executes a view for the given context.
func Render(view *v1alpha1.ConfigurationView, ctx Context) ([]byte, error) {
	t, err := Parse(view)
	if err != nil {
		return nil, fmt.Errorf("view %s: %w", view.Name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, ctx); err != nil {
		return nil, fmt.Errorf("view %s: %w", view.Name, err)
	}
	return buf.Bytes(), nil
}
