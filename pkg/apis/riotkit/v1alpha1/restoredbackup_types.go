/*
Copyright 2022 Riotkit.

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

type BackupRefSpec struct {
	Name string `json:"name"`
}

// RestoredBackupSpec defines the desired state of RestoredBackup
type RestoredBackupSpec struct {
	// Foo is an example field of RestoredBackup. Edit restoredbackup_types.go to remove/update
	Foo string `json:"foo,omitempty"`

	TargetVersion      string        `json:"targetVersion"`
	ScheduledBackupRef BackupRefSpec `json:"scheduledBackupRef"`
}

// RestoredBackupStatus defines the observed state of RestoredBackup
type RestoredBackupStatus struct {
	PodName   string `json:"podName"`
	Succeeded bool   `json:"succeeded"`
	Started   bool   `json:"started"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// RestoredBackup is the Schema for the restoredbackups API
type RestoredBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// todo: make .spec immutable

	Spec   RestoredBackupSpec   `json:"spec,omitempty"`
	Status RestoredBackupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RestoredBackupList contains a list of RestoredBackup
type RestoredBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RestoredBackup `json:"items"`
}
