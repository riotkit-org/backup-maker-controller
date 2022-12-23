package controllers

import (
	"context"
	"github.com/pkg/errors"
	riotkitorgv1alpha1 "github.com/riotkit-org/backup-maker-operator/pkg/apis/riotkit/v1alpha1"
	"github.com/riotkit-org/backup-maker-operator/pkg/client/clientset/versioned/typed/riotkit/v1alpha1"
	"github.com/riotkit-org/backup-maker-operator/pkg/domain"
	"github.com/riotkit-org/backup-maker-operator/pkg/factory"
	"github.com/riotkit-org/backup-maker-operator/pkg/integration"
	"github.com/riotkit-org/backup-maker-operator/pkg/locking"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"
)

// -----------------------------------------------------------------------------------------------------------------------------------------------------------------
// RequestedBackupAction is spawning multiple objects, mostly objects creating Pods. Those Pods are performing a Backup or Restore action,
// and each action has a RESULT - In progress, Failed or Succeeded. This result is collected and applied back to the RequestedBackupAction as an aggregated status.
// -----------------------------------------------------------------------------------------------------------------------------------------------------------------

type JobsManagedByRequestedBackupActionObserver struct {
	Client       client.Client
	BRClient     v1alpha1.RiotkitV1alpha1Interface
	Integrations *integration.AllSupportedJobResourceTypes
	Fetcher      factory.CachedFetcher
	Locker       locking.Locker
}

func (r *JobsManagedByRequestedBackupActionObserver) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := createLogger(ctx, req, "JobsManagedByRequestedBackupActionObserver")

	//
	// 0. Do not allow doing same action multiple times at the same moment
	//
	lock := r.Locker.Obtain(ctx, req)
	if lock.AlreadyLocked() {
		logger.Infoln("Already processed, requeuing")
		return ctrl.Result{RequeueAfter: time.Second * 10}, nil
	}
	if lock.HasFailure() {
		return ctrl.Result{}, lock.GetError()
	}
	defer r.Locker.Done(ctx, lock)

	//
	// 1. Fetch and populate the context
	//
	aggregate, _, err := factory.FetchRBAAggregate(ctx, r.Fetcher, r.Client, logger, req)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "cannot fetch RequestedBackupAction from cache")
	}

	//
	// 2. Collect the report about all managed resources in our context
	//
	ownedReferences := aggregate.GetReferencesOfOwnedObjects()
	report, healthy, err := createOwnedReferencesHealthReport(ctx, ownedReferences, r.Integrations, logger, req.Namespace)

	//
	// 3. The Jobs are still running, wait for them to be finished (in next controller iteration - REQUEUE)
	//
	for _, healthStatus := range report {
		if healthStatus.Running {
			return ctrl.Result{RequeueAfter: time.Second * 10}, nil
		}
	}

	// Update the status
	r.updateStatus(ctx, aggregate, report, healthy, logger)

	return ctrl.Result{}, nil
}

func (r *JobsManagedByRequestedBackupActionObserver) updateStatus(ctx context.Context, aggregate *domain.RequestedBackupActionAggregate, report []riotkitorgv1alpha1.JobHealthStatus, healthy bool, logger *logrus.Entry) {
	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		res, getErr := r.BRClient.RequestedBackupActions(aggregate.Namespace).Get(ctx, aggregate.Name, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		res.Status = aggregate.RequestedBackupAction.Status
		res.Status.ChildrenResourcesHealth = report
		res.Status.Healthy = healthy

		_, updateErr := r.BRClient.RequestedBackupActions(aggregate.Namespace).UpdateStatus(ctx, res, metav1.UpdateOptions{})
		logger.Debugf(".status field updated with .ChildrenResourcesHealth and .Healthy")
		return updateErr
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *JobsManagedByRequestedBackupActionObserver) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&riotkitorgv1alpha1.RequestedBackupAction{}).
		Owns(&batchv1.Job{}).
		Owns(&batchv1.CronJob{}).
		WithEventFilter(predicate.Funcs{
			DeleteFunc: func(e event.DeleteEvent) bool {
				return false
			},
		}).
		Complete(r)
}
