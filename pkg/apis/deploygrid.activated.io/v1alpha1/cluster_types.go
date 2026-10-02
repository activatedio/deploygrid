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

// ClusterCollectionMode says how observations for a Cluster reach the server.
// +kubebuilder:validation:Enum=agent;kubeconfig;local
type ClusterCollectionMode string

const (
	// ClusterCollectionModeAgent means a collector running in the cluster
	// pushes observations to the server.
	ClusterCollectionModeAgent ClusterCollectionMode = "agent"
	// ClusterCollectionModeKubeconfig means the server pulls using a
	// kubeconfig stored in a Secret (the v1 behaviour).
	ClusterCollectionModeKubeconfig ClusterCollectionMode = "kubeconfig"
	// ClusterCollectionModeLocal means the server collects from the cluster
	// it is running in using its service account.
	ClusterCollectionModeLocal ClusterCollectionMode = "local"
)

// SecretKeyRef points at a key in a Secret in the same namespace. The key
// defaults depend on the use: "config" for a kubeconfig, "token" for a
// collector token.
type SecretKeyRef struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// +optional
	Key string `json:"key,omitempty"`
}

// ClusterCollection configures how the cluster is observed.
type ClusterCollection struct {
	// +kubebuilder:default=agent
	// +optional
	Mode ClusterCollectionMode `json:"mode,omitempty"`
	// KubeconfigSecretRef is required for mode=kubeconfig. The key defaults
	// to "config".
	// +optional
	KubeconfigSecretRef *SecretKeyRef `json:"kubeconfigSecretRef,omitempty"`
	// TokenSecretRef holds the bearer token an agent must present for
	// mode=agent; the key defaults to "token". When unset the server
	// generates Secret <cluster>-collector-token.
	// +optional
	TokenSecretRef *SecretKeyRef `json:"tokenSecretRef,omitempty"`
	// InsecureSkipTLSVerify disables TLS verification for mode=kubeconfig.
	// Intended for local development only.
	// +optional
	InsecureSkipTLSVerify bool `json:"insecureSkipTLSVerify,omitempty"`
}

// ClusterNamespaceRule maps namespaces on this cluster to an environment.
type ClusterNamespaceRule struct {
	// Match is a glob (for example "qa-*") applied to the namespace name.
	// +kubebuilder:validation:MinLength=1
	Match string `json:"match"`
	// +kubebuilder:validation:MinLength=1
	Environment string `json:"environment"`
}

// ClusterSpec defines the desired state of Cluster.
type ClusterSpec struct {
	// +optional
	DisplayName string `json:"displayName,omitempty"`
	// +optional
	Description string `json:"description,omitempty"`
	// Addresses are API server URLs by which other resources refer to this
	// cluster, for example an Argo CD Application destination.server.
	// +optional
	Addresses []string `json:"addresses,omitempty"`
	// Environment is the default environment for resources observed on this
	// cluster when neither a label/annotation nor a namespace rule applies.
	// +optional
	Environment string `json:"environment,omitempty"`
	// NamespaceRules are evaluated in order; the first match wins.
	// +optional
	NamespaceRules []ClusterNamespaceRule `json:"namespaceRules,omitempty"`
	// +optional
	Collection ClusterCollection `json:"collection,omitempty"`
}

// ClusterStatus defines the observed state of Cluster.
type ClusterStatus struct {
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Conditions include "Connected".
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// +optional
	LastHeartbeatTime *metav1.Time `json:"lastHeartbeatTime,omitempty"`
	// +optional
	CollectorVersion string `json:"collectorVersion,omitempty"`
	// +optional
	KubernetesVersion string `json:"kubernetesVersion,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Environment",type=string,JSONPath=`.spec.environment`
// +kubebuilder:printcolumn:name="Mode",type=string,JSONPath=`.spec.collection.mode`
// +kubebuilder:printcolumn:name="Connected",type=string,JSONPath=`.status.conditions[?(@.type=="Connected")].status`
// +kubebuilder:printcolumn:name="Heartbeat",type=date,JSONPath=`.status.lastHeartbeatTime`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Cluster registers a Kubernetes cluster that deploygrid observes, and maps
// what runs there to environments.
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}
