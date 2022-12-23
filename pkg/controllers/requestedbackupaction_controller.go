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
	riotkitorgv1alpha1 "github.com/riotkit-org/backup-maker-operator/pkg/apis/riotkit/v1alpha1"
	"github.com/riotkit-org/backup-maker-operator/pkg/bmg"
	"github.com/riotkit-org/backup-maker-operator/pkg/client/clientset/versioned/typed/riotkit/v1alpha1"
	"github.com/riotkit-org/backup-maker-operator/pkg/domain"
	"github.com/riotkit-org/backup-maker-operator/pkg/factory"
	"github.com/riotkit-org/backup-maker-operator/pkg/locking"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RequestedBackupActionReconciler reconciles a RequestedBackupAction object
type RequestedBackupActionReconciler struct {
	client.Client
	RestCfg   *rest.Config
	DynClient dynamic.Interface
	BRClient  v1alpha1.RiotkitV1alpha1Interface
	Scheme    *runtime.Scheme
	Cache     cache.Cache
	Fetcher   factory.CachedFetcher
	Recorder  record.EventRecorder
	Locker    locking.Locker
}

func (r *RequestedBackupActionReconciler) fetchAggregate(ctx context.Context, logger *logrus.Entry, req ctrl.Request) (*domain.RequestedBackupActionAggregate, ctrl.Result, error) {
	return factory.FetchRBAAggregate(ctx, r.Fetcher, r.Client, logger, req)
}

// +kubebuilder:rbac:groups=riotkit.org,resources=requestedbackupactions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=riotkit.org,resources=requestedbackupactions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=riotkit.org,resources=requestedbackupactions/finalizers,verbs=update

// Reconcile main loop for RequestedBackupAction controller
func (r *RequestedBackupActionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := createLogger(ctx, req, "RequestedBackupActionReconciler")
	logger.Debugf("Processing %s/%s", req.Name, req.Namespace)

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
	// 1. Fetch all required objects
	//
	aggregate, ctrlResult, err := r.fetchAggregate(ctx, logger, req)
	if err != nil {
		return ctrlResult, err
	}

	logger.Debugf("Fetched aggregate .status = %v, .resourceVersion = %v",
		aggregate.RequestedBackupAction.Status,
		aggregate.RequestedBackupAction.ResourceVersion)

	if aggregate.WasAlreadyProcessed() {
		r.Recorder.Event(aggregate.RequestedBackupAction, "Normal", "Unchanged", "Successfully reconciled, action already performed, cannot do it once more")
		return ctrl.Result{}, nil
	}

	//
	// 2. Template & Create selected resources (only `kind: Job` type resources. The rest like Secrets and ConfigMaps we expect will be there already, created by ScheduledBackup)
	//
	if applyErr := bmg.ApplyScheduledBackup(ctx, logger, r.Recorder, r.RestCfg, r.DynClient, aggregate); applyErr != nil {
		r.updateObjectStatus(ctx, logger, aggregate, metav1.Condition{
			Status:  "False",
			Message: fmt.Sprintf("Cannot find required dependencies: %s", applyErr.Error()),
		})
		r.Recorder.Event(aggregate.RequestedBackupAction, "Warning", "ErrorOccurred", applyErr.Error())
		return ctrl.Result{RequeueAfter: time.Second * 30}, applyErr
	}

	//
	// 3. Update - mark as processed, update status and send notification event
	//
	logger.Debug("Marking resource as processed")
	aggregate.MarkAsProcessed()

	r.updateObjectStatus(ctx, logger, aggregate, metav1.Condition{
		Status:  "True",
		Message: "Successfully templated and applied",
	})
	r.Recorder.Event(aggregate.RequestedBackupAction, "Normal", "Updated", "Successfully reconciled")
	return ctrl.Result{}, nil
}

// updateObjectStatus is updating the .status field
func (r *RequestedBackupActionReconciler) updateObjectStatus(ctx context.Context, logger *logrus.Entry, aggregate *domain.RequestedBackupActionAggregate, condition metav1.Condition) {
	updateErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Fetch a fresh object to avoid: "the object has been modified; please apply your changes to the latest version and try again"
		res, getErr := r.BRClient.RequestedBackupActions(aggregate.Namespace).Get(ctx, aggregate.Name, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		res.Status = aggregate.RequestedBackupAction.Status

		// Update condition
		condition.Reason = "SpecWasUpdated"
		condition.Type = "BackupObjectsInstallation"
		condition.ObservedGeneration = res.Generation
		meta.SetStatusCondition(&res.Status.Conditions, condition)
		logger.Debugf("Setting condition: %v", condition)

		// Update main status
		if strings.ToLower(string(condition.Status)) != "true" {
			res.Status.Healthy = false
		}

		// Update object's status field
		logger.Debugf("Saving .status = %v", res.Status)
		_, updateErr := r.BRClient.RequestedBackupActions(aggregate.Namespace).UpdateStatus(ctx, res, metav1.UpdateOptions{})
		logger.Debugf("Status field updated")
		return updateErr
	})
	if updateErr != nil {
		r.Recorder.Event(aggregate.RequestedBackupAction, "Warning", "ErrorOccurred", fmt.Sprintf("Cannot update .status field: %s", updateErr.Error()))
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *RequestedBackupActionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&riotkitorgv1alpha1.RequestedBackupAction{}).
		WithEventFilter(predicate.Funcs{
			DeleteFunc: func(e event.DeleteEvent) bool {
				return false
			},
		}).
		Complete(r)
}
