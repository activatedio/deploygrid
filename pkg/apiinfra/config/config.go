package config

import (
	"github.com/activatedio/cs"
	"github.com/activatedio/cs/sources"
	"github.com/activatedio/cs/sources/yaml"
)

const envPrefix = "DEPLOYGRID"

// Validating is implemented by configuration roots that check themselves after
// being read.
type Validating interface {
	DoValidate() error
}

// NewConfig builds a cs.Config that reads the optional YAML file at configPath
// and lets DEPLOYGRID_* environment variables override any key.
func NewConfig(configPath string) cs.Config {

	c := cs.New()

	if configPath != "" {
		c.AddSource(yaml.FromPath(configPath, ""))
	}
	c.AddLateBindingSource(sources.FromEnvironment(envPrefix))

	c.SetValidatingHook(func(in any) error {
		switch v := in.(type) {
		case Validating:
			return v.DoValidate()
		case cs.Validating:
			return v.Validate()
		}
		return nil
	})

	return c
}
