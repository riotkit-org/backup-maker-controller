package integration

import (
	"context"
	"github.com/pkg/errors"
	"github.com/riotkit-org/backup-maker-controller/pkg/apis/riotkit/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

const ErrorUnrecognizedResourceType = "unrecognized resource type"

type SupportedJobResourceType interface {
	MatchesGVK(gvk schema.GroupVersionKind) bool
	GetScheduledJobHealthStatus(ctx context.Context, trackingId string, namespace string) (v1alpha1.JobHealthStatus, error)
}

// AllSupportedJobResourceTypes is a bridge/interface to all adapters implementing various job types
// instead of asking multiple adapters you ask a single service that aggregates all of them
type AllSupportedJobResourceTypes struct {
	cfg       *rest.Config
	supported []SupportedJobResourceType
}

func (all *AllSupportedJobResourceTypes) GetScheduledJobHealthStatus(ctx context.Context, gvk schema.GroupVersionKind, trackingId string, namespace string) (v1alpha1.JobHealthStatus, error) {
	for _, resource := range all.supported {
		if resource.MatchesGVK(gvk) {
			return resource.GetScheduledJobHealthStatus(ctx, trackingId, namespace)
		}
	}
	return v1alpha1.JobHealthStatus{}, errors.New(ErrorUnrecognizedResourceType)
}

func NewAllSupportedJobResourceTypes(cfg *rest.Config) AllSupportedJobResourceTypes {
	return AllSupportedJobResourceTypes{
		cfg: cfg,
		supported: []SupportedJobResourceType{
			NewKubernetesJobResourceType(cfg),
		},
	}
}
