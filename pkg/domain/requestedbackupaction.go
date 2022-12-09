package domain

import (
	"github.com/riotkit-org/backup-maker-operator/pkg/apis/riotkit/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// RequestedBackupActionAggregate is aggregating already hydrated (fetched from cache/cluster) objects all together
type RequestedBackupActionAggregate struct {
	*v1alpha1.RequestedBackupAction

	Scheduled *ScheduledBackupAggregate
}

func (a *RequestedBackupActionAggregate) MarkAsProcessed() {
	a.Status.Processed = true
}

func (a RequestedBackupActionAggregate) WasAlreadyProcessed() bool {
	return a.Status.Processed
}

func (a RequestedBackupActionAggregate) GetReferencesOfOwnedObjects() v1alpha1.ChildrenReferences {
	return a.RequestedBackupAction.Status.OwnedReferences
}

func (a RequestedBackupActionAggregate) GetOperation() string {
	return a.Spec.Action
}

func (a RequestedBackupActionAggregate) GetTemplate() *v1alpha1.ClusterBackupProcedureTemplate {
	return a.Scheduled.Template
}

func (a RequestedBackupActionAggregate) GetScheduledBackup() *v1alpha1.ScheduledBackup {
	return a.Scheduled.ScheduledBackup
}

func (a RequestedBackupActionAggregate) GetBackupAggregate() *ScheduledBackupAggregate {
	return a.Scheduled
}

func (a *RequestedBackupActionAggregate) SetTargetKindType(name string) {
	a.Scheduled.AdditionalVarsList["HelmValues.kindType"] = []byte(name)
}

func (a RequestedBackupActionAggregate) GetObjectForOwnerReference() KubernetesResource {
	return a.RequestedBackupAction
}

// AddOwnedObject is adding a child element
func (a *RequestedBackupActionAggregate) AddOwnedObject(doc *unstructured.Unstructured) {
	v1alpha1.AddOwnedObject(&a.Status.OwnedReferences, doc)
}

func NewRequestedBackupActionAggregate(action *v1alpha1.RequestedBackupAction, scheduled *ScheduledBackupAggregate) *RequestedBackupActionAggregate {
	aggregate := RequestedBackupActionAggregate{}
	aggregate.RequestedBackupAction = action
	aggregate.Scheduled = scheduled
	aggregate.RequestedBackupAction.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "riotkit.org",
		Version: "v1alpha1",
		Kind:    "RequestedBackupAction",
	})

	targetKind := "Job"
	if action.Spec.KindType != "" {
		targetKind = action.Spec.KindType
	}
	aggregate.SetTargetKindType(targetKind)

	return &aggregate
}
