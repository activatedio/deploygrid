package config_test

import (
	"github.com/activatedio/deploygrid/pkg/apiinfra/config"
	"github.com/go-errors/errors"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
)

type Dummy struct {
	Value1 string
	Value2 string
}

func (d *Dummy) DoValidate() error {
	if d.Value1 == "" {
		return errors.New("value1 is empty")
	}
	return nil
}

func TestUpperUnderscore(t *testing.T) {

	type s struct {
		input    string
		expected string
	}

	cases := map[string]s{
		"simple": {
			input:    "simple",
			expected: "SIMPLE",
		},
		"full": {
			input:    "testValue",
			expected: "TEST_VALUE",
		},
		"full two": {
			input:    "testSomethingElse",
			expected: "TEST_SOMETHING_ELSE",
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			assert.Equal(t, v.expected, config.UpperUnderscore(v.input))
		})
	}
}

func TestMustUnmarshallAndValidate(t *testing.T) {

	a := assert.New(t)

	type s struct {
		arrange func() (*viper.Viper, string, *Dummy)
		assert  func(got *Dummy, panicValue any)
	}

	cases := map[string]s{
		"invalid": {
			arrange: func() (*viper.Viper, string, *Dummy) {
				v := viper.New()
				return v, "prefix", &Dummy{}
			},
			assert: func(got *Dummy, panicValue any) {
				a.Nil(got)
				a.EqualError(panicValue.(error), "type: Dummy key: prefix detail: value1 is empty")
			},
		},
		"no prefix": {
			arrange: func() (*viper.Viper, string, *Dummy) {
				v := viper.New()
				v.Set("value1", "a")
				v.Set("value2", "b")
				return v, "", &Dummy{}
			},
			assert: func(got *Dummy, panicValue any) {
				a.Equal(&Dummy{
					Value1: "a",
					Value2: "b",
				}, got)
				a.Nil(panicValue)
			},
		},
		"with prefix": {
			arrange: func() (*viper.Viper, string, *Dummy) {
				v := viper.New()
				v.Set("prefix.value1", "a")
				v.Set("prefix.value2", "b")
				return v, "prefix", &Dummy{}
			},
			assert: func(got *Dummy, panicValue any) {
				a.Equal(&Dummy{
					Value1: "a",
					Value2: "b",
				}, got)
				a.Nil(panicValue)
			},
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {

			defer func() {
				if r := recover(); r != nil {
					v.assert(nil, r)
				}
			}()

			v.assert(config.MustUnmarshallAndValidate(v.arrange()), nil)
		})
	}
}
