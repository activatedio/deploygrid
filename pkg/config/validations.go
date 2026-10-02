package config

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

func validate(in any, rules ...*validation.FieldRules) error {
	return validation.ValidateStruct(in, rules...)
}

// DoValidate validates the whole tree; cs only invokes the hook on the root
// object that was read.
func (m *Main) DoValidate() error {
	return validate(m,
		validation.Field(&m.Logging),
		validation.Field(&m.Server),
		validation.Field(&m.Swagger),
		validation.Field(&m.Control),
		validation.Field(&m.Clusters),
		validation.Field(&m.Collector),
		validation.Field(&m.Sources),
	)
}

func (s SourcesConfig) Validate() error {
	return validate(&s, validation.Field(&s.ApplicationKinds))
}

func (a ApplicationKindConfig) Validate() error {
	return validate(&a,
		validation.Field(&a.Group, validation.Required),
		validation.Field(&a.Version, validation.Required),
		validation.Field(&a.Resource, validation.Required),
		validation.Field(&a.Kind, validation.Required),
	)
}

func (c CollectorConfig) Validate() error {
	return validate(&c,
		validation.Field(&c.Server, is.RequestURL),
		validation.Field(&c.FlushSeconds, validation.Min(1)),
		validation.Field(&c.HeartbeatSeconds, validation.Min(1)),
	)
}

// ValidateForRun checks the fields the collector command needs at start.
func (c CollectorConfig) ValidateForRun() error {
	return validate(&c,
		validation.Field(&c.Server, validation.Required, is.RequestURL),
		validation.Field(&c.Cluster, validation.Required),
		validation.Field(&c.Token, validation.When(c.TokenFile == "", validation.Required.Error("token or tokenFile is required"))),
	)
}

func (l LoggingConfig) Validate() error {
	return validate(&l,
		validation.Field(&l.Level, validation.In("trace", "debug", "info", "warn", "error", "fatal", "panic", "")),
	)
}

func (s ServerConfig) Validate() error {
	return validate(&s,
		validation.Field(&s.Port, validation.Required, validation.Min(1), validation.Max(65535)),
	)
}

func (s SwaggerConfig) Validate() error {
	return validate(&s,
		validation.Field(&s.SwaggerUiUrl, is.URL),
	)
}

func (c ControlConfig) Validate() error {
	return validate(&c,
		validation.Field(&c.Namespace, validation.When(c.Enabled, validation.Required)),
	)
}

func (c ClustersConfig) Validate() error {
	return validate(&c,
		validation.Field(&c.Clusters),
	)
}

func (c ClusterConfig) Validate() error {
	mode := c.EffectiveMode()
	return validate(&c,
		validation.Field(&c.Name, validation.Required),
		validation.Field(&c.Mode, validation.In(ClusterModeKubeconfig, ClusterModeLocal, ClusterModeAgent, "")),
		validation.Field(&c.KubeConfigPath, validation.When(mode == ClusterModeKubeconfig, validation.Required)),
		validation.Field(&c.Address, validation.When(mode == ClusterModeKubeconfig, validation.Required), is.URL),
		validation.Field(&c.Token, validation.When(mode != ClusterModeAgent, validation.Empty.Error("only agent clusters take a token"))),
	)
}
