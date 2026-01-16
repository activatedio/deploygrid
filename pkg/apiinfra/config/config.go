package config

import (
	"github.com/activatedio/cs"
	"github.com/activatedio/cs/sources"
	"github.com/activatedio/cs/sources/yaml"
)

type Validating interface {
	DoValidate() error
}

func NewConfig(configPath string) cs.Config {

	c := cs.New()

	c.AddSource(yaml.FromPath(configPath, ""))
	c.AddLateBindingSource(sources.FromEnvironment("DEPLOYGRID"))

	c.SetValidatingHook(func(in any) error {
		if v, ok := in.(Validating); ok {
			return v.DoValidate()
		}
		return nil
	})

	return c
}
