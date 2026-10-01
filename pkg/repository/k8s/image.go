package k8s

import "strings"

// ImageReference is a parsed container image reference such as
// registry.example.com:5000/team/app:1.2.3@sha256:abcd...
type ImageReference struct {
	Repository string
	Tag        string
	Digest     string
}

// ParseImageReference splits an image reference into repository, tag and
// digest. It tolerates registry host ports, which a naive split on ':' does
// not.
func ParseImageReference(image string) ImageReference {
	ref := ImageReference{}

	rest := image
	if at := strings.Index(rest, "@"); at >= 0 {
		ref.Digest = rest[at+1:]
		rest = rest[:at]
	}

	// A ':' after the last '/' separates the tag; one before it is a port.
	lastSlash := strings.LastIndex(rest, "/")
	if colon := strings.LastIndex(rest, ":"); colon > lastSlash {
		ref.Tag = rest[colon+1:]
		rest = rest[:colon]
	}

	ref.Repository = rest
	return ref
}

// Version is the value displayed in a grid cell: the tag when present,
// otherwise a shortened digest, otherwise "latest".
func (r ImageReference) Version() string {
	if r.Tag != "" {
		return r.Tag
	}
	if r.Digest != "" {
		hex := r.Digest
		if i := strings.Index(hex, ":"); i >= 0 {
			hex = hex[i+1:]
		}
		if len(hex) > 12 {
			hex = hex[:12]
		}
		return hex
	}
	return "latest"
}
