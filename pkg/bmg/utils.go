package bmg

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/record"
	"strings"
)

func CreateOrUpdate(
	ctx context.Context,
	recorder record.EventRecorder,
	dyn dynamic.Interface,
	restCfg *rest.Config,
	obj *unstructured.Unstructured,
	backup runtime.Object, // todo: change to runtime.Object
) error {
	dc, err := discovery.NewDiscoveryClientForConfig(restCfg)
	if err != nil {
		return errors.Wrap(err, "cannot create Discovery Client for checking installed api resources")
	}

	// construct dynamic client mapped to proper ApiVersion, Group, Kind
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))
	mapping, err := mapper.RESTMapping(obj.GroupVersionKind().GroupKind(), obj.GroupVersionKind().Version)
	c := dyn.Resource(mapping.Resource).Namespace(obj.GetNamespace())

	apiVersion, kind := obj.GroupVersionKind().ToAPIVersionAndKind()

	// try to fetch resource
	if _, getErr := c.Get(ctx, obj.GetName(), v1.GetOptions{}); getErr != nil {
		if !apierrors.IsNotFound(getErr) {
			return errors.Wrap(getErr, "cannot fetch object information from API server to check if it exists")
		}

		// if not found, then CREATE
		if _, createErr := c.Create(ctx, obj, v1.CreateOptions{}); createErr != nil {
			return errors.Wrap(createErr, "cannot create object in API")
		}
		recorder.Event(backup, "Normal", "Created", fmt.Sprintf("Creating %s/%s, named %s/%s", apiVersion, kind, obj.GetNamespace(), obj.GetName()))
		return nil
	}

	// else - if found, UPDATE
	if _, updateErr := c.Update(ctx, obj, v1.UpdateOptions{}); updateErr != nil {
		if strings.Contains(updateErr.Error(), "is immutable") {
			delErr := c.Delete(ctx, obj.GetName(), v1.DeleteOptions{})
			if delErr != nil {
				return errors.Wrap(delErr, "cannot delete object for recreation on immutability conflict")
			}
			// retry
			if _, updateErr := c.Create(ctx, obj, v1.CreateOptions{}); updateErr != nil {
				return errors.Wrap(updateErr, "cannot update object in API (retried)")
			}
		} else {
			return errors.Wrap(updateErr, "cannot update object in API")
		}
	}
	recorder.Event(backup, "Normal", "Updated", fmt.Sprintf("Updating %s/%s, named %s/%s", apiVersion, kind, obj.GetNamespace(), obj.GetName()))
	return nil
}
