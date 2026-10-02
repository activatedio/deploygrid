package config

import "strings"

// Main is the root of the deploygrid configuration. Keys are lower camel case
// of the field names (for example server.port) and every key can be overridden
// by an environment variable of the form DEPLOYGRID_SERVER_PORT.
type Main struct {
	Logging LoggingConfig `description:"Logging settings"`
	Server  ServerConfig  `description:"HTTP server settings"`
	Swagger SwaggerConfig `description:"Swagger UI settings"`
	// Control is the connection to the cluster that stores deploygrid custom
	// resources (System, Component, Cluster, ...).
	Control ControlConfig `description:"Control cluster settings"`
	// Clusters are the observed clusters: how the server reaches them (or
	// which token their collector presents). Cluster custom resources add
	// environment mapping on top.
	Clusters ClustersConfig `description:"Observed clusters"`
	// Collector is only used by the `deploygrid collector` command.
	Collector CollectorConfig `description:"Collector settings"`
	// Sources configures what is observed beyond the built-in Argo CD
	// Applications, Deployments and Ingresses. Used by server-side watches
	// and by collectors.
	Sources SourcesConfig `description:"Additional observed resources"`
}

// SourcesConfig lists extra observed resource kinds.
type SourcesConfig struct {
	// ApplicationKinds are custom resources that represent an application
	// installed by an operator, for example platform.example.com
	// AppSuite. Workloads that carry a controller ownerReference to such a
	// resource are grouped under it.
	ApplicationKinds []ApplicationKindConfig `description:"Operator application custom resources"`
}

// ApplicationKindConfig describes one operator application kind. Paths are
// Kubernetes JSONPath expressions evaluated against the custom resource,
// with or without the surrounding braces.
type ApplicationKindConfig struct {
	Group    string `description:"API group, e.g. platform.example.com"`
	Version  string `description:"API version, e.g. v1alpha1"`
	Resource string `description:"Plural resource, e.g. appsuites"`
	Kind     string `description:"Kind, as it appears in ownerReferences, e.g. AppSuite"`
	// Component is the grid row every instance of this kind maps to. When
	// empty, the app.kubernetes.io/name label or the resource name is used.
	Component string `description:"Component (row) for all instances; defaults to the app.kubernetes.io/name label or the resource name"`
	// DesiredVersionPath reads the version the resource asks for. Defaults
	// to {.spec.version}.
	DesiredVersionPath string `description:"JSONPath of the desired version (default {.spec.version})"`
	// RunningVersionPath reads what the operator reports as running, either
	// a version or one or more image references whose tags are used. When
	// empty, the owned workloads decide.
	RunningVersionPath string `description:"JSONPath of the running version or images (optional)"`
	// PinnedVersionsPath lists versions (or image references) the resource
	// intentionally pins for some of its parts, for example
	// {.spec.services[*].image.tag}. Workloads running a pinned version do
	// not count as inconsistent.
	PinnedVersionsPath string `description:"JSONPath of intentionally pinned versions or images (optional)"`
	// EnvironmentPath reads the environment from the resource; otherwise
	// labels, Cluster namespace rules and the cluster default apply.
	EnvironmentPath string `description:"JSONPath of the environment (optional)"`
	// HealthConditionType is the status condition that means healthy.
	// Defaults to Ready. A Degraded=True or <type>=False condition means
	// degraded; Progressing=True means progressing.
	HealthConditionType string `description:"Condition type meaning healthy (default Ready)"`
}

// Key names the store kind for resources of this application kind.
func (a ApplicationKindConfig) Key() string {
	return "application/" + a.Group + "/" + a.Resource
}

type LoggingConfig struct {
	Level   string `description:"zerolog level: trace, debug, info, warn, error"`
	DevMode bool   `description:"Pretty console output at debug level"`
}

type ServerConfig struct {
	Host string `description:"Listen address"`
	Port int    `description:"Listen port"`
	// CorsAllowedOrigins enables CORS for the listed origins (comma
	// separated). Empty disables CORS: in the chart the UI's nginx proxies
	// /api so the browser sees one origin. Development against the Vite
	// server needs http://localhost:5173.
	CorsAllowedOrigins string `description:"Comma-separated origins allowed by CORS; empty disables CORS"`
}

