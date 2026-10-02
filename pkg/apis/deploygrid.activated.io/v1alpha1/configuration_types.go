/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigurationSpec defines the desired state of Configuration.
type ConfigurationSpec struct {
	// System is the name of the System (same namespace) these values belong
	// to.
	// +kubebuilder:validation:MinLength=1
	System string `json:"system"`
	// Environment scopes the values to one environment. When empty the values
	// apply to every environment and are overridden by environment-scoped
	// Configurations with the same name.
	// +optional
	Environment string `json:"environment,omitempty"`
	// Values is an arbitrary document rendered by ConfigurationViews.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +optional
	Values *apiextensionsv1.JSON `json:"values,omitempty"`
}

// ConfigurationStatus defines the observed state of Configuration.
type ConfigurationStatus struct {
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=cfg
// +kubebuilder:printcolumn:name="System",type=string,JSONPath=`.spec.system`
// +kubebuilder:printcolumn:name="Environment",type=string,JSONPath=`.spec.environment`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Configuration is a named document of values attached to a System and
// optionally one of its environments. Environment-scoped documents are deep
// merged over the system-wide document of the same name when rendered.
type Configuration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConfigurationSpec   `json:"spec,omitempty"`
	Status ConfigurationStatus `json:"status,omitempty"`
}

// ConfigurationViewSpec defines the desired state of ConfigurationView.
type ConfigurationViewSpec struct {
	// System is the name of the System (same namespace) this view renders.
	// +kubebuilder:validation:MinLength=1
	System string `json:"system"`
	// +optional
	DisplayName string `json:"displayName,omitempty"`
	// Configuration is the name of the Configuration rendered by this view.
	// Defaults to the view's own name.
	// +optional
	Configuration string `json:"configuration,omitempty"`
	// ContentType of the rendered output.
	// +kubebuilder:default="text/plain"
	// +optional
	ContentType string `json:"contentType,omitempty"`
	// Template is a Go text/template. The context exposes .System,
	// .Environment, .Values (the merged Configuration), .Components (the
	// grid rows for the environment, each with .Name, .DisplayName,
	// .Version, .DesiredVersion, .Health, .Cluster, .Namespace and .Hosts)
	// and .Grid (the whole grid). Functions: default, upper, lower, title,
	// trim, replace, join, split, quote, indent, nindent, toYaml, toJson.
	// +kubebuilder:validation:MinLength=1
	Template string `json:"template"`
}

// ConfigurationViewStatus defines the observed state of ConfigurationView.
type ConfigurationViewStatus struct {
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Conditions include "TemplateValid".
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=cfgview
// +kubebuilder:printcolumn:name="System",type=string,JSONPath=`.spec.system`
// +kubebuilder:printcolumn:name="Content Type",type=string,JSONPath=`.spec.contentType`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ConfigurationView renders a Configuration for an environment through a
// template, for example to produce a values file or a status page fragment.
type ConfigurationView struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConfigurationViewSpec   `json:"spec,omitempty"`
	Status ConfigurationViewStatus `json:"status,omitempty"`
}
