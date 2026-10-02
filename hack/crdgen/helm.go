package main

import (
	"gopkg.in/yaml.v3"
)

var (
	NameMetadata    = "metadata"
	NameLabels      = "labels"
	NameAnnotations = "annotations"

	AnnotationNameHelmResourcePolicy = "helm.sh/resource-policy"
	LableNameAppShort                = "app"
	LableNameAppLong                 = "app.kubernetes.io/name"
	LableNameInstanceLong            = "app.kubernetes.io/instance"
)

type helmNodeVisitorContext struct {
	inMetadata bool
}

type innerMetadataVisitorContext struct {
	inLabels bool
}

type innerLabelsVisitorContext struct {
	commentWritten bool
}

var (
	innerLabelsVisitor = NewNodeVisitorFunc(func(node *yaml.Node, ctx any) (any, NodeVisitor, error) {

		if ctx == nil {
			ctx = &innerLabelsVisitorContext{}
		}

		tctx := ctx.(*innerLabelsVisitorContext)

		if node.Kind == yaml.ScalarNode && !tctx.commentWritten {
			tctx.commentWritten = true
			node.FootComment = `# Generated Labels {{- include "` + chartName + `.labels" . | nindent 4 }}`
		}

		return tctx, nil, nil

	})
	innerMetadataVisitor = NewNodeVisitorFunc(func(node *yaml.Node, ctx any) (any, NodeVisitor, error) {

		if ctx == nil {
			ctx = &innerMetadataVisitorContext{}
		}

		tctx := ctx.(*innerMetadataVisitorContext)

		if node.Kind == yaml.ScalarNode && node.Value == NameAnnotations {
			node.HeadComment = "# START ANNOTATIONS {{- if .Values.crds.keep }}"
			node.FootComment = "# END ANNOTATIONS {{- end }}"
		} else if node.Kind == yaml.ScalarNode && node.Value == NameLabels {
			tctx.inLabels = true
		} else if tctx.inLabels && node.Kind == yaml.MappingNode {
			return tctx, innerLabelsVisitor, nil
		}
		return tctx, nil, nil
	})
	helmNodeVisitor = NewNodeVisitorFunc(func(node *yaml.Node, ctx any) (any, NodeVisitor, error) {

		if ctx == nil {
			ctx = &helmNodeVisitorContext{}
		}

		tctx := ctx.(*helmNodeVisitorContext)

		if node.Kind == yaml.ScalarNode && node.Value == NameMetadata {
			tctx.inMetadata = true
			return tctx, nil, nil
		} else if tctx.inMetadata && node.Kind == yaml.MappingNode {
			// We are now in the metadata map
			tctx.inMetadata = false
			return tctx, innerMetadataVisitor, nil
		}
		return tctx, nil, nil
	})
)

type helmDocumentVisitor struct {
	chartName string
}

func (h helmDocumentVisitor) Visit(doc map[string]any) error {
	var md map[string]any
	if _md, ok := doc[NameMetadata]; ok {
		md = _md.(map[string]any)
	} else {
		md = map[string]any{}
		doc[NameMetadata] = md
	}
	md[NameLabels] = map[string]any{
		// TODO - make this into an argument
		LableNameAppShort:     `{{ template "` + chartName + `.name" . }}`,
		LableNameAppLong:      `{{ template "` + chartName + `.name" . }}`,
		LableNameInstanceLong: `{{ .Release.Name }}`,
	}
	md[NameAnnotations] = map[string]any{
		AnnotationNameHelmResourcePolicy: "keep",
	}

	return nil
}
