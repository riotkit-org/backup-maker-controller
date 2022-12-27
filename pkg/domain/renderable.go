package domain

import (
	"github.com/riotkit-org/backup-maker-operator/pkg/apis/riotkit/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Renderable interface {
	AcceptedResourceTypes() []v1.GroupVersionKind
	GetOperation() Operation
	GetTemplate() *v1alpha1.ClusterBackupProcedureTemplate
	GetScheduledBackup() *v1alpha1.ScheduledBackup
	GetBackupAggregate() *ScheduledBackupAggregate
	GetObjectForOwnerReference() KubernetesResource
	AddOwnedObject(doc *unstructured.Unstructured)
	ShouldRenderDependentObjectsForAllOperationTypes() bool
	ShouldCreateCronJob() bool
}

type KubernetesResource interface {
	GetTypeMeta() *v1.TypeMeta
	GetObjectMeta() *v1.ObjectMeta
}
