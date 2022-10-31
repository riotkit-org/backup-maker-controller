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

// RequestedBackupActionSpec defines the desired state of RequestedBackupAction
type RequestedBackupActionSpec struct {
	// +kubebuilder:validation:Enum=backup;restore
	Action             string        `json:"action"`
	TargetVersion      string        `json:"targetVersion,omitempty"` // can be empty, when action = "backup"
	ScheduledBackupRef BackupRefSpec `json:"scheduledBackupRef"`
}

// RequestedBackupActionStatus defines the observed state of RequestedBackupAction
type RequestedBackupActionStatus struct {
	Processed  bool               `json:"processed"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// RequestedBackupAction is the Schema for the requestedbackupactions API
type RequestedBackupAction struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// todo: make .spec immutable

	Spec   RequestedBackupActionSpec   `json:"spec,omitempty"`
	Status RequestedBackupActionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RequestedBackupActionList contains a list of RequestedBackupAction
type RequestedBackupActionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RequestedBackupAction `json:"items"`
}
