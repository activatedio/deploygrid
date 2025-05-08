package config_test

import (
	config "github.com/activatedio/deploygrid/pkg/apiinfra/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"os"
	"strings"
	"testing"
)

type Config struct {
	ConfigPath string `mapstructure:"config_path"`
	DevMode    bool   `mapstructure:"dev_mode"`
}

func Test_E2E_EnvironmentVariables_WithPrefix(t *testing.T) {

	check(os.Setenv("PREFIX2_CONFIG_PATH", "config.yaml"))
	check(os.Setenv("PREFIX2_DEV_MODE", "true"))

	a := assert.New(t)

	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(`.`, `_`))
	v.AutomaticEnv()

	a.Equal("config.yaml", v.GetString("prefix2.config_path"))
	a.Equal(true, v.GetBool("prefix2.dev_mode"))

	v.SetDefault("prefix2", &Config{
		ConfigPath: "",
		DevMode:    false,
	})

	c := &Config{}

	a.Equal(&Config{
		ConfigPath: "config.yaml",
		DevMode:    true,
	}, config.MustUnmarshall(v, "prefix2", c))
	a.Equal(&Config{
		ConfigPath: "config.yaml",
		DevMode:    true,
	}, config.MustUnmarshall(v, "prefix2", c))

}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
