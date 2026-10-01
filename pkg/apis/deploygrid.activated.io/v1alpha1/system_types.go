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

// SystemEnvironment is one column of a System's grid. Environments are
// displayed in the order they are declared, which is also treated as the
// promotion order (earlier environments are "lower").
type SystemEnvironment struct {
	// Name is the stable identifier used in labels, annotations and the API.
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	// +kubebuilder:validation:MaxLength=63
	Name string `json:"name"`
	// DisplayName is shown in the UI. Defaults to Name.
	// +optional
	DisplayName string `json:"displayName,omitempty"`
	// +optional
	Description string `json:"description,omitempty"`
}

// SystemGroup is a named display grouping for Components. Groups are
// displayed in the order declared; components whose group is not declared
// are appended alphabetically.
type SystemGroup struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// +optional
	DisplayName string `json:"displayName,omitempty"`
	// +optional
	Description string `json:"description,omitempty"`
}

// SystemSpec defines the desired state of System.
type SystemSpec struct {
	// +optional
	DisplayName string `json:"displayName,omitempty"`
	// +optional
	Description string `json:"description,omitempty"`
	// Environments are the columns of the grid, in display order.
	// +kubebuilder:validation:MinItems=1
	// +listType=map
	// +listMapKey=name
	Environments []SystemEnvironment `json:"environments"`
	// Groups are the row groupings, in display order.
	// +optional
	// +listType=map
	// +listMapKey=name
	Groups []SystemGroup `json:"groups,omitempty"`
}

// SystemStatus defines the observed state of System.
type SystemStatus struct {
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// Components is the number of Components (declared and discovered) that
	// belong to this System.
	// +optional
	Components int32 `json:"components,omitempty"`
	// LastObservedTime is the time of the most recent observation that
	// affected this System.
	// +optional
	LastObservedTime *metav1.Time `json:"lastObservedTime,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=sys
// +kubebuilder:printcolumn:name="Display Name",type=string,JSONPath=`.spec.displayName`
// +kubebuilder:printcolumn:name="Components",type=integer,JSONPath=`.status.components`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// System is a product or platform boundary: one grid. It declares the
// ordered environments (columns) and groups (row sections) of that grid.
type System struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SystemSpec   `json:"spec,omitempty"`
	Status SystemStatus `json:"status,omitempty"`
}
