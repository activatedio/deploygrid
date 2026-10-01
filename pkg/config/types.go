package config

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
	// Clusters is the v1 pull-mode cluster list. It is superseded by Cluster
	// custom resources and will be removed once the collector lands.
	Clusters ClustersConfig `description:"Observed clusters (v1 pull mode)"`
}

type LoggingConfig struct {
	Level   string `description:"zerolog level: trace, debug, info, warn, error"`
	DevMode bool   `description:"Pretty console output at debug level"`
}

type ServerConfig struct {
	Host string `description:"Listen address"`
	Port int    `description:"Listen port"`
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
	Name                  string `description:"Cluster name"`
	Address               string `description:"API server URL as referenced by Argo CD destinations"`
	KubeConfigPath        string `description:"Kubeconfig path"`
	ContextName           string `description:"Kubeconfig context"`
	Local                 bool   `description:"Use the in-cluster service account"`
	InsecureSkipTLSVerify bool   `description:"Skip TLS verification (development only)"`
}

type ClustersConfig struct {
	Clusters     []ClusterConfig `description:"Clusters to observe"`
	Environments []string        `description:"Environment names, in column order"`
}
