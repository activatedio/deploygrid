package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	apiinfraconfig "github.com/activatedio/deploygrid/pkg/apiinfra/config"
	"github.com/activatedio/deploygrid/pkg/config"
)

func TestCollectorConfig_EnvOverrides(t *testing.T) {
	t.Setenv("DEPLOYGRID_COLLECTOR_SERVER", "https://ingest.example/api")
	t.Setenv("DEPLOYGRID_COLLECTOR_CLUSTER", "c1")
	t.Setenv("DEPLOYGRID_COLLECTOR_TOKEN_FILE", "/etc/deploygrid/token/token")
	t.Setenv("DEPLOYGRID_COLLECTOR_FLUSH_SECONDS", "7")

	m := config.NewMainConfig(apiinfraconfig.NewConfig(""))
	a := assert.New(t)
	a.Equal("https://ingest.example/api", m.Collector.Server)
	a.Equal("c1", m.Collector.Cluster)
	a.Equal("/etc/deploygrid/token/token", m.Collector.TokenFile)
	a.Equal(7, m.Collector.FlushSeconds)
}
