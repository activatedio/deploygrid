package history_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/activatedio/deploygrid/pkg/deploygrid"
	"github.com/activatedio/deploygrid/pkg/history"
)

func grid(cells map[string]map[string]string) *deploygrid.Grid {
	g := &deploygrid.Grid{System: &deploygrid.System{Name: "apps"}}
	grp := &deploygrid.GridGroup{Name: "g"}
	for comp, envs := range cells {
		row := &deploygrid.GridRow{Component: &deploygrid.ComponentRef{Name: comp}, Cells: map[string]*deploygrid.Cell{}}
		for env, v := range envs {
			row.Cells[env] = &deploygrid.Cell{Version: v, Cluster: "c1", DesiredVersion: v}
		}
		grp.Rows = append(grp.Rows, row)
	}
	g.Groups = []*deploygrid.GridGroup{grp}
	return g
}

func TestRing(t *testing.T) {
	r := history.New(3)
	t0 := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

	// seeding records versions silently, even for cells seen before
	r.Seed(grid(map[string]map[string]string{"api": {"dev": "0.5"}}))
	r.Seed(grid(map[string]map[string]string{"api": {"dev": "1.0"}}))
	assert.Empty(t, r.List(history.Query{System: "apps"}))

	// first sight of a cell seeds, no changes
	assert.Empty(t, r.Observe(grid(map[string]map[string]string{"api": {"dev": "1.0", "qa": "0.9"}}), t0))

	ch := r.Observe(grid(map[string]map[string]string{"api": {"dev": "1.1", "qa": "0.9"}}), t0.Add(time.Minute))
	require.Len(t, ch, 1)
	assert.Equal(t, "1.0", ch[0].From)
	assert.Equal(t, "1.1", ch[0].To)
	assert.Equal(t, "dev", ch[0].Environment)
	assert.Equal(t, "changed", ch[0].Kind())

	// qa cell disappears, a new component appears
	ch = r.Observe(grid(map[string]map[string]string{"api": {"dev": "1.1"}, "web": {"dev": "2.0"}}), t0.Add(2*time.Minute))
	require.Len(t, ch, 1, "web is seeded silently, qa disappearance is recorded")
	assert.Equal(t, "disappeared", ch[0].Kind())
	assert.Equal(t, "qa", ch[0].Environment)

	// qa comes back
	ch = r.Observe(grid(map[string]map[string]string{"api": {"dev": "1.1", "qa": "1.1"}, "web": {"dev": "2.0"}}), t0.Add(3*time.Minute))
	require.Len(t, ch, 1)
	assert.Equal(t, "appeared", ch[0].Kind())

	// ring capacity
	for i := 0; i < 5; i++ {
		r.Observe(grid(map[string]map[string]string{"api": {"dev": "1." + string(rune('2'+i)), "qa": "1.1"}, "web": {"dev": "2.0"}}), t0.Add(time.Duration(4+i)*time.Minute))
	}
	dev := r.List(history.Query{System: "apps", Component: "api", Environment: "dev"})
	require.Len(t, dev, 3, "bounded per cell")
	assert.Equal(t, "1.6", dev[0].To, "newest first")

	all := r.List(history.Query{System: "apps", Since: t0.Add(2 * time.Minute)})
	assert.NotEmpty(t, all)
	for _, c := range all {
		assert.False(t, c.ObservedAt.Before(t0.Add(2*time.Minute)))
	}
	assert.Empty(t, r.List(history.Query{System: "other"}))
	assert.Len(t, r.List(history.Query{System: "apps", Limit: 2}), 2)
}
