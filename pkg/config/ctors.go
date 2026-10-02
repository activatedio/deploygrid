package config

import (
	"go.uber.org/fx"

	"github.com/activatedio/cs"
	"github.com/activatedio/cs/sources"
)

// NewMainConfig reads the configuration from the given cs.Config, layering
// Defaults underneath whatever sources the config already has.
//
// Sections are read individually (rather than the root in one call) so that
// cs derives environment variable names such as DEPLOYGRID_SERVER_PORT from
// the full key path.
func NewMainConfig(c cs.Config) *Main {
	c.AddDefaultSource(sources.FromValue("", Defaults()))

	m := &Main{}
	c.MustRead("logging", &m.Logging)
	c.MustRead("server", &m.Server)
	c.MustRead("swagger", &m.Swagger)
	c.MustRead("control", &m.Control)
	c.MustRead("clusters", &m.Clusters)
	c.MustRead("sources", &m.Sources)

	if err := m.DoValidate(); err != nil {
		panic(err)
	}

	return m
}

// SubConfigs exposes the sections of Main as individual fx dependencies so
// that components only declare what they need.
type SubConfigs struct {
	fx.Out
	Logging  *LoggingConfig
	Server   *ServerConfig
	Swagger  *SwaggerConfig
	Control  *ControlConfig
	Clusters *ClustersConfig
	Sources  *SourcesConfig
}

func NewSubConfigs(m *Main) SubConfigs {
	return SubConfigs{
		Logging:  &m.Logging,
		Server:   &m.Server,
		Swagger:  &m.Swagger,
		Control:  &m.Control,
		Clusters: &m.Clusters,
		Sources:  &m.Sources,
	}
}
