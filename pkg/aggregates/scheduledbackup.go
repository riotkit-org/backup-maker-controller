package aggregates

import (
	"github.com/riotkit-org/backup-maker-operator/pkg/apis/riotkit/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

// ScheduledBackupAggregate is aggregating already hydrated (fetched from cache/cluster) objects all together
type ScheduledBackupAggregate struct {
	*v1alpha1.ScheduledBackup

	Template       *v1alpha1.ClusterBackupProcedureTemplate
	GPGSecret      *v1.Secret
	TokenSecret    *v1.Secret
	VarsListSecret *v1.Secret
}
