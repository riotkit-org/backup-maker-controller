package aggregates

import (
	"github.com/riotkit-org/backup-maker-operator/pkg/apis/riotkit/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Renderable interface {
	AcceptedResourceTypes() []v1.GroupVersionKind
	GetOperation() string
	GetTemplate() *v1alpha1.ClusterBackupProcedureTemplate
	GetScheduledBackup() *v1alpha1.ScheduledBackup
	GetBackupAggregate() *ScheduledBackupAggregate
	GetObjectForOwnerReference() KubernetesResource
}

type KubernetesResource interface {
	GetTypeMeta() *v1.TypeMeta
	GetObjectMeta() *v1.ObjectMeta
}
