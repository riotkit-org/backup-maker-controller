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

// TemplateSpec represents .spec.templateRef section
type TemplateSpec struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// GPGKeySecretSpec represents .spec.gpgKeySecretRef section
type GPGKeySecretSpec struct {
	SecretName    string `json:"secretName"`
	PublicKey     string `json:"publicKey"`
	PrivateKey    string `json:"privateKey"`
	PassphraseKey string `json:"passphraseKey"`
	Email         string `json:"email"`

	CreateIfNotExists bool `json:"createIfNotExists"`
}

// TokenSecretSpec represents .spec.tokenSecretRef
type TokenSecretSpec struct {
	SecretName string `json:"secretName"`
	TokenKey   string `json:"tokenKey"`
}

// VarsSecretSpec represents .spec.varsSecretRef
type VarsSecretSpec struct {
	SecretName     string   `json:"secretName"`
	ImportOnlyKeys []string `json:"importOnlyKeys"`
}

// VarsSpec represents .spec.vars - a hashmap of values applied to template's backup & restore scripts
type VarsSpec map[string]string

// ScheduledBackupSpec defines the desired state of ScheduledBackup
type ScheduledBackupSpec struct {
	CollectionId    string           `json:"collectionId"`
	TemplateRef     TemplateSpec     `json:"templateRef"`
	GPGKeySecretRef GPGKeySecretSpec `json:"gpgKeySecretRef"`
	TokenSecretRef  TokenSecretSpec  `json:"tokenSecretRef"`
	VarsSecretRef   VarsSecretSpec   `json:"varsSecretRef"`
	Vars            VarsSpec         `json:"vars"`
}

// ScheduledBackupStatus defines the observed state of ScheduledBackup
type ScheduledBackupStatus struct {
	WorkloadStatus  bool `json:"workloadStatus"`
	ConfigmapStatus bool `json:"configmapStatus"`
	SecretStatus    bool `json:"secretStatus"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ScheduledBackup is the Schema for the scheduledbackups API
type ScheduledBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScheduledBackupSpec   `json:"spec,omitempty"`
	Status ScheduledBackupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ScheduledBackupList contains a list of ScheduledBackup
type ScheduledBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ScheduledBackup `json:"items"`
}
