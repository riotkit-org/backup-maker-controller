package aggregates

import (
	"github.com/riotkit-org/backup-maker-operator/pkg/apis/riotkit/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RequestedBackupActionAggregate is aggregating already hydrated (fetched from cache/cluster) objects all together
type RequestedBackupActionAggregate struct {
	*v1alpha1.RequestedBackupAction

	Scheduled *ScheduledBackupAggregate
}

func (a RequestedBackupActionAggregate) MarkAsProcessed() {
	a.Status.Processed = true
}

func (a RequestedBackupActionAggregate) WasAlreadyProcessed() bool {
	return a.Status.Processed
}

func (a RequestedBackupActionAggregate) AcceptedResourceTypes() []v1.GroupVersionKind {
	return []v1.GroupVersionKind{
		{Group: "batch", Version: "v1", Kind: "Job"},
	}
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
