package hack

import (
	"github.com/activatedio/deploygrid/api/v1alpha1"
	"github.com/rancher/wrangler/v3/pkg/crd"
)

func ListCRDs() []crd.CRD {

	system := crd.NamespacedType("System.deploygrid.activated.io/v1alpha1").
		WithSchemaFromStruct(v1alpha1.System{})

	metadata := crd.NamespacedType("Metadata.deploygrid.activated.io/v1alpha1").
		WithSchemaFromStruct(v1alpha1.Metadata{})

	metadataView := crd.NamespacedType("Metadata.deploygrid.activated.io/v1alpha1").
		WithSchemaFromStruct(v1alpha1.MetadataView{})

	return []crd.CRD{system, metadata, metadataView}
}
