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
	Mode                  string `description:"kubeconfig | local | agent"`
	KubeConfigPath        string `description:"Kubeconfig path (mode kubeconfig)"`
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
