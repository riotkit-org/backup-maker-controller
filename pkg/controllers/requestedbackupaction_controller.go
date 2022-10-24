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
	"github.com/go-logr/logr"
	"github.com/riotkit-org/backup-maker-operator/pkg/aggregates"
	riotkitorgv1alpha1 "github.com/riotkit-org/backup-maker-operator/pkg/apis/riotkit/v1alpha1"
	"github.com/riotkit-org/backup-maker-operator/pkg/bmg"
	"github.com/riotkit-org/backup-maker-operator/pkg/client/clientset/versioned/typed/riotkit/v1alpha1"
	"github.com/riotkit-org/backup-maker-operator/pkg/factory"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
}

func (r *RequestedBackupActionReconciler) fetchAggregate(ctx context.Context, logger logr.Logger, req ctrl.Request) (*aggregates.RequestedBackupActionAggregate, ctrl.Result, error) {
	requestedAction, err := r.Fetcher.FetchRequestedBackupAction(ctx, req)
	if err != nil {
		return &aggregates.RequestedBackupActionAggregate{}, ctrl.Result{RequeueAfter: time.Second * 30}, err
	}
	// todo: immutable - do not process twice
	scheduledBackup, err := r.Fetcher.FetchScheduledBackup(ctx, ctrl.Request{NamespacedName: types.NamespacedName{
		Name:      requestedAction.Spec.ScheduledBackupRef.Name,
		Namespace: requestedAction.Namespace,
	}})
	if err != nil {
		return &aggregates.RequestedBackupActionAggregate{}, ctrl.Result{RequeueAfter: time.Second * 30}, err
	}
	f := factory.NewFactory(r.Client, r.Fetcher, logger)
	aggregate, _, hydrateErr := f.CreateRequestedBackupActionAggregate(
		ctx, requestedAction, scheduledBackup,
	)
	if hydrateErr != nil {
		return &aggregates.RequestedBackupActionAggregate{}, ctrl.Result{RequeueAfter: time.Second * 30}, err
	}
	return aggregate, ctrl.Result{}, nil
}

// +kubebuilder:rbac:groups=riotkit.org,resources=restoredbackups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=riotkit.org,resources=restoredbackups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=riotkit.org,resources=restoredbackups/finalizers,verbs=update

// Reconcile main loop for RequestedBackupAction controller
func (r *RequestedBackupActionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 1. Fetch all required objects
	aggregate, ctrlResult, err := r.fetchAggregate(ctx, logger, req)
	if err != nil {
		return ctrlResult, err
	}

	if aggregate.WasAlreadyProcessed() {
		// todo: emit event
		return ctrl.Result{}, nil
	}

	// 2. Template & Create selected resources (only `kind: Job` type resources. The rest like Secrets and ConfigMaps we expect will be there already, created by ScheduledBackup)
	if applyErr := bmg.ApplyScheduledBackup(ctx, r.Recorder, r.RestCfg, r.DynClient, aggregate); applyErr != nil {
		r.updateObject(ctx, aggregate, metav1.Condition{
			Status:  "False",
			Message: fmt.Sprintf("Cannot find required dependencies: %s", applyErr.Error()),
		})
		r.Recorder.Event(aggregate.RequestedBackupAction, "Warning", "ErrorOccurred", applyErr.Error())
		return ctrl.Result{RequeueAfter: time.Second * 15}, applyErr
	}

	// 3. Update - mark as processed, update status and send notification event
	aggregate.MarkAsProcessed()
	r.updateObject(ctx, aggregate, metav1.Condition{
		Status:  "True",
		Message: "Successfully templated and applied",
	})
	r.Recorder.Event(aggregate.RequestedBackupAction, "Normal", "Updated", "Successfully reconciled")
	return ctrl.Result{}, nil
}

// updateObject is updating the .status field
func (r *RequestedBackupActionReconciler) updateObject(ctx context.Context, aggregate *aggregates.RequestedBackupActionAggregate, condition metav1.Condition) {
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
		meta.SetStatusCondition(&res.Status.Conditions, condition)

		// Update object's status field
		_, updateErr := r.BRClient.ScheduledBackups(aggregate.Namespace).UpdateStatus(ctx, res, metav1.UpdateOptions{})
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
		Complete(r)
}
