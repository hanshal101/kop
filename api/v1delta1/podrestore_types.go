/*
Copyright 2024 hanshal101.

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

package v1delta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PodRestoreSpec defines the desired state of PodRestore.
type PodRestoreSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	BackupName      string `json:"backupName"`
	BackupNamespace string `json:"backupNamespace"`
	AutoRestore     bool   `json:"autoRestore"`
}

// PodRestoreStatus defines the observed state of PodRestore.
type PodRestoreStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	LastRestore       string `json:"lastRestore"`
	LastRestoreStatus string `json:"lastRestoreStatus"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PodRestore is the Schema for the podrestores API.
type PodRestore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PodRestoreSpec   `json:"spec,omitempty"`
	Status PodRestoreStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PodRestoreList contains a list of PodRestore.
type PodRestoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PodRestore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PodRestore{}, &PodRestoreList{})
}
