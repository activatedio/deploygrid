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
	return validate(&c,
		validation.Field(&c.Name, validation.Required),
		validation.Field(&c.KubeConfigPath, validation.When(!c.Local, validation.Required)),
		validation.Field(&c.Address, validation.When(!c.Local, validation.Required), is.URL),
	)
}
