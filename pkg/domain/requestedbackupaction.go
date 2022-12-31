package domain

import (
	"github.com/riotkit-org/backup-maker-controller/pkg/apis/riotkit/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// todo: move to integration package
func (a RequestedBackupActionAggregate) AcceptedResourceTypes() []v1.GroupVersionKind {
	return []v1.GroupVersionKind{
		{Group: "batch", Version: "v1", Kind: "Job"},
		{Group: "", Version: "v1", Kind: "Pod"},

		//// Tekton Pipelines v1alpha1
		//{Group: "tekton.dev", Version: "v1alpha1", Kind: "PipelineRun"},
		//{Group: "tekton.dev", Version: "v1alpha1", Kind: "TaskRun"},
		//{Group: "tekton.dev", Version: "v1alpha1", Kind: "Pipeline"},
		//{Group: "tekton.dev", Version: "v1alpha1", Kind: "Task"},
		//
		//// Tekton Pipelines v1
		//{Group: "tekton.dev", Version: "v1", Kind: "PipelineRun"},
		//{Group: "tekton.dev", Version: "v1", Kind: "TaskRun"},
		//{Group: "tekton.dev", Version: "v1", Kind: "Pipeline"},
		//{Group: "tekton.dev", Version: "v1", Kind: "Task"},
		//
		//// Argo Workflows
		//{Group: "argoproj.io", Version: "v1alpha1", Kind: "Workflow"},
		//{Group: "argoproj.io", Version: "v1alpha1", Kind: "WorkflowTemplate"},
	}
}

func (a RequestedBackupActionAggregate) GetOperation() Operation {
	return Operation(a.Spec.Action)
}

func (sb RequestedBackupActionAggregate) ShouldCreateCronJob() bool {
	return false
}

func (a RequestedBackupActionAggregate) GetTemplate() Template {
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

func (a *RequestedBackupActionAggregate) ShouldRenderDependentObjectsForAllOperationTypes() bool {
	return false
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
