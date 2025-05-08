package config

import validation "github.com/go-ozzo/ozzo-validation/v4"

func validate(in any, rules ...*validation.FieldRules) error {
	return validation.ValidateStruct(in, rules...)
}

func (r *LoggingConfig) DoValidate() error {
	return validate(r)
}
