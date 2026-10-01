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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ComponentKind classifies what a Component's version refers to.
// +kubebuilder:validation:Enum=helm-chart;container;argocd-application;service;custom
type ComponentKind string

const (
	ComponentKindHelmChart         ComponentKind = "helm-chart"
	ComponentKindContainer         ComponentKind = "container"
	ComponentKindArgoCDApplication ComponentKind = "argocd-application"
	ComponentKindService           ComponentKind = "service"
	ComponentKindCustom            ComponentKind = "custom"
)

// ComponentSelector describes how observed resources are matched to this
// Component. All specified criteria must match. When empty, the component is
// matched by the deploygrid.activated.io/component label or annotation whose
// value equals the Component name.
type ComponentSelector struct {
	// MatchLabels must all be present on the observed resource.
	// +optional
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
	// Namespaces restricts matches to these namespaces (in any cluster).
	// +optional
	Namespaces []string `json:"namespaces,omitempty"`
	// Names restricts matches to resources with one of these names.
	// +optional
	Names []string `json:"names,omitempty"`
}

// ComponentLink is a templated URL rendered per environment. The template
// context exposes .System, .Component, .Environment, .Cluster, .Namespace,
// .Version and .Hosts.
type ComponentLink struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// +kubebuilder:validation:MinLength=1
	URLTemplate string `json:"urlTemplate"`
}

// ComponentSpec defines the desired state of Component.
type ComponentSpec struct {
	// System is the name of the System (same namespace) this Component
	// belongs to.
	// +kubebuilder:validation:MinLength=1
	System string `json:"system"`
	// +optional
	DisplayName string `json:"displayName,omitempty"`
	// +optional
	Description string `json:"description,omitempty"`
	// Group places the component under a System group. Unknown groups are
	// created on the fly in alphabetical order after declared groups.
	// +optional
	Group string `json:"group,omitempty"`
	// Kind classifies the version shown in the cell.
	// +kubebuilder:default=custom
	// +optional
	Kind ComponentKind `json:"kind,omitempty"`
	// Parent is the name of another Component in the same System under which
	// this one is nested in the grid.
	// +optional
	Parent string `json:"parent,omitempty"`
	// Order is a sort hint within the group; lower sorts first, ties sort by
	// name.
	// +optional
	Order *int32 `json:"order,omitempty"`
	// +optional
	Selector ComponentSelector `json:"selector,omitempty"`
	// +optional
	// +listType=map
	// +listMapKey=name
	Links []ComponentLink `json:"links,omitempty"`
}

// ComponentEnvironmentStatus is the current cell for one environment.
type ComponentEnvironmentStatus struct {
	Environment string `json:"environment"`
	// Version is the running (actual) version.
	// +optional
	Version string `json:"version,omitempty"`
	// DesiredVersion is the version declared by the delivery source (for
	// example an Argo CD Application targetRevision), when known.
	// +optional
	DesiredVersion string `json:"desiredVersion,omitempty"`
	// +optional
	Cluster string `json:"cluster,omitempty"`
	// +optional
	Namespace string `json:"namespace,omitempty"`
	// Health is a coarse rollup: Healthy, Progressing, Degraded or Unknown.
	// +optional
	Health string `json:"health,omitempty"`
	// Hosts are ingress hostnames serving this component.
	// +optional
	Hosts []string `json:"hosts,omitempty"`
	// LastChangeTime is when Version last changed.
	// +optional
	LastChangeTime *metav1.Time `json:"lastChangeTime,omitempty"`
	// LastObservedTime is when this cell was last reported.
	// +optional
	LastObservedTime *metav1.Time `json:"lastObservedTime,omitempty"`
}

// ComponentStatus defines the observed state of Component.
type ComponentStatus struct {
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// Discovered is true when this Component was created automatically from
	// observed resources rather than declared by a user.
	// +optional
	Discovered bool `json:"discovered,omitempty"`
	// Environments holds the current cell for each environment in which the
	// component has been observed.
	// +optional
	// +listType=map
	// +listMapKey=environment
	Environments []ComponentEnvironmentStatus `json:"environments,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=comp
// +kubebuilder:printcolumn:name="System",type=string,JSONPath=`.spec.system`
// +kubebuilder:printcolumn:name="Group",type=string,JSONPath=`.spec.group`
// +kubebuilder:printcolumn:name="Kind",type=string,JSONPath=`.spec.kind`
// +kubebuilder:printcolumn:name="Discovered",type=boolean,JSONPath=`.status.discovered`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Component is one row of a System's grid: a deployable unit whose identity
// is stable across environments. Its status carries the current version in
// each environment.
type Component struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ComponentSpec   `json:"spec,omitempty"`
	Status ComponentStatus `json:"status,omitempty"`
}
