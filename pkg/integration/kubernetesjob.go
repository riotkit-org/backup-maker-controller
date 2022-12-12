package integration

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/riotkit-org/backup-maker-operator/pkg/apis/riotkit/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	v1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	"k8s.io/client-go/rest"
)

type KubernetesJobResourceType struct {
	client *v1.BatchV1Client
}

func (kj KubernetesJobResourceType) MatchesGVK(gvk schema.GroupVersionKind) bool {
	comparison := schema.GroupVersionKind{
		Group:   "batch",
		Version: "v1",
		Kind:    "Job",
	}
	return gvk.String() == comparison.String()
}

func (kj KubernetesJobResourceType) GetScheduledJobHealthStatus(ctx context.Context, trackingId string, namespace string) (v1alpha1.JobHealthStatus, error) {
	list, err := kj.client.Jobs(namespace).List(ctx, metav1.ListOptions{LabelSelector: v1alpha1.LabelTrackingId + "=" + trackingId})
	if err != nil {
		// shit happens
		return v1alpha1.JobHealthStatus{}, errors.Wrap(err, "cannot list integration")
	}

	var running = false
	if len(list.Items) == 0 {
		return v1alpha1.JobHealthStatus{}, errors.New(fmt.Sprintf("cannot find any job labelled with %s=%s", v1alpha1.LabelTrackingId, trackingId))
	}

	// iterate over all labelled jobs - we mostly expect a one object there
	// but in case, when Backup Repository Client would produce more objects we are prepared for it
	for _, job := range list.Items {
		if job.Status.Failed > 0 {
			// if at least one job fails, then our workflow has failed and needs to be repeated
			return v1alpha1.JobHealthStatus{
				ChildReference: v1alpha1.ChildReference{
					APIVersion: job.APIVersion,
					Kind:       job.Kind,
					TrackingId: job.Labels[v1alpha1.LabelTrackingId],
				},
				Message:   fmt.Sprintf("Job %s/%s failed", job.GetNamespace(), job.GetName()),
				Failed:    true,
				Succeeded: false,
				Running:   false,
			}, nil
		}

		// count active or pending jobs
		if job.Status.Active > 0 {
			running = true
		}
	}

	// return one status for all matched jobs
	return v1alpha1.JobHealthStatus{
		ChildReference: v1alpha1.ChildReference{
			APIVersion: "v1",
			Kind:       "Job",
			TrackingId: trackingId,
		},
		Message:   fmt.Sprintf("All labelled jobs by %s=%s succeed in %s namespace", v1alpha1.LabelTrackingId, trackingId, namespace),
		Failed:    false,
		Succeeded: !running,
		Running:   running,
	}, nil
}

// NewKubernetesJobResourceType is creating an instance of a `Kind: Job` checker
func NewKubernetesJobResourceType(cfg *rest.Config) KubernetesJobResourceType {
	batchClient, clErr := v1.NewForConfig(cfg)
	if clErr != nil {
		panic(clErr.Error())
	}
	return KubernetesJobResourceType{
		batchClient,
	}
}
