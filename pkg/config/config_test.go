package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiinfraconfig "github.com/activatedio/deploygrid/pkg/apiinfra/config"
	"github.com/activatedio/deploygrid/pkg/config"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(p, []byte(content), 0o600))
	return p
}

func TestNewMainConfig_DefaultsAndOverrides(t *testing.T) {

	p := writeConfig(t, `
logging:
  devMode: true
control:
  enabled: true
  kubeConfigPath: /tmp/kubeconfig
clusters:
  environments: [dev, qa]
  clusters:
    - name: c1
      address: https://127.0.0.1:6444
      kubeConfigPath: /tmp/c1
`)

	t.Setenv("DEPLOYGRID_SERVER_PORT", "9090")
	t.Setenv("DEPLOYGRID_CONTROL_NAMESPACE", "other")

	m := config.NewMainConfig(apiinfraconfig.NewConfig(p))

	a := assert.New(t)
	// defaults survive partial files
	a.Equal("127.0.0.1", m.Server.Host)
	a.Equal("info", m.Logging.Level)
	// file values
	a.True(m.Logging.DevMode)
	a.True(m.Control.Enabled)
	a.Equal("/tmp/kubeconfig", m.Control.KubeConfigPath)
	a.Equal([]string{"dev", "qa"}, m.Clusters.Environments)
	require.Len(t, m.Clusters.Clusters, 1)
	a.Equal("c1", m.Clusters.Clusters[0].Name)
	// environment overrides, including inside sections absent from the file
	a.Equal(9090, m.Server.Port)
	a.Equal("other", m.Control.Namespace)
}

func TestNewMainConfig_NoFile(t *testing.T) {
	m := config.NewMainConfig(apiinfraconfig.NewConfig(""))
	assert.Equal(t, 8080, m.Server.Port)
	assert.False(t, m.Control.Enabled)
}

func TestNewMainConfig_Validation(t *testing.T) {
	p := writeConfig(t, `
clusters:
  clusters:
    - name: broken
`)
	assert.Panics(t, func() {
		config.NewMainConfig(apiinfraconfig.NewConfig(p))
	})
}
