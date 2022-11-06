package bmg

import (
	"context"
	"github.com/pkg/errors"
	"github.com/riotkit-org/backup-maker-operator/pkg/aggregates"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
)

// ApplyScheduledBackup is applying objects to the cluster, while adding necessary metadata
func ApplyScheduledBackup(ctx context.Context, recorder record.EventRecorder, restCfg *rest.Config, dynClient dynamic.Interface, backup aggregates.Renderable) error {
	rendered, renderErr := RenderKubernetesResourcesFor(backup)
	if renderErr != nil {
		return errors.Wrap(renderErr, "cannot apply rendered objects to the cluster")
	}

	// add owner references and namespaces to all objects that this controller creates
	for _, doc := range rendered {
		addOwnerReferences(&doc, backup)
		addNamespace(&doc, backup.GetScheduledBackup().Namespace)
	}

	if len(rendered) == 0 {
		return errors.Errorf("no objects rendered, no objects to apply. Please check your template - it did not render any of those objects: %v", backup.AcceptedResourceTypes())
	}
	for _, doc := range rendered {
		apiVersion, kind := doc.GroupVersionKind().ToAPIVersionAndKind()
		logrus.Infof("Applying %s, kind: %s, %s/%s", apiVersion, kind, doc.GetNamespace(), doc.GetName())
		if err := CreateOrUpdate(ctx, recorder, dynClient, restCfg, &doc, backup.GetScheduledBackup()); err != nil {
			return errors.Wrap(err, "cannot apply manifest to the cluster")
		}
	}

	return nil
}

func addNamespace(doc *unstructured.Unstructured, namespace string) {
	doc.SetNamespace(namespace)
}

func addOwnerReferences(doc *unstructured.Unstructured, backup aggregates.Renderable) {
	if _, exists := doc.Object["metadata"]; !exists {
		doc.Object["metadata"] = make(map[string]interface{}, 32)
	}

	owner := backup.GetObjectForOwnerReference()
	metadata := doc.Object["metadata"].(map[string]interface{})
	metadata["ownerReferences"] = []map[string]interface{}{
		{
			"apiVersion": owner.GetTypeMeta().APIVersion,
			"kind":       owner.GetTypeMeta().Kind,
			"name":       owner.GetObjectMeta().Name,
			"uid":        owner.GetObjectMeta().UID,
		},
	}
	logrus.Debugf("Attaching ownerReferences = %v", metadata["ownerReferences"])
}
