package configuration_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/activatedio/deploygrid/pkg/apis/deploygrid.activated.io/v1alpha1"
	"github.com/activatedio/deploygrid/pkg/configuration"
	"github.com/activatedio/deploygrid/pkg/deploygrid"
)

func cfg(env, raw string) *v1alpha1.Configuration {
	return &v1alpha1.Configuration{Spec: v1alpha1.ConfigurationSpec{
		System: "apps", Environment: env,
		Values: &apiextensionsv1.JSON{Raw: []byte(raw)},
	}}
}

func TestMerge(t *testing.T) {
	base := cfg("", `{"domain":"example.com","db":{"host":"db","port":5432,"pool":{"min":1,"max":10}},"features":["a","b"]}`)
	over := cfg("qa", `{"domain":"qa.example.com","db":{"host":"qa-db","pool":{"max":3}},"features":["c"],"extra":true}`)

	out, err := configuration.Merge(base, over)
	require.NoError(t, err)
	m := out.(map[string]any)
	a := assert.New(t)
	a.Equal("qa.example.com", m["domain"])
	db := m["db"].(map[string]any)
	a.Equal("qa-db", db["host"])
	a.InDelta(5432, db["port"], 0, "untouched keys survive")
	pool := db["pool"].(map[string]any)
	a.InDelta(1, pool["min"], 0)
	a.InDelta(3, pool["max"], 0)
	a.Equal([]any{"c"}, m["features"], "lists replace, they do not append")
	a.Equal(true, m["extra"])

	only, err := configuration.Merge(base, nil)
	require.NoError(t, err)
	a.Equal("example.com", only.(map[string]any)["domain"])

	none, err := configuration.Merge(nil, nil)
	require.NoError(t, err)
	a.Nil(none)

	_, err = configuration.Merge(cfg("", `{bad`), nil)
	require.Error(t, err)
}

func TestRender(t *testing.T) {
	g := &deploygrid.Grid{
		System: &deploygrid.System{Name: "apps", DisplayName: "Apps"},
		Groups: []*deploygrid.GridGroup{{Name: "core", Rows: []*deploygrid.GridRow{
			{Component: &deploygrid.ComponentRef{Name: "api", DisplayName: "API"}, Cells: map[string]*deploygrid.Cell{
				"qa": {Version: "1.2.0", DesiredVersion: "1.3.0", Drifted: true, Hosts: []string{"api.qa.example.com"}},
			}, Children: []*deploygrid.GridRow{
				{Component: &deploygrid.ComponentRef{Name: "api-db"}, Cells: map[string]*deploygrid.Cell{"qa": {Version: "15"}}},
			}},
			{Component: &deploygrid.ComponentRef{Name: "web"}, Cells: map[string]*deploygrid.Cell{}},
		}}},
	}
	comps := configuration.ComponentsFor(g, "qa")
	require.Len(t, comps, 3)
	assert.Equal(t, "core", comps[1].Group)
	assert.Equal(t, "15", comps[1].Version)
	assert.Empty(t, comps[2].Version, "no cell in this environment")

	view := &v1alpha1.ConfigurationView{
		ObjectMeta: metav1.ObjectMeta{Name: "hosts"},
		Spec: v1alpha1.ConfigurationViewSpec{Template: `# {{ .System.DisplayName }} / {{ .Environment | upper }}
domain: {{ .Values.domain | default "none" }}
{{- range .Components }}
{{ .Name }}: {{ .Version | default "-" }}{{ if .Drifted }} (wants {{ .DesiredVersion }}){{ end }}
{{- end }}
hosts: {{ join "," (index .Components 0).Hosts }}
values: {{ toJson .Values }}
`},
	}
	out, err := configuration.Render(view, configuration.Context{
		System: g.System, Environment: "qa", Grid: g, Components: comps,
		Values: map[string]any{"domain": "qa.example.com"},
	})
	require.NoError(t, err)
	assert.Equal(t, `# Apps / QA
domain: qa.example.com
api: 1.2.0 (wants 1.3.0)
api-db: 15
web: -
hosts: api.qa.example.com
values: {"domain":"qa.example.com"}
`, string(out))

	_, err = configuration.Parse(&v1alpha1.ConfigurationView{Spec: v1alpha1.ConfigurationViewSpec{Template: "{{ .Broken "}})
	require.Error(t, err)
}
