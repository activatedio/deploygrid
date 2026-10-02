package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/activatedio/deploygrid/pkg/apis/deploygrid.activated.io/v1alpha1"
	"github.com/activatedio/deploygrid/pkg/grid"
)

func TestMergeClusters(t *testing.T) {
	configured := []grid.ClusterInfo{
		{Name: "a", Addresses: []string{"https://a:6443"}},
		{Name: "b", Addresses: []string{"https://b:6443"}},
	}
	crs := []*v1alpha1.Cluster{
		{ObjectMeta: metav1.ObjectMeta{Name: "a"}, Spec: v1alpha1.ClusterSpec{Environment: "dev", Addresses: []string{"https://a.internal"}}},
		{ObjectMeta: metav1.ObjectMeta{Name: "c"}, Spec: v1alpha1.ClusterSpec{Environment: "prod"}},
	}

	out := mergeClusters(configured, crs)
	require.Len(t, out, 3)
	a := assert.New(t)
	a.Equal("dev", out[0].Environment, "CR settings overlay the configured cluster")
	a.ElementsMatch([]string{"https://a.internal", "https://a:6443"}, out[0].Addresses, "configured address is kept")
	a.Equal("b", out[1].Name)
	a.Empty(out[1].Environment)
	a.Equal("c", out[2].Name)
	a.Equal("prod", out[2].Environment)
}