// CorsOrigins returns the configured origins as a list.
func (s ServerConfig) CorsOrigins() []string {
	parts := strings.Split(s.CorsAllowedOrigins, ",")
	out := make([]string, 0, len(parts))
	for _, o := range parts {
		if o = strings.TrimSpace(o); o != "" {
			out = append(out, o)
		}
	}
	return out
}

type SwaggerConfig struct {
	SwaggerUiUrl string `description:"Upstream swagger-ui to proxy under /api/swagger-ui; disabled when empty"`
}

type ControlConfig struct {
	Enabled bool `description:"Read deploygrid custom resources from the control cluster"`
	// Namespace holding the deploygrid custom resources.
	Namespace string `description:"Namespace that holds deploygrid custom resources"`
	// KubeConfigPath selects an out-of-cluster control cluster. When empty
	// the in-cluster service account is used.
	KubeConfigPath string `description:"Kubeconfig for the control cluster; in-cluster when empty"`
	Context        string `description:"Kubeconfig context to use"`
}

type ClusterConfig struct {
	Name    string `description:"Cluster name"`
	Address string `description:"API server URL as referenced by Argo CD destinations"`
	// Mode is kubeconfig (default: the server watches the cluster through
	// KubeConfigPath), local (in-cluster service account) or agent (a
	// collector pushes observations; nothing is watched server-side).
	Mode string `description:"kubeconfig | local | agent"`
	// Auth, for mode kubeconfig, selects a credential instead of a kubeconfig
	// file: "google" uses Application Default Credentials (GKE workload
	// identity, GOOGLE_APPLICATION_CREDENTIALS) as the bearer token against
	// Address, the way Argo CD reaches GKE clusters from another project.
	Auth           string `description:"Credential for mode kubeconfig without a kubeconfig file: google"`
	KubeConfigPath string `description:"Kubeconfig path (mode kubeconfig, unless auth is set)"`
	// CAFile verifies Address when it is not signed by a public CA (GKE DNS
	// endpoints are).
	CAFile                string `description:"CA bundle for Address (optional)"`
	ContextName           string `description:"Kubeconfig context"`
	Local                 bool   `description:"Deprecated: same as mode local"`
	InsecureSkipTLSVerify bool   `description:"Skip TLS verification (development only)"`
	// Token is the static bearer token a collector presents for this cluster
	// (mode agent). Cluster custom resources can supply tokens from Secrets
	// instead.
	Token string `description:"Collector bearer token (mode agent)"`
}

// EffectiveMode normalises Mode and the deprecated Local flag.
func (c ClusterConfig) EffectiveMode() string {
	switch {
	case c.Mode != "":
		return c.Mode
	case c.Local:
		return ClusterModeLocal
	default:
		return ClusterModeKubeconfig
	}
}

// CollectorConfig configures the `deploygrid collector` command.
type CollectorConfig struct {
	// Server is the base URL of the deploygrid API, for example
	// https://deploygrid.example.com/api.
	Server string `description:"deploygrid API base URL"`
	// Cluster is the name the observations are reported for; it must match
	// the token.
	Cluster string `description:"Cluster name"`
	Token   string `description:"Bearer token"`
	// TokenFile is read instead of Token when set (for mounted Secrets).
	TokenFile string `description:"File containing the bearer token"`
	// KubeConfigPath selects an out-of-cluster target; in-cluster when empty.
	KubeConfigPath string `description:"Kubeconfig for the observed cluster; in-cluster when empty"`
	Context        string `description:"Kubeconfig context"`
	// FlushSeconds is how long changes are coalesced before being pushed.
	FlushSeconds int `description:"Seconds to coalesce changes before pushing"`
	// HeartbeatSeconds is the interval of empty observations when nothing
	// changed.
	HeartbeatSeconds int `description:"Heartbeat interval in seconds"`
}

type ClustersConfig struct {
	Clusters     []ClusterConfig `description:"Clusters to observe"`
	Environments []string        `description:"Environment names, in column order"`
}
