package domain

//
// ResourceTypes filtering is deciding which "kinds" of resources will be applied by which
// CRD - "ScheduledBackup" or "RequestedBackupAction"
//

import v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type ResourceTypes struct {
	gvk []v1.GroupVersionKind
}

func (rt *ResourceTypes) GetKinds() []v1.GroupVersionKind {
	return rt.gvk
}

// NewResourceTypesFilterForScheduledBackup decides that ScheduledBackup owns all helper objects like Secrets, ConfigMaps for all kinds of actions
// for example app1-backup, app1-restore configmaps would be created at once
//
// IMPORTANT: There are no Jobs maintained by ScheduledBackup CRDs
//
//	The runnable Jobs are maintained by RequestedBackupAction
//
// The ScheduledBackup is optionally managing CronJobs, which are spawning Jobs according to how Kubernetes works by default
func NewResourceTypesFilterForScheduledBackup(resource Renderable, op Operation) ResourceTypes {
	// Helper kinds
	gvk := []v1.GroupVersionKind{
		{Group: "", Version: "v1", Kind: "Secret"},
		{Group: "", Version: "v1", Kind: "ConfigMap"},
		{Group: "bitnami.com", Version: "v1alpha1", Kind: "SealedSecret"},
		{Group: "kubernetes-client.io", Version: "v1", Kind: "ExternalSecret"},
	}

	// Case: Create target CronJob for desired action for that ScheduledBackup
	if resource.GetOperation() == op && resource.ShouldCreateCronJob() {
		gvk = append(gvk, v1.GroupVersionKind{
			Group: "batch", Version: "v1", Kind: "CronJob",
		})
	}

	return ResourceTypes{
		gvk: gvk,
	}
}

// NewResourceTypesFilterForRequestedBackupAction decides that RequestedBackupAction is owning only runnable objects (JOBS)
func NewResourceTypesFilterForRequestedBackupAction() ResourceTypes {
	return ResourceTypes{
		gvk: []v1.GroupVersionKind{
			{Group: "batch", Version: "v1", Kind: "Job"},
		},
	}
}
