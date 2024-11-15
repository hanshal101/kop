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

// PodBackupSpec defines the desired state of PodBackup.
type PodBackupSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	PodName           string `json:"podName"`
	Namespace         string `json:"namespace"`
	PodBackupLocation string `json:"podBackupLocation"`
	PodBackupVolumes  bool   `json:"podBackupVolumes"`
	PodBackupTime     int64  `json:"podBackupTime"`
}

// PodBackupStatus defines the observed state of PodBackup.
type PodBackupStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	LastBackup string `json:"lastBackup"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PodBackup is the Schema for the podbackups API.
type PodBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PodBackupSpec   `json:"spec,omitempty"`
	Status PodBackupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PodBackupList contains a list of PodBackup.
type PodBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PodBackup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PodBackup{}, &PodBackupList{})
}
