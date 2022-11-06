package factory

import (
	"context"
	riotkitorgv1alpha1 "github.com/riotkit-org/backup-maker-operator/pkg/apis/riotkit/v1alpha1"
	"github.com/riotkit-org/backup-maker-operator/pkg/client/clientset/versioned/typed/riotkit/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CachedFetcher is fetching objects collected by the controller-runtime
type CachedFetcher struct {
	Cache  cache.Cache
	Client v1alpha1.RiotkitV1alpha1Interface
}

// FetchScheduledBackup is fetching ScheduledBackup object from the cache
func (r *CachedFetcher) FetchScheduledBackup(ctx context.Context, req ctrl.Request) (*riotkitorgv1alpha1.ScheduledBackup, error) {
	backup := riotkitorgv1alpha1.ScheduledBackup{}
	getErr := r.Cache.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, &backup)
	return &backup, getErr
}

// FetchRequestedBackupAction is fetching ScheduledBackup object from the cache
func (r *CachedFetcher) FetchRequestedBackupAction(ctx context.Context, req ctrl.Request) (*riotkitorgv1alpha1.RequestedBackupAction, error) {
	backup, getErr := r.Client.RequestedBackupActions(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	return backup, getErr
}

// fetchTemplate is fetching a template from cache
func (r *CachedFetcher) fetchTemplate(ctx context.Context, backup *riotkitorgv1alpha1.ScheduledBackup) (*riotkitorgv1alpha1.ClusterBackupProcedureTemplate, error) {
	template := riotkitorgv1alpha1.ClusterBackupProcedureTemplate{}
	getErr := r.Cache.Get(ctx, client.ObjectKey{Name: backup.Spec.TemplateRef.Name}, &template)
	return &template, getErr
}

func (r *CachedFetcher) fetchSecret(ctx context.Context, name string, namespace string) (*v1.Secret, error) {
	secret := v1.Secret{}
	getErr := r.Cache.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, &secret)
	return &secret, getErr
}
