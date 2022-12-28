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
	riotkitorgv1alpha1 "github.com/riotkit-org/backup-maker-controller/pkg/apis/riotkit/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/cache"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterBackupProcedureTemplateReconciler reconciles a ClusterBackupProcedureTemplate object
type ClusterBackupProcedureTemplateReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Cache  cache.Cache
}

// +kubebuilder:rbac:groups=riotkit.org,resources=clusterbackupproceduretemplates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=riotkit.org,resources=clusterbackupproceduretemplates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=riotkit.org,resources=clusterbackupproceduretemplates/finalizers,verbs=update

func (r *ClusterBackupProcedureTemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// do nothing, just keep templates cached
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterBackupProcedureTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&riotkitorgv1alpha1.ClusterBackupProcedureTemplate{}).
		Complete(r)
}
