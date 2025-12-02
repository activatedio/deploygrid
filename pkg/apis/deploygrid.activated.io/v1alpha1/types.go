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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MetadataSpec defines the desired state of Metadata.
type MetadataSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Metadata. Edit metadata_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// MetadataStatus defines the observed state of Metadata.
type MetadataStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Metadata is the Schema for the metadata API.
type Metadata struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MetadataSpec   `json:"spec,omitempty"`
	Status MetadataStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MetadataList contains a list of Metadata.
type MetadataList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Metadata `json:"items"`
}

// MetadataViewSpec defines the desired state of MetadataView.
type MetadataViewSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of MetadataView. Edit metadataview_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// MetadataViewStatus defines the observed state of MetadataView.
type MetadataViewStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// MetadataView is the Schema for the metadataviews API.
type MetadataView struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MetadataViewSpec   `json:"spec,omitempty"`
	Status MetadataViewStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MetadataViewList contains a list of MetadataView.
type MetadataViewList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MetadataView `json:"items"`
}

type SystemEnvironment struct {
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description,omitempty"`
}

// SystemSpec defines the desired state of System.
type SystemSpec struct {
	DisplayName  string              `json:"displayName,omitempty"`
	Description  string              `json:"description,omitempty"`
	Environments []SystemEnvironment `json:"environments,omitempty"`
}

// SystemStatus defines the observed state of System.
type SystemStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// System is the Schema for the systems API.
type System struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SystemSpec   `json:"spec,omitempty"`
	Status SystemStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SystemList contains a list of System.
type SystemList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []System `json:"items"`
}
