package domain

import (
	"github.com/riotkit-org/backup-maker-operator/pkg/apis/riotkit/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Renderable interface {
	GetOperation() string
	GetTemplate() *v1alpha1.ClusterBackupProcedureTemplate
	GetScheduledBackup() *v1alpha1.ScheduledBackup
	GetBackupAggregate() *ScheduledBackupAggregate
	GetObjectForOwnerReference() KubernetesResource
	AddOwnedObject(doc *unstructured.Unstructured)
}

type KubernetesResource interface {
	GetTypeMeta() *v1.TypeMeta
	GetObjectMeta() *v1.ObjectMeta
}
