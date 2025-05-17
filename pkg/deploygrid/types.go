package deploygrid

type Environment struct {
	Name string `json:"name"`
}

type Component struct {
	Name          string                `json:"name"`
	ComponentType string                `json:"component_type"`
	Children      []Component           `json:"children"`
	Deployments   map[string]Deployment `json:"deployments"`
}

type Deployment struct {
	Version string `json:"version"`
}

type Grid struct {
	Errors       []string      `json:"errors"`
	Environments []Environment `json:"environments"`
	Components   []Component   `json:"components"`
}
