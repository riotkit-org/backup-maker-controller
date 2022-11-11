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
	"crypto/sha256"
	"encoding/hex"
	json "encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
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
	SecretName     string   `json:"secretName,omitempty"`
	ImportOnlyKeys []string `json:"importOnlyKeys,omitempty"`
}

// VarsSpec represents .spec.vars - a hashmap of values applied to template's backup & restore scripts
type VarsSpec string

// ScheduledBackupSpec defines the desired state of ScheduledBackup
type ScheduledBackupSpec struct {
	CollectionId    string           `json:"collectionId"`
	TemplateRef     TemplateSpec     `json:"templateRef"`
	GPGKeySecretRef GPGKeySecretSpec `json:"gpgKeySecretRef"`
	TokenSecretRef  TokenSecretSpec  `json:"tokenSecretRef"`
	VarsSecretRef   VarsSecretSpec   `json:"varsSecretRef"`
	Vars            VarsSpec         `json:"vars"`
	CronJob         CronJobSpec      `json:"cronJob"`

	// +kubebuilder:validation:Enum=backup;restore
	Operation string `json:"operation"`
}

func (in *ScheduledBackupSpec) CalculateHash() string {
	doc, _ := json.Marshal(in)
	sum := sha256.Sum256(doc)
	return hex.EncodeToString(sum[0:])
}

type CronJobSpec struct {
	Enabled bool `json:"enabled"`

	// +kubebuilder:default:="00 02 * * *"
	ScheduleEvery string `json:"scheduleEvery,omitempty"`
}

// ScheduledBackupStatus defines the observed state of ScheduledBackup
type ScheduledBackupStatus struct {
	LastAppliedSpecHash     string             `json:"lastAppliedSpecHash,omitempty"`
	Conditions              []metav1.Condition `json:"conditions,omitempty"`
	OwnedReferences         ChildrenReferences `json:"ownedReferences"`
	ChildrenResourcesHealth []JobHealthStatus  `json:"childrenResourcesHealth"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Schedule",type="string",JSONPath=".spec.cronjob.scheduleEvery",description="Cron expression"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// ScheduledBackup is the Schema for the scheduledbackups API
type ScheduledBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScheduledBackupSpec   `json:"spec,omitempty"`
	Status ScheduledBackupStatus `json:"status,omitempty"`
}

// HasSpecChanged is telling if the current object's spec differs from the already applied object, basing on the status field
func (in *ScheduledBackup) HasSpecChanged() bool {
	return in.Spec.CalculateHash() != in.Status.LastAppliedSpecHash
}

func (in *ScheduledBackup) IsBeingReconciledAlready() bool {
	for _, condition := range in.Status.Conditions {
		timeDiff := time.Now().Sub(condition.LastTransitionTime.Time).Seconds()

		// if there were less than 60 seconds from previous reconcilation
		if timeDiff < 60 && condition.Status == "Unknown" {
			return true
		}
	}
	return false
}

func (in *ScheduledBackup) GetTypeMeta() *metav1.TypeMeta {
	return &in.TypeMeta
}

func (in *ScheduledBackup) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ScheduledBackupList contains a list of ScheduledBackup
type ScheduledBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ScheduledBackup `json:"items"`
}
