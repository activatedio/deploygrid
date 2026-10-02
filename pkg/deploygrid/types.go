// Package deploygrid holds the API model served over HTTP.
package deploygrid

// Environment is one column of a grid.
type Environment struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`
}

// Group is a row section of a grid.
type Group struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`
}

// System is a product or platform boundary: one grid.
type System struct {
	Name         string         `json:"name"`
	DisplayName  string         `json:"display_name,omitempty"`
	Description  string         `json:"description,omitempty"`
	Environments []*Environment `json:"environments"`
	Groups       []*Group       `json:"groups,omitempty"`
}

type SystemList struct {
	Items []*System `json:"items"`
}

// ComponentRef identifies a row.
type ComponentRef struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`
	Description string `json:"description,omitempty"`
	Kind        string `json:"kind,omitempty"`
	// Discovered is true when no Component resource declares this row; it was
	// derived from labels on observed resources.
	Discovered bool `json:"discovered"`
}

// Version is one versioned part of an artifact.
type Version struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Value   string `json:"value"`
	Image   string `json:"image,omitempty"`
	Desired string `json:"desired,omitempty"`
}

// Artifact is an observed resource that contributed to a cell.
type Artifact struct {
	Kind      string     `json:"kind"`
	Cluster   string     `json:"cluster"`
	Namespace string     `json:"namespace,omitempty"`
	Name      string     `json:"name"`
	Health    string     `json:"health,omitempty"`
	Versions  []*Version `json:"versions,omitempty"`
	// Environment is set on unassigned artifacts when it could be resolved.
	Environment string `json:"environment,omitempty"`
}

// Link is a rendered component link for one cell.
type Link struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Cell is the state of one component in one environment.
type Cell struct {
	Version        string      `json:"version"`
	DesiredVersion string      `json:"desired_version,omitempty"`
	Drifted        bool        `json:"drifted"`
	Inconsistent   bool        `json:"inconsistent"`
	Health         string      `json:"health"`
	Cluster        string      `json:"cluster,omitempty"`
	Namespace      string      `json:"namespace,omitempty"`
	Hosts          []string    `json:"hosts,omitempty"`
	Links          []*Link     `json:"links,omitempty"`
	Artifacts      []*Artifact `json:"artifacts,omitempty"`
}

// GridRow is one component with its cells, keyed by environment name.
type GridRow struct {
	Component *ComponentRef    `json:"component"`
	Cells     map[string]*Cell `json:"cells"`
	Children  []*GridRow       `json:"children,omitempty"`
}

// GridGroup is an ordered section of rows.
type GridGroup struct {
	Name        string     `json:"name"`
	DisplayName string     `json:"display_name,omitempty"`
	Rows        []*GridRow `json:"rows"`
}

// Grid is the full view of one System.
type Grid struct {
	System       *System        `json:"system"`
	Environments []*Environment `json:"environments"`
	Groups       []*GridGroup   `json:"groups"`
	Warnings     []string       `json:"warnings,omitempty"`
	Errors       []string       `json:"errors,omitempty"`
}

type ArtifactList struct {
	Items []*Artifact `json:"items"`
}

type GridRowList struct {
	Items []*GridRow `json:"items"`
}
