/*
Copyright 2022 Riotkit.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"github.com/riotkit-org/backup-maker-operator/pkg/aggregates"
	riotkitorgv1alpha1 "github.com/riotkit-org/backup-maker-operator/pkg/apis/riotkit/v1alpha1"
	"github.com/riotkit-org/backup-maker-operator/pkg/bmg"
	"github.com/riotkit-org/backup-maker-operator/pkg/client/clientset/versioned/typed/riotkit/v1alpha1"
	"github.com/riotkit-org/backup-maker-operator/pkg/factory"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

// ScheduledBackupReconciler reconciles a ScheduledBackup object
type ScheduledBackupReconciler struct {
	client.Client
	RestCfg   *rest.Config
	DynClient dynamic.Interface
	BRClient  v1alpha1.RiotkitV1alpha1Interface
	Scheme    *runtime.Scheme
	Cache     cache.Cache
	Fetcher   factory.CachedFetcher
	Recorder  record.EventRecorder
}

// +kubebuilder:rbac:groups=,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=riotkit.org,resources=scheduledbackups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=riotkit.org,resources=scheduledbackups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=riotkit.org,resources=scheduledbackups/finalizers,verbs=update

// Reconcile is the main loop for ScheduledBackup type objects
func (r *ScheduledBackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// todo: support case, when cron=false. Then do not create CronJob or Job objects. Such case would mean manual triggering of the backup process
	// todo: backup rotation - basing on the server settings

	//
	// Fetch reconciled object [ScheduledBackup]
	//
	backup, err := r.Fetcher.FetchScheduledBackup(ctx, req)
	logger.Info(fmt.Sprintf("Processing '%s' from '%s' namespace", backup.Name, backup.Namespace))
	if err != nil {
		return ctrl.Result{}, err
	}

	if backup.HasSpecChanged() && !backup.IsBeingReconciledAlready() {
		f := factory.NewFactory(r.Client, r.Fetcher, logger)
		aggregate, controllerAction, hydrateErr := f.CreateScheduledBackupAggregate(
			ctx, backup,
		)

		r.updateObject(ctx, aggregate, metav1.Condition{
			Status:  "Unknown",
			Message: "Reconciling to template and push target objects to the cluster (Jobs/CronJobs and related things)",
		})

		if hydrateErr != nil {
			r.updateObject(ctx, aggregate, metav1.Condition{
				Status:  "False",
				Message: fmt.Sprintf("Cannot find required dependencies: %s", hydrateErr.Error()),
			})
			r.Recorder.Event(backup, "Warning", "ErrorOccurred", hydrateErr.Error())

			if controllerAction == factory.ErrorActionRequeue {
				return ctrl.Result{RequeueAfter: time.Second * 5}, nil
			}
			return ctrl.Result{RequeueAfter: time.Minute * 15}, err
		}

		if applyErr := bmg.ApplyScheduledBackup(ctx, r.Recorder, r.RestCfg, r.DynClient, aggregate); applyErr != nil {
			r.updateObject(ctx, aggregate, metav1.Condition{
				Status:  "False",
				Message: fmt.Sprintf("Cannot template or apply objects to the cluster: %s", applyErr.Error()),
			})
			r.Recorder.Event(backup, "Warning", "ErrorOccurred", applyErr.Error())
			return ctrl.Result{RequeueAfter: time.Minute * 1}, nil
		}

		// todo: handle panics

		r.updateObject(ctx, aggregate, metav1.Condition{
			Status:  "True",
			Message: "Successfully templated and applied objects to the cluster",
		})
		r.Recorder.Event(backup, "Normal", "Updated", fmt.Sprintf("Successfully reconciled '%s' from '%s' namespace", backup.Name, backup.Namespace))
	} else {
		logrus.Infof("Spec not changed for '%s' from '%s' namespace", backup.Name, backup.Namespace)
	}

	return ctrl.Result{}, nil
}

// updateObject is updating the .status field
func (r *ScheduledBackupReconciler) updateObject(ctx context.Context, aggregate *aggregates.ScheduledBackupAggregate, condition metav1.Condition) {
	updateErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Fetch a fresh object to avoid: "the object has been modified; please apply your changes to the latest version and try again"
		res, getErr := r.BRClient.ScheduledBackups(aggregate.Namespace).Get(ctx, aggregate.Name, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}

		// Update condition
		condition.Reason = "SpecWasUpdated"
		condition.Type = "BackupObjectsInstallation"
		condition.ObservedGeneration = res.Generation
		if condition.Status == "True" {
			res.Status.LastAppliedSpecHash = aggregate.Spec.CalculateHash()
		}
		meta.SetStatusCondition(&res.Status.Conditions, condition)

		// Update object's status field
		_, updateErr := r.BRClient.ScheduledBackups(aggregate.Namespace).UpdateStatus(ctx, res, metav1.UpdateOptions{})
		return updateErr
	})
	if updateErr != nil {
		r.Recorder.Event(aggregate.ScheduledBackup, "Warning", "ErrorOccurred", fmt.Sprintf("Cannot update .status field: %s", updateErr.Error()))
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScheduledBackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&riotkitorgv1alpha1.ScheduledBackup{}).
		WithEventFilter(predicate.Funcs{
			DeleteFunc: func(e event.DeleteEvent) bool {
				return false
			},
		}).
		Complete(r)
}
