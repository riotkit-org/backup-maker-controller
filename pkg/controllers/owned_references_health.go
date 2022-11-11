package controllers

import (
	"context"
	riotkitorgv1alpha1 "github.com/riotkit-org/backup-maker-operator/pkg/apis/riotkit/v1alpha1"
	"github.com/riotkit-org/backup-maker-operator/pkg/integration"
	"github.com/sirupsen/logrus"
)

// createOwnedReferencesHealthReport Check health status of all managed backup jobs and put into a single report list
func createOwnedReferencesHealthReport(ctx context.Context, ownedReferences riotkitorgv1alpha1.ChildrenReferences, integrations *integration.AllSupportedJobResourceTypes, namespace string) ([]riotkitorgv1alpha1.JobHealthStatus, bool, error) {
	report := make([]riotkitorgv1alpha1.JobHealthStatus, 0)
	var healthy = true

	// go through all children jobs
	for _, resource := range ownedReferences {

		// get a health status
		status, err := integrations.GetScheduledJobHealthStatus(ctx, resource.GetGVK(), resource.TrackingId, namespace)
		logrus.Debugf("JobStatus = %v, err = %v", status, err)
		if err != nil {
			if err.Error() == integration.ErrorUnrecognizedResourceType {
				continue
			}
			return report, false, err
		}

		// fill up the report with a health status
		report = append(report, status)

		// this allows to set the health status globally in the .status.healthy
		if status.Failed {
			healthy = false
		}
	}
	return report, healthy, nil
}
