package controllers

import (
	"context"
	"github.com/pkg/errors"
	riotkitorgv1alpha1 "github.com/riotkit-org/backup-maker-operator/pkg/apis/riotkit/v1alpha1"
	"github.com/riotkit-org/backup-maker-operator/pkg/client/clientset/versioned/typed/riotkit/v1alpha1"
	"github.com/riotkit-org/backup-maker-operator/pkg/domain"
	"github.com/riotkit-org/backup-maker-operator/pkg/factory"
	"github.com/riotkit-org/backup-maker-operator/pkg/integration"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// -----------------------------------------------------------------------------------------------------------------------------------------------------------------
// RequestedBackupAction is spawning multiple objects, mostly objects creating Pods. Those Pods are performing a Backup or Restore action,
// and each action has a RESULT - In progress, Failed or Succeeded. This result is collected and applied back to the RequestedBackupAction as an aggregated status.
// -----------------------------------------------------------------------------------------------------------------------------------------------------------------

type JobsManagedByScheduledBackupObserver struct {
	Client       client.Client
	BRClient     v1alpha1.RiotkitV1alpha1Interface
	Integrations *integration.AllSupportedJobResourceTypes
	Fetcher      factory.CachedFetcher
}

func (r *JobsManagedByScheduledBackupObserver) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logrus.WithContext(ctx).WithFields(map[string]interface{}{
		"name":       req.NamespacedName,
		"controller": "JobsManagedByScheduledBackupObserver",
	})
	logger.Info("Reconciling children")

	// Fetch and populate the context
	aggregate, err := factory.FetchSBAggregate(ctx, r.Fetcher, r.Client, logger, req)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "cannot fetch ScheduledBackup from cache")
	}

	// Collect the report about all managed resources in our context
	ownedReferences := aggregate.GetReferencesOfOwnedObjects()
	report, healthy, err := createOwnedReferencesHealthReport(ctx, ownedReferences, r.Integrations, logger, req.Namespace)

	// Update the status
	r.updateStatus(ctx, logger, aggregate, report, healthy)

	return ctrl.Result{}, nil
}

func (r *JobsManagedByScheduledBackupObserver) updateStatus(ctx context.Context, logger *logrus.Entry, aggregate *domain.ScheduledBackupAggregate, report []riotkitorgv1alpha1.JobHealthStatus, healthy bool) {
	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		res, getErr := r.BRClient.ScheduledBackups(aggregate.Namespace).Get(ctx, aggregate.Name, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		res.Status = aggregate.ScheduledBackup.Status
		res.Status.ChildrenResourcesHealth = report
		res.Status.Healthy = healthy

		_, updateErr := r.BRClient.ScheduledBackups(aggregate.Namespace).UpdateStatus(ctx, res, metav1.UpdateOptions{})
		logger.Debugf(".status field updated with .ChildrenResourcesHealth and .Healthy")
		return updateErr
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *JobsManagedByScheduledBackupObserver) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&riotkitorgv1alpha1.ScheduledBackup{}).
		Owns(&batchv1.Job{}).
		Owns(&batchv1.CronJob{}).
		WithEventFilter(predicate.Funcs{
			DeleteFunc: func(e event.DeleteEvent) bool {
				return false
			},
		}).
		Complete(r)
}
