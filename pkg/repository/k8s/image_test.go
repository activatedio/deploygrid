package k8s_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/activatedio/deploygrid/pkg/repository/k8s"
)

func TestParseImageReference(t *testing.T) {
	cases := map[string]struct {
		in      string
		want    k8s.ImageReference
		version string
	}{
		"bare":            {"nginx", k8s.ImageReference{Repository: "nginx"}, "latest"},
		"tag":             {"nginx:1.14.2", k8s.ImageReference{Repository: "nginx", Tag: "1.14.2"}, "1.14.2"},
		"registry port":   {"registry:5000/team/app", k8s.ImageReference{Repository: "registry:5000/team/app"}, "latest"},
		"port and tag":    {"registry:5000/team/app:v2", k8s.ImageReference{Repository: "registry:5000/team/app", Tag: "v2"}, "v2"},
		"digest only":     {"ghcr.io/a/b@sha256:0123456789abcdef0123", k8s.ImageReference{Repository: "ghcr.io/a/b", Digest: "sha256:0123456789abcdef0123"}, "0123456789ab"},
		"tag and digest":  {"ghcr.io/a/b:1.0@sha256:0123456789abcdef0123", k8s.ImageReference{Repository: "ghcr.io/a/b", Tag: "1.0", Digest: "sha256:0123456789abcdef0123"}, "1.0"},
		"port no tag dot": {"localhost:5000/app", k8s.ImageReference{Repository: "localhost:5000/app"}, "latest"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got := k8s.ParseImageReference(c.in)
			assert.Equal(t, c.want, got)
			assert.Equal(t, c.version, got.Version())
		})
	}
}
