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
	"strconv"
)

// ClusterBackupProcedureTemplateSpec defines the desired state of ClusterBackupProcedureTemplate
type ClusterBackupProcedureTemplateSpec struct {
	Image   string `json:"image"`
	Backup  string `json:"backup"`
	Restore string `json:"restore"`
}

// ClusterBackupProcedureTemplateStatus defines the observed state of ClusterBackupProcedureTemplate
type ClusterBackupProcedureTemplateStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ClusterBackupProcedureTemplate is the Schema for the clusterbackupproceduretemplates API
type ClusterBackupProcedureTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterBackupProcedureTemplateSpec   `json:"spec,omitempty"`
	Status ClusterBackupProcedureTemplateStatus `json:"status,omitempty"`
}

func (in *ClusterBackupProcedureTemplate) GetCacheId() string {
	return in.Name + "_" + in.Namespace + "_" + in.GenerateName + "_" + strconv.Itoa(int(in.GetGeneration()))
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// ClusterBackupProcedureTemplateList contains a list of ClusterBackupProcedureTemplate
type ClusterBackupProcedureTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterBackupProcedureTemplate `json:"items"`
}
