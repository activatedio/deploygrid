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

type Component struct {
	Name          string                 `json:"name"`
	ComponentType string                 `json:"component_type"`
	Children      []*Component           `json:"children"`
	Deployments   map[string]*Deployment `json:"deployments"`
}

type Deployment struct {
	Version string `json:"version"`
}

type Grid struct {
	Errors       []string       `json:"errors"`
	Environments []*Environment `json:"environments"`
	Components   []*Component   `json:"components"`
}
