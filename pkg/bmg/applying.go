package bmg

import (
	"context"
	"github.com/pkg/errors"
	"github.com/riotkit-org/backup-maker-controller/pkg/apis/riotkit/v1alpha1"
	"github.com/riotkit-org/backup-maker-controller/pkg/domain"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
)

// ApplyObjects is applying objects to the cluster, while adding necessary metadata
func ApplyObjects(ctx context.Context, logger *logrus.Entry, recorder record.EventRecorder, restCfg *rest.Config, dynClient dynamic.Interface, backup domain.Renderable) error {
	rendered, renderErr := RenderKubernetesResourcesFor(logger, backup)
	if renderErr != nil {
		logger.Errorln(renderErr)
		return errors.Wrap(renderErr, "cannot apply rendered objects to the cluster")
	}

	// add owner references and namespaces to all objects that this controller creates
	for _, doc := range rendered {
		addOwnerReferences(logger, &doc, backup)
		addNamespace(&doc, backup.GetScheduledBackup().Namespace)
	}

	if len(rendered) == 0 {
		return errors.Errorf("no objects rendered, no objects to apply. Please check your template - it did not render any of those objects: %v", backup.AcceptedResourceTypes())
	}
	for _, doc := range rendered {
		apiVersion, kind := doc.GroupVersionKind().ToAPIVersionAndKind()
		logger.Infof("Applying %s, kind: %s, %s/%s", apiVersion, kind, doc.GetNamespace(), doc.GetName())

		// mark a resource with a unique identifier in the label
		v1alpha1.AppendJobIdTo(&doc)

		if err := CreateOrUpdate(ctx, recorder, dynClient, restCfg, &doc, backup.GetScheduledBackup()); err != nil {
			return errors.Wrap(err, "cannot apply manifest to the cluster")
		}
		addChildReferences(&doc, backup)
	}

	return nil
}

func addNamespace(doc *unstructured.Unstructured, namespace string) {
	doc.SetNamespace(namespace)
}

func addOwnerReferences(logger *logrus.Entry, doc *unstructured.Unstructured, backup domain.Renderable) {
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
	logger.Debugf("Attaching ownerReferences = %v", metadata["ownerReferences"])
}

func addChildReferences(doc *unstructured.Unstructured, backup domain.Renderable) {
	// add that resource to the ChildrenReferences field for the parent, so the parent
	// could find all of its resources using a previously generated set of labels
	backup.AddOwnedObject(doc)
}
