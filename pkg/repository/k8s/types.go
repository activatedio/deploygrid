package k8s

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ApplicationSourceSpec struct {
	Chart          string `json:"chart"`
	Path           string `json:"path"`
	RepoURL        string `json:"repoURL"`
	TargetRevision string `json:"targetRevision"`
}

type ApplicationDestinationSpec struct {
	Server    string `json:"server"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type ApplicationSpec struct {
	Destination ApplicationDestinationSpec `json:"destination"`
	Source      ApplicationSourceSpec      `json:"source"`
}

type ApplicationHealthStatus struct {
	Status string `json:"status"`
}

type ApplicationSyncStatus struct {
	Status   string `json:"status"`
	Revision string `json:"revision"`
}

type ApplicationStatus struct {
	Health ApplicationHealthStatus `json:"health"`
	Sync   ApplicationSyncStatus   `json:"sync"`
}

type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ApplicationSpec   `json:"spec,omitempty"`
	Status            ApplicationStatus `json:"status,omitempty"`
}
