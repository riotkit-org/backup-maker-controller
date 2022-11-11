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
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"
)

type ManagedJobObserver struct {
	Client       client.Client
	BRClient     v1alpha1.RiotkitV1alpha1Interface
	Integrations *integration.AllSupportedJobResourceTypes
	Fetcher      factory.CachedFetcher
}

func (r *ManagedJobObserver) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx) // todo: logrus.WithContext(ctx)
	logger.Info("Reconciling children")

	// Fetch and populate the context
	aggregate, _, err := factory.FetchRBAAggregate(ctx, r.Fetcher, r.Client, logger, req)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "cannot fetch RequestedBackupAction from cache")
	}

	// Collect the report about all managed resources in our context
	ownedReferences := aggregate.GetReferencesOfOwnedObjects()
	report := make([]riotkitorgv1alpha1.JobHealthStatus, 0)
	var healthy = true
	for _, resource := range ownedReferences {
		status, err := r.Integrations.GetScheduledJobHealthStatus(ctx, resource.GetGVK(), resource.TrackingId, req.Namespace)
		logrus.Debugf("JobStatus = %v, err = %v", status, err)
		if err != nil {
			if err.Error() == integration.ErrorUnrecognizedResourceType {
				continue
			}
			return ctrl.Result{RequeueAfter: time.Second * 30}, err
		}
		report = append(report, status)
		if status.Failed {
			healthy = false
		}
	}

	// Update the status
	r.updateStatus(ctx, aggregate, report, healthy)

	return ctrl.Result{}, nil
}

func (r *ManagedJobObserver) updateStatus(ctx context.Context, aggregate *domain.RequestedBackupActionAggregate, report []riotkitorgv1alpha1.JobHealthStatus, healthy bool) {
	retry.RetryOnConflict(retry.DefaultRetry, func() error {
		res, getErr := r.BRClient.RequestedBackupActions(aggregate.Namespace).Get(ctx, aggregate.Name, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		res.Status = aggregate.RequestedBackupAction.Status
		res.Status.ChildrenResourcesHealth = report
		res.Status.Healthy = healthy

		_, updateErr := r.BRClient.RequestedBackupActions(aggregate.Namespace).UpdateStatus(ctx, res, metav1.UpdateOptions{})
		logrus.Debugf(".status field updated with .ChildrenResourcesHealth and .Healthy")
		return updateErr
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *ManagedJobObserver) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&riotkitorgv1alpha1.RequestedBackupAction{}).
		//For(&riotkitorgv1alpha1.ScheduledBackup{}).
		Owns(&batchv1.Job{}).
		Owns(&batchv1.CronJob{}).
		WithEventFilter(predicate.Funcs{
			DeleteFunc: func(e event.DeleteEvent) bool {
				return false
			},
		}).
		Complete(r)
}
