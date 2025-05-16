package config

import validation "github.com/go-ozzo/ozzo-validation/v4"

func validate(in any, rules ...*validation.FieldRules) error {
	return validation.ValidateStruct(in, rules...)
}

func (l *LoggingConfig) DoValidate() error {
	return validate(l)
}

func (c *ClustersConfig) DoValidate() error {
	return validate(c)
}

func (s *SwaggerConfig) DoValidate() error {
	return validate(s)
}

func (s *ServerConfig) DoValidate() error {
	return validate(s)
}
