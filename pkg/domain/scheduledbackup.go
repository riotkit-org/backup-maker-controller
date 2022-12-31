package domain

import (
	"github.com/riotkit-org/backup-maker-controller/pkg/apis/riotkit/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type AdditionalVarsList map[string][]byte

// ScheduledBackupAggregate is aggregating already hydrated (fetched from cache/cluster) objects all together
type ScheduledBackupAggregate struct {
	*v1alpha1.ScheduledBackup

	Template           Template
	GPGSecret          *v1.Secret
	TokenSecret        *v1.Secret
	VarsListSecret     *v1.Secret
	AdditionalVarsList AdditionalVarsList
}

func (sb ScheduledBackupAggregate) AcceptedResourceTypes() []metav1.GroupVersionKind {
	return []metav1.GroupVersionKind{}
}

func (sb ScheduledBackupAggregate) GetOperation() Operation {
	return Operation(sb.Spec.Operation)
}

func (sb ScheduledBackupAggregate) ShouldCreateCronJob() bool {
	return sb.Spec.CronJob.Enabled
}

func (sb ScheduledBackupAggregate) GetTemplate() Template {
	return sb.Template
}

func (sb ScheduledBackupAggregate) GetScheduledBackup() *v1alpha1.ScheduledBackup {
	return sb.ScheduledBackup
}

func (sb ScheduledBackupAggregate) GetBackupAggregate() *ScheduledBackupAggregate {
	return &sb
}

func (sb ScheduledBackupAggregate) GetObjectForOwnerReference() KubernetesResource {
	return sb.ScheduledBackup
}

func (sb ScheduledBackupAggregate) GetReferencesOfOwnedObjects() v1alpha1.ChildrenReferences {
	return sb.ScheduledBackup.Status.OwnedReferences
}

// AddOwnedObject is adding a child element
func (sb *ScheduledBackupAggregate) AddOwnedObject(doc *unstructured.Unstructured) {
	v1alpha1.AddOwnedObject(&sb.Status.OwnedReferences, doc)
}

func (sb *ScheduledBackupAggregate) ShouldRenderDependentObjectsForAllOperationTypes() bool {
	return true
}
