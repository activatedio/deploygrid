// Package grid builds a System's grid from declared resources (System,
// Component, Cluster) and observed resources (per-cluster store snapshots).
// It is pure: no Kubernetes clients, so it can be tested with fixtures.
package grid

// Labels and annotations understood on observed resources. Identity keys are
// labels so collectors can select on them; display keys are annotations.
const (
	LabelPrefix          = "deploygrid.activated.io/"
	LabelSystem          = LabelPrefix + "system"
	LabelComponent       = LabelPrefix + "component"
	LabelEnvironment     = LabelPrefix + "environment"
	AnnotationGroup      = LabelPrefix + "group"
	AnnotationDisplayNam = LabelPrefix + "display-name"

	// Legacy v1 annotations, read for one release and reported as a warning.
	LegacyAnnotationName        = "deploygrid/name"
	LegacyAnnotationEnvironment = "deploygrid/environment"
	LegacyAnnotationGroup       = "deploygrid/group"

	// GroupDefault holds rows that declare no group.
	GroupDefault = "Default"
)

// SystemNamespaces are never reported as unassigned; resources in them are
// only considered when they carry deploygrid labels.
var SystemNamespaces = map[string]bool{
	"kube-system":     true,
	"kube-public":     true,
	"kube-node-lease": true,
}
